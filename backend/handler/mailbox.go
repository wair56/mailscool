package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"mailer/config"
	"mailer/database"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ===== 统一域名权限校验 =====

// hasDomainAccess 检查当前管理员是否有指定域名的访问权限
func hasDomainAccess(c *gin.Context, domainID int64) bool {
	role, _ := c.Get("role")
	if role == "super_admin" {
		return true
	}
	adminID, _ := c.Get("admin_id")
	var count int
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM admin_domains WHERE admin_id = ? AND domain_id = ?",
		adminID, domainID,
	).Scan(&count)
	return count > 0
}

// ===== Admin 邮箱管理 =====

type MailboxItem struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	PasswordPlain string     `json:"password_plain"`
	DomainID      int64      `json:"domain_id"`
	DomainName    string     `json:"domain_name"`
	TotalEmails   int64      `json:"total_emails"`
	IsTemp        bool       `json:"is_temp"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CreatedIP     string     `json:"created_ip,omitempty"`
	CreatedUA     string     `json:"created_ua,omitempty"`
	WebhookURL    string     `json:"webhook_url,omitempty"`
}

func AdminListMailboxes(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	var whereClause string
	var args []interface{}

	if role == "super_admin" {
		// 超管看所有
		whereClause = ""
	} else {
		// 非超管：仅看自己域名下的邮箱
		whereClause = "WHERE m.domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)"
		args = append(args, adminID)
	}

	rows, err := database.DB.Query(`
		SELECT m.id, m.email, COALESCE(m.password_plain,''), m.domain_id, COALESCE(d.name,''),
		       COALESCE(m.is_temp,0), m.expires_at, m.created_at,
		       (SELECT COUNT(*) FROM emails e WHERE e.recipient = m.email),
		       COALESCE(m.created_ip,''), COALESCE(m.created_ua,''), COALESCE(m.webhook_url,'')
		FROM mailboxes m
		LEFT JOIN domains d ON d.id = m.domain_id
		`+whereClause+`
		ORDER BY m.created_at DESC
	`, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []MailboxItem
	for rows.Next() {
		var m MailboxItem
		var isTemp int
		var expiresAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.Email, &m.PasswordPlain, &m.DomainID, &m.DomainName, &isTemp, &expiresAt, &m.CreatedAt, &m.TotalEmails, &m.CreatedIP, &m.CreatedUA, &m.WebhookURL); err != nil {
			continue
		}
		m.IsTemp = isTemp == 1
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		// 解密密码明文
		if m.PasswordPlain != "" {
			if decrypted, err := config.Decrypt(m.PasswordPlain); err == nil {
				m.PasswordPlain = decrypted
			}
		}
		items = append(items, m)
	}
	if items == nil {
		items = []MailboxItem{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

type CreateMailboxRequest struct {
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required"`
	WebhookURL string `json:"webhook_url"`
}

func AdminCreateMailbox(c *gin.Context) {
	var req CreateMailboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	// 提取域名
	parts := strings.Split(req.Email, "@")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}
	domain := parts[1]

	// 查域名 ID
	var domainID int64
	err := database.DB.QueryRow("SELECT id FROM domains WHERE name = ?", domain).Scan(&domainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain not found: " + domain})
		return
	}

	// 校验域名权限
	if !hasDomainAccess(c, domainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此域名的邮箱"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	encryptedPwd, _ := config.Encrypt(req.Password)
	_, err = database.DB.Exec(
		"INSERT INTO mailboxes (email, password_plain, password_hash, domain_id, webhook_url) VALUES (?, ?, ?, ?, ?)",
		req.Email, encryptedPwd, string(hash), domainID, req.WebhookURL,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "mailbox already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func AdminDeleteMailbox(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 查邮箱信息
	var domainID int64
	var email string
	err := database.DB.QueryRow("SELECT domain_id, email FROM mailboxes WHERE id = ?", id).Scan(&domainID, &email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mailbox not found"})
		return
	}
	if !hasDomainAccess(c, domainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此邮箱"})
		return
	}

	// 是否删除关联邮件
	deletedEmails := int64(0)
	if c.Query("delete_emails") == "true" {
		result, delErr := database.DB.Exec(
			"DELETE FROM emails WHERE recipient = ?",
			email,
		)
		if delErr != nil {
			log.Printf("[warn] failed to delete emails for %s: %v", email, delErr)
		}
		if result != nil {
			deletedEmails, _ = result.RowsAffected()
		}
	}

	if _, err := database.DB.Exec("DELETE FROM mailboxes WHERE id = ?", id); err != nil {
		log.Printf("[warn] failed to delete mailbox %d: %v", id, err)
	}

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "delete_mailbox", email, fmt.Sprintf("deleted_emails: %d", deletedEmails))

	c.JSON(http.StatusOK, gin.H{"message": "deleted", "deleted_emails": deletedEmails})
}

type UpdateMailboxRequest struct {
	IsTemp     *bool   `json:"is_temp"`
	ExpiresAt  *string `json:"expires_at"`
	WebhookURL *string `json:"webhook_url"`
}

func AdminUpdateMailbox(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpdateMailboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 校验域名权限
	var domainID int64
	if err := database.DB.QueryRow("SELECT domain_id FROM mailboxes WHERE id = ?", id).Scan(&domainID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mailbox not found"})
		return
	}
	if !hasDomainAccess(c, domainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此邮箱"})
		return
	}

	if req.IsTemp != nil {
		isTemp := 0
		if *req.IsTemp {
			isTemp = 1
		}
		database.DB.Exec("UPDATE mailboxes SET is_temp = ? WHERE id = ?", isTemp, id)
	}

	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			database.DB.Exec("UPDATE mailboxes SET expires_at = NULL WHERE id = ?", id)
		} else {
			t, err := time.Parse("2006-01-02T15:04:05Z07:00", *req.ExpiresAt)
			if err != nil {
				t, err = time.Parse("2006-01-02", *req.ExpiresAt)
			}
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format"})
				return
			}
			database.DB.Exec("UPDATE mailboxes SET expires_at = ? WHERE id = ?", t, id)
		}
	}

	if req.WebhookURL != nil {
		database.DB.Exec("UPDATE mailboxes SET webhook_url = ? WHERE id = ?", *req.WebhookURL, id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// ===== 邮箱用户登录 =====

type MailboxLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func MailboxLogin(c *gin.Context) {
	var req MailboxLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	clientIP := c.ClientIP()

	// 检查 IP 锁定（复用 admin 登录锁定机制）
	if CheckLoginLock(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
		return
	}

	var id int64
	var passwordHash string
	var isTemp int
	var expiresAt sql.NullTime
	err := database.DB.QueryRow("SELECT id, password_hash, COALESCE(is_temp,0), expires_at FROM mailboxes WHERE email = ?", req.Email).Scan(&id, &passwordHash, &isTemp, &expiresAt)
	if err != nil {
		RecordLoginFailure(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 检查是否过期
	if isTemp == 1 && expiresAt.Valid && time.Now().After(expiresAt.Time) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "mailbox expired"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		RecordLoginFailure(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 清除登录失败记录
	ClearLoginFailure(clientIP)

	// 生成 JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"mailbox_id": id,
		"email":      req.Email,
		"type":       "mailbox",
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString(getJWTSecret())

	c.JSON(http.StatusOK, gin.H{"token": tokenStr, "email": req.Email})
}

// getJWTSecret 运行时读取 JWT 密钥，避免包初始化竞态
func getJWTSecret() []byte {
	return []byte(config.C.JWTSecret)
}

// ===== 真人风格邮箱名生成 =====

var firstNames = []string{
	// English - Male
	"james", "john", "david", "michael", "robert", "william", "richard", "joseph",
	"thomas", "daniel", "matthew", "anthony", "mark", "steven", "paul", "andrew",
	"kevin", "brian", "alex", "ryan", "eric", "jason", "adam", "nathan", "tyler",
	"samuel", "benjamin", "lucas", "henry", "owen", "jack", "leo", "dylan", "luke",
	"gabriel", "connor", "evan", "noah", "ethan", "logan", "mason", "liam", "caleb",
	"travis", "marcus", "derek", "victor", "carlos", "frank", "peter", "raymond",
	"sean", "joel", "keith", "patrick", "dennis", "jerry", "brandon", "philip",
	"russell", "craig", "scott", "jesse", "todd", "lance", "terry", "barry",
	"carl", "roger", "bruce", "wayne", "dale", "troy", "brett", "chad", "kirk",
	// English - Female
	"mary", "sarah", "emma", "olivia", "sophia", "isabella", "mia", "charlotte",
	"amelia", "harper", "evelyn", "abigail", "emily", "elizabeth", "ella", "grace",
	"chloe", "victoria", "lily", "hannah", "nora", "riley", "zoey", "stella",
	"lucy", "aurora", "hazel", "violet", "penelope", "layla", "ellie", "maya",
	"isla", "willow", "ivy", "alice", "elena", "clara", "ruby", "vivian",
	"naomi", "diana", "julia", "rachel", "monica", "tiffany", "nicole", "jessica",
	"ashley", "amber", "melissa", "laura", "kimberly", "heather", "andrea", "megan",
	"crystal", "brittany", "lindsey", "chelsea", "holly", "brooke", "tabitha", "paige",
	"lena", "vera", "nina", "iris", "fiona", "gwen", "tessa", "skye", "blair",
	// International
	"yuki", "hana", "kenji", "ryo", "akira", "sora", "rin", "kai", "mei", "lin",
	"wei", "jing", "ming", "xin", "yan", "zhen", "arjun", "priya", "ravi", "anita",
	"omar", "leila", "ali", "fatima", "nadia", "sami", "lara", "dario", "marco",
	"elena", "sofia", "luca", "matteo", "leon", "felix", "milo", "ava", "elsa",
	"freya", "astrid", "erik", "sven", "lars", "nico", "hugo", "jules", "remi",
	"andre", "pierre", "claude", "hans", "max", "otto", "rosa", "carmen", "pablo",
}

var lastNames = []string{
	// English common
	"smith", "johnson", "williams", "brown", "jones", "garcia", "miller", "davis",
	"martinez", "anderson", "taylor", "thomas", "jackson", "white", "harris",
	"martin", "thompson", "moore", "young", "allen", "king", "wright", "scott",
	"hill", "green", "adams", "baker", "nelson", "carter", "mitchell", "turner",
	"phillips", "campbell", "parker", "evans", "edwards", "collins", "stewart",
	"morris", "rogers", "reed", "cook", "morgan", "bell", "murphy", "bailey",
	"rivera", "cooper", "cox", "howard", "ward", "torres", "peterson", "gray",
	"woods", "barnes", "ross", "henderson", "coleman", "jenkins", "perry", "powell",
	"long", "patterson", "hughes", "flores", "washington", "butler", "simmons",
	"foster", "gonzales", "bryant", "russell", "griffin", "hayes", "hudson",
	"marshall", "owens", "webb", "ford", "newman", "wallace", "brooks", "cole",
	"west", "jordan", "reynolds", "fisher", "ellis", "stone", "spencer", "fox",
	"mason", "hunt", "dean", "black", "burns", "porter", "lane", "grant", "hart",
	"price", "wells", "dunn", "wolf", "snow", "drake", "cross", "dale", "frost",
	// East Asian
	"lee", "chen", "wang", "liu", "zhao", "wu", "kim", "park", "choi", "yang",
	"huang", "zhou", "xu", "sun", "ma", "zhu", "lin", "guo", "he", "luo",
	"tang", "han", "feng", "deng", "cao", "xie", "song", "pan", "yuan", "dong",
	"tanaka", "yamamoto", "suzuki", "watanabe", "sato", "ito", "takahashi", "nakamura",
	// South Asian / Middle Eastern / European
	"singh", "patel", "sharma", "khan", "ali", "kumar", "gupta", "nair", "rao",
	"silva", "santos", "oliveira", "costa", "ferreira", "almeida", "lima", "rocha",
	"schmidt", "weber", "wagner", "becker", "meyer", "richter", "klein", "braun",
	"mueller", "berger", "rosen", "berg", "lund", "borg", "larsson", "nilsson",
	"dubois", "moreau", "leroy", "simon", "blanc", "petit", "vidal", "rossi",
}

// generateRealisticUsername produces names like sarah.chen29, mike_taylor, jw.morgan03
func generateRealisticUsername() string {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	first := firstNames[r.Intn(len(firstNames))]
	last := lastNames[r.Intn(len(lastNames))]
	num := r.Intn(100)
	year := 1985 + r.Intn(25) // 1985-2009
	num3 := 100 + r.Intn(900) // 100-999

	adjectives := []string{"cool", "real", "the", "just", "not", "hey", "go", "my", "new", "top", "pro", "big", "its", "one"}
	nouns := []string{"coder", "dev", "design", "digital", "data", "cloud", "pixel", "byte", "logic", "studio", "lab", "hub", "io", "hq", "ai"}

	// 16 patterns for maximum variety
	pattern := r.Intn(16)
	switch pattern {
	case 0: // sarah.chen29
		return fmt.Sprintf("%s.%s%02d", first, last, num)
	case 1: // sarahchen5
		return fmt.Sprintf("%s%s%d", first, last, r.Intn(10))
	case 2: // sarah_chen
		return fmt.Sprintf("%s_%s", first, last)
	case 3: // s.chen07
		return fmt.Sprintf("%c.%s%02d", first[0], last, num)
	case 4: // sarah.c92
		return fmt.Sprintf("%s.%c%02d", first, last[0], num)
	case 5: // sc.morgan
		return fmt.Sprintf("%c%c.%s", first[0], last[0], lastNames[r.Intn(len(lastNames))])
	case 6: // sarah2003
		return fmt.Sprintf("%s%d", first, year)
	case 7: // sarah_t
		return fmt.Sprintf("%s_%c", first, last[0])
	case 8: // chen.sarah
		return fmt.Sprintf("%s.%s", last, first)
	case 9: // sarah-chen
		return fmt.Sprintf("%s-%s", first, last)
	case 10: // sarah.chen.dev
		return fmt.Sprintf("%s.%s.%s", first, last, nouns[r.Intn(len(nouns))])
	case 11: // sarah123
		return fmt.Sprintf("%s%d", first, num3)
	case 12: // coolsarah / thesarah
		return fmt.Sprintf("%s%s", adjectives[r.Intn(len(adjectives))], first)
	case 13: // sarahdev / sarahcloud
		return fmt.Sprintf("%s%s", first, nouns[r.Intn(len(nouns))])
	case 14: // sc2847 (initials + 4 digits)
		return fmt.Sprintf("%c%c%04d", first[0], last[0], r.Intn(10000))
	default: // sarah.chen1997
		return fmt.Sprintf("%s.%s%d", first, last, year)
	}
}

// ===== 公开注册临时邮箱 =====

func RegisterTempMailbox(c *gin.Context) {
	clientIP := c.ClientIP()

	// IP 每日限制
	perIPLimit := database.GetSettingInt("temp_mailbox_per_ip_daily", 3)
	if perIPLimit > 0 {
		var ipCount int
		today := time.Now().Format("2006-01-02")
		database.DB.QueryRow(
			"SELECT COUNT(*) FROM mailboxes WHERE is_temp = 1 AND created_ip = ? AND DATE(created_at) = ?",
			clientIP, today,
		).Scan(&ipCount)
		if ipCount >= perIPLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Daily limit reached (%d/IP/day) / 已达每日上限（每IP %d个/天）", perIPLimit, perIPLimit),
			})
			return
		}
	}

	// Turnstile 人机验证（仅在配置了 secret_key 时启用）
	turnstileSecret := database.GetSetting("turnstile_secret_key", "")
	origin := c.GetHeader("Origin")
	isExtension := strings.HasPrefix(origin, "chrome-extension://") || strings.HasPrefix(origin, "moz-extension://")
	
	if turnstileSecret != "" && !isExtension {
		turnstileToken := c.Query("turnstile_token")
		if turnstileToken == "" {
			// 也支持从 JSON body 读取
			var body struct{ TurnstileToken string `json:"turnstile_token"` }
			c.ShouldBindJSON(&body)
			turnstileToken = body.TurnstileToken
		}
		if turnstileToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Turnstile verification required / 需要人机验证"})
			return
		}
		if !verifyTurnstile(turnstileSecret, turnstileToken, clientIP) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Turnstile verification failed / 人机验证失败"})
			return
		}
	}

	// 全站每日总量限制
	dailyTotal := database.GetSettingInt("temp_mailbox_daily_total", 0)
	if dailyTotal > 0 {
		var totalCount int
		today := time.Now().Format("2006-01-02")
		database.DB.QueryRow(
			"SELECT COUNT(*) FROM mailboxes WHERE is_temp = 1 AND DATE(created_at) = ?",
			today,
		).Scan(&totalCount)
		if totalCount >= dailyTotal {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Daily total limit reached / 今日创建总量已达上限",
			})
			return
		}
	}

	// 获取域名：优先从 temp_email_domains 中随机选
	var domainID int64
	var domainName string
	configured := database.GetSetting("temp_email_domains", "")
	if configured != "" {
		domains := strings.Split(configured, ",")
		var activeDomains []struct{ id int64; name string }
		for _, d := range domains {
			d = strings.TrimSpace(d)
			var did int64
			var dname string
			if e := database.DB.QueryRow("SELECT id, name FROM domains WHERE name = ? AND is_active = 1", d).Scan(&did, &dname); e == nil {
				activeDomains = append(activeDomains, struct{ id int64; name string }{did, dname})
			}
		}
		if len(activeDomains) > 0 {
			idx := mathrand.Intn(len(activeDomains))
			domainID = activeDomains[idx].id
			domainName = activeDomains[idx].name
		}
	}
	if domainName == "" {
		err := database.DB.QueryRow("SELECT id, name FROM domains WHERE is_active = 1 ORDER BY id LIMIT 1").Scan(&domainID, &domainName)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no active domain"})
			return
		}
	}

	// 随机生成邮箱和密码
	username := generateRealisticUsername()
	email := username + "@" + domainName

	pwdBytes := make([]byte, 6)
	rand.Read(pwdBytes)
	password := hex.EncodeToString(pwdBytes)

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	expiresMonths := database.GetSettingInt("temp_mailbox_expiry_months", 3)
	expiresAt := time.Now().AddDate(0, expiresMonths, 0)

	encryptedPwd, _ := config.Encrypt(password)
	userAgent := c.GetHeader("User-Agent")
	referer := c.GetHeader("Referer")
	_, err := database.DB.Exec(
		"INSERT INTO mailboxes (email, password_plain, password_hash, domain_id, is_temp, expires_at, created_ip, created_ua) VALUES (?, ?, ?, ?, 1, ?, ?, ?)",
		email, encryptedPwd, string(hash), domainID, expiresAt, clientIP, userAgent+"|"+origin+"|"+referer,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mailbox"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":      email,
		"password":   password,
		"expires_at": expiresAt.Format("2006-01-02"),
	})
}

// ===== API Key 鉴权创建临时邮箱 =====

func ApiCreateTempMailbox(c *gin.Context) {
	// 支持可选参数
	domainParam := c.Query("domain")
	usernameParam := c.Query("username")
	passwordParam := c.Query("password")

	var domainID int64
	var domainName string
	var err error

	if domainParam != "" {
		err = database.DB.QueryRow("SELECT id, name FROM domains WHERE name = ? AND is_active = 1", domainParam).Scan(&domainID, &domainName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "domain not found or inactive"})
			return
		}
	} else {
		// Pick from temp_email_domains setting, or fallback to all active domains
		configured := database.GetSetting("temp_email_domains", "")
		if configured != "" {
			domains := strings.Split(configured, ",")
			for i := range domains {
				domains[i] = strings.TrimSpace(domains[i])
			}
			// Filter to only active ones
			var activeDomains []struct{ id int64; name string }
			for _, d := range domains {
				var did int64
				var dname string
				if e := database.DB.QueryRow("SELECT id, name FROM domains WHERE name = ? AND is_active = 1", d).Scan(&did, &dname); e == nil {
					activeDomains = append(activeDomains, struct{ id int64; name string }{did, dname})
				}
			}
			if len(activeDomains) > 0 {
				pick := activeDomains[mathrand.Intn(len(activeDomains))]
				domainID = pick.id
				domainName = pick.name
			} else {
				err = fmt.Errorf("no active domain found")
			}
		} else {
			// Fallback: random from all active
			rows, _ := database.DB.Query("SELECT id, name FROM domains WHERE is_active = 1")
			var all []struct{ id int64; name string }
			if rows != nil {
				defer rows.Close()
				for rows.Next() {
					var did int64
					var dname string
					rows.Scan(&did, &dname)
					all = append(all, struct{ id int64; name string }{did, dname})
				}
			}
			if len(all) > 0 {
				pick := all[mathrand.Intn(len(all))]
				domainID = pick.id
				domainName = pick.name
			} else {
				err = fmt.Errorf("no active domain")
			}
		}
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no active domain"})
		return
	}

	// 检查 API Key 是否有此域名权限
	apiKeyID, exists := c.Get("api_key_id")
	if exists {
		var count int
		database.DB.QueryRow(
			"SELECT COUNT(*) FROM api_key_domains WHERE api_key_id = ?", apiKeyID,
		).Scan(&count)
		if count > 0 {
			var domainCount int
			database.DB.QueryRow(
				"SELECT COUNT(*) FROM api_key_domains WHERE api_key_id = ? AND domain_id = ?",
				apiKeyID, domainID,
			).Scan(&domainCount)
			if domainCount == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "API key not authorized for this domain"})
				return
			}
		}
	}

	username := usernameParam
	if username == "" {
		username = generateRealisticUsername()
	}
	email := username + "@" + domainName

	// 检查邮箱是否已存在
	var existCount int
	database.DB.QueryRow("SELECT COUNT(*) FROM mailboxes WHERE email = ?", email).Scan(&existCount)
	if existCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}

	password := passwordParam
	if password == "" {
		pwdBytes := make([]byte, 6)
		rand.Read(pwdBytes)
		password = hex.EncodeToString(pwdBytes)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	expiresMonths := database.GetSettingInt("temp_mailbox_expiry_months", 3)
	expiresAt := time.Now().AddDate(0, expiresMonths, 0)

	encryptedPwd, _ := config.Encrypt(password)
	
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	origin := c.GetHeader("Origin")
	referer := c.GetHeader("Referer")

	_, err = database.DB.Exec(
		"INSERT INTO mailboxes (email, password_plain, password_hash, domain_id, is_temp, expires_at, created_ip, created_ua) VALUES (?, ?, ?, ?, 1, ?, ?, ?)",
		email, encryptedPwd, string(hash), domainID, expiresAt, clientIP, userAgent+"|"+origin+"|"+referer,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mailbox"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":      email,
		"password":   password,
		"domain":     domainName,
		"expires_at": expiresAt.Format("2006-01-02"),
	})
}

// ===== 邮箱用户信息 =====

func MailboxGetMe(c *gin.Context) {
	email, _ := c.Get("mailbox_email")
	emailStr := email.(string)

	var id int64
	var isTemp int
	var expiresAt *string
	var createdAt string
	err := database.DB.QueryRow(
		"SELECT id, is_temp, expires_at, created_at FROM mailboxes WHERE email = ?", emailStr,
	).Scan(&id, &isTemp, &expiresAt, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mailbox not found"})
		return
	}

	result := gin.H{
		"email":      emailStr,
		"is_temp":    isTemp == 1,
		"created_at": createdAt,
	}
	if expiresAt != nil {
		result["expires_at"] = *expiresAt
	}
	c.JSON(http.StatusOK, result)
}

func MailboxRenew(c *gin.Context) {
	email, _ := c.Get("mailbox_email")
	emailStr := email.(string)

	// Only temp mailboxes can be renewed
	var isTemp int
	var expiresAt *string
	err := database.DB.QueryRow(
		"SELECT COALESCE(is_temp,0), expires_at FROM mailboxes WHERE email = ?", emailStr,
	).Scan(&isTemp, &expiresAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mailbox not found"})
		return
	}
	if isTemp != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only temp mailboxes can be renewed"})
		return
	}

	// Only allow renewal within 10 days before expiry
	if expiresAt != nil {
		expTime, err := time.Parse("2006-01-02T15:04:05Z", *expiresAt)
		if err != nil {
			expTime, err = time.Parse("2006-01-02 15:04:05", *expiresAt)
		}
		if err == nil {
			daysLeft := expTime.Sub(time.Now()).Hours() / 24
			if daysLeft > 10 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Renewal is only available within 10 days before expiry / 到期前 10 天内才能续期",
				})
				return
			}
		}
	}

	// Extend by 3 months from now
	newExpiry := time.Now().AddDate(0, 3, 0)
	database.DB.Exec("UPDATE mailboxes SET expires_at = ? WHERE email = ?", newExpiry, emailStr)

	c.JSON(http.StatusOK, gin.H{
		"message":    "renewed",
		"expires_at": newExpiry.Format("2006-01-02"),
	})
}

// ===== 邮箱用户查看邮件 =====

func MailboxListEmails(c *gin.Context) {
	email, _ := c.Get("mailbox_email")
	emailStr := email.(string)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	// 查邮件：recipient 已统一为纯邮箱地址
	where := "WHERE recipient = ?"
	args := []interface{}{emailStr}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM emails "+where, args...).Scan(&total)

	queryArgs := append(args, size, offset)
	rows, err := database.DB.Query(
		`SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(extracted_code,''), has_attachments, is_read, COALESCE(is_starred,0), received_at
		 FROM emails `+where+` ORDER BY received_at DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var emails []database.EmailListItem
	for rows.Next() {
		var e database.EmailListItem
		var hasAtt, isRead, isStarred int
		if err := rows.Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject, &e.ExtractedCode, &hasAtt, &isRead, &isStarred, &e.ReceivedAt); err != nil {
			continue
		}
		e.HasAttachments = hasAtt == 1
		e.IsRead = isRead == 1
		e.IsStarred = isStarred == 1
		emails = append(emails, e)
	}
	if emails == nil {
		emails = []database.EmailListItem{}
	}

	c.JSON(http.StatusOK, database.PaginatedResponse{
		Total: total, Page: page, Size: size, Data: emails,
	})
}

func MailboxGetEmail(c *gin.Context) {
	email, _ := c.Get("mailbox_email")
	emailStr := email.(string)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var e database.Email
	var hasAtt, isRead, isStarred int
	err := database.DB.QueryRow(
		`SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(body_text,''), COALESCE(body_html,''),
		 COALESCE(extracted_code,''), COALESCE(extracted_links,''), has_attachments, raw_size, is_read, COALESCE(is_starred,0), received_at
		 FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject,
		&e.BodyText, &e.BodyHTML, &e.ExtractedCode, &e.ExtractedLinks,
		&hasAtt, &e.RawSize, &isRead, &isStarred, &e.ReceivedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否是自己的邮件
	if e.Recipient != emailStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	e.HasAttachments = hasAtt == 1
	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1

	database.DB.Exec("UPDATE emails SET is_read = 1 WHERE id = ?", id)
	c.JSON(http.StatusOK, e)
}

// ===== 邮箱用户导出数据 =====

func MailboxExportData(c *gin.Context) {
	email, _ := c.Get("mailbox_email")
	emailStr := email.(string)

	rows, err := database.DB.Query(
		`SELECT COALESCE(sender,''), COALESCE(subject,''), COALESCE(body_text,''), COALESCE(body_html,''),
		        COALESCE(extracted_code,''), received_at
		 FROM emails WHERE recipient = ? ORDER BY received_at DESC`,
		emailStr,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type ExportEmail struct {
		Sender     string `json:"sender"`
		Subject    string `json:"subject"`
		BodyText   string `json:"body_text"`
		BodyHTML   string `json:"body_html"`
		Code       string `json:"code,omitempty"`
		ReceivedAt string `json:"received_at"`
	}

	var emails []ExportEmail
	for rows.Next() {
		var e ExportEmail
		rows.Scan(&e.Sender, &e.Subject, &e.BodyText, &e.BodyHTML, &e.Code, &e.ReceivedAt)
		emails = append(emails, e)
	}

	export := map[string]interface{}{
		"email":       emailStr,
		"exported_at": time.Now().Format("2006-01-02 15:04:05"),
		"total":       len(emails),
		"emails":      emails,
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_export.json"`, emailStr))
	c.JSON(http.StatusOK, export)
}

// ===== 邮箱用户 JWT 中间件 =====

func MailboxAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader || tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["type"] != "mailbox" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		c.Set("mailbox_email", claims["email"])
		c.Set("mailbox_id", claims["mailbox_id"])
		c.Next()
	}
}

// verifyTurnstile calls Cloudflare Turnstile siteverify API
func verifyTurnstile(secret, token, remoteIP string) bool {
	resp, err := http.Post("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(fmt.Sprintf("secret=%s&response=%s&remoteip=%s", secret, token, remoteIP)))
	if err != nil {
		log.Printf("[turnstile] verify error: %v", err)
		return false
	}
	defer resp.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Success
}

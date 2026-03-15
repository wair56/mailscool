package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mailer/config"
	"mailer/database"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Telegram Bot — 纯 HTTP 实现，无需第三方库

const tgAPI = "https://api.telegram.org/bot"

type tgUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      *struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat *struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type tgResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// getChatEmail returns the latest mailbox email for a TG chatID from DB
func getChatEmail(chatID int64) string {
	var email string
	database.DB.QueryRow(
		`SELECT email FROM mailboxes WHERE telegram_chat_id = ? AND (expires_at IS NULL OR expires_at > datetime('now')) ORDER BY id DESC LIMIT 1`,
		chatID,
	).Scan(&email)
	return email
}

// StartTelegramBot starts the bot if token is configured (DB setting or env var)
func StartTelegramBot() {
	// Try DB setting first, then env var
	token := database.GetSetting("telegram_bot_token", "")
	if token == "" {
		token = config.C.TelegramBotToken
	}
	if token == "" {
		log.Println("[telegram] No bot token configured (set in Settings or TELEGRAM_BOT_TOKEN env), bot disabled")
		// Keep polling DB for token every 30s in case admin sets it later
		go watchForToken()
		return
	}

	log.Println("[telegram] Bot starting...")
	runBot(token)
}

func watchForToken() {
	for {
		time.Sleep(30 * time.Second)
		token := database.GetSetting("telegram_bot_token", "")
		if token != "" {
			log.Println("[telegram] Bot token found in settings, starting bot...")
			runBot(token)
			return
		}
	}
}

func runBot(token string) {
	// Long polling loop (push notifications are now event-driven via webhook.go)
	offset := 0
	for {
		updates, err := getUpdates(token, offset)
		if err != nil {
			log.Printf("[telegram] poll error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil && u.Message.Text != "" {
				handleCommand(token, u)
			}
		}
	}
}

func getUpdates(token string, offset int) ([]tgUpdate, error) {
	url := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=30", tgAPI, token, offset)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result tgResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func handleCommand(token string, u tgUpdate) {
	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)
	cmd := strings.ToLower(strings.Split(text, " ")[0])

	switch cmd {
	case "/start":
		sendMessage(token, chatID, "🔐 *MailsCool Telegram Bot*\n\n"+
			"📬 /new — Create temp mailbox\n"+
			"📥 /check — View latest emails\n"+
			"🔑 /code — Get latest verification code\n"+
			"ℹ️ /status — Current mailbox info")
	case "/new":
		handleNew(token, chatID)
	case "/check":
		handleCheck(token, chatID)
	case "/code":
		handleCode(token, chatID)
	case "/status":
		handleStatus(token, chatID)
	default:
		sendMessage(token, chatID, "Unknown command. Send /start for help.")
	}
}



func handleNew(token string, chatID int64) {
	// Rate limit: max 5 mailboxes per chatID per day
	var todayCount int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM mailboxes WHERE created_ua LIKE ? AND created_at >= datetime('now', '-1 day')`,
		fmt.Sprintf("telegram:chat_%d", chatID),
	).Scan(&todayCount)
	maxPerDay := database.GetSettingInt("telegram_max_mailboxes_per_day", 5)
	if todayCount >= maxPerDay {
		sendMessage(token, chatID, fmt.Sprintf("❌ Daily limit reached (%d/day). Try again tomorrow.", maxPerDay))
		return
	}

	// Generate realistic username (same style as web registration)
	firstNames := []string{"alex", "sam", "mike", "lisa", "emma", "jack", "anna", "tom", "kate", "ryan"}
	lastNames := []string{"chen", "wang", "lee", "kim", "garcia", "smith", "jones", "brown", "taylor", "wilson"}
	seps := []string{".", "_", ""}
	rIdx := make([]byte, 4)
	rand.Read(rIdx)
	first := firstNames[int(rIdx[0])%len(firstNames)]
	last := lastNames[int(rIdx[1])%len(lastNames)]
	sep := seps[int(rIdx[2])%len(seps)]
	num := fmt.Sprintf("%02d", int(rIdx[3])%100)
	username := first + sep + last + num

	// Find a domain: MUST use temp_email_domains config, no fallback
	var domainID int64
	var domain string
	configured := database.GetSetting("temp_email_domains", "")
	if configured == "" {
		sendMessage(token, chatID, "❌ Temp email domains not configured. Contact admin.")
		return
	}
	parts := strings.Split(configured, ",")
	type domainInfo struct {
		id   int64
		name string
	}
	var activeDomains []domainInfo
	for _, d := range parts {
		d = strings.TrimSpace(d)
		var did int64
		var dname string
		if e := database.DB.QueryRow("SELECT id, name FROM domains WHERE name = ? AND is_active = 1", d).Scan(&did, &dname); e == nil {
			activeDomains = append(activeDomains, domainInfo{did, dname})
		}
	}
	if len(activeDomains) == 0 {
		sendMessage(token, chatID, "❌ No active temp email domains available.")
		return
	}
	pickBytes := make([]byte, 1)
	rand.Read(pickBytes)
	pick := activeDomains[int(pickBytes[0])%len(activeDomains)]
	domainID = pick.id
	domain = pick.name

	email := username + "@" + domain

	// Generate password (same as web: 6 bytes = 12 hex chars)
	passBytes := make([]byte, 6)
	rand.Read(passBytes)
	password := hex.EncodeToString(passBytes)

	// Hash and encrypt
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
	encrypted, _ := config.Encrypt(password)

	// Set expiry (from config, default 3 months)
	expiryMonths := database.GetSettingInt("temp_mailbox_expiry_months", 3)
	expiresAt := time.Now().AddDate(0, expiryMonths, 0).Format("2006-01-02 15:04:05")

	_, err := database.DB.Exec(
		`INSERT INTO mailboxes (email, password_hash, password_plain, is_temp, expires_at, domain_id, created_ip, created_ua, telegram_chat_id) VALUES (?, ?, ?, 1, ?, ?, 'telegram', ?, ?)`,
		email, string(hash), encrypted, expiresAt, domainID, fmt.Sprintf("telegram:chat_%d", chatID), chatID,
	)
	if err != nil {
		sendMessage(token, chatID, "❌ Failed to create mailbox: "+err.Error())
		return
	}

	// Build login URL
	baseURL := database.GetSetting("base_url", "")
	if baseURL == "" {
		baseURL = "https://mails.cool"
	}
	loginURL := fmt.Sprintf("%s/login?email=%s&password=%s", baseURL, email, password)

	sendMessage(token, chatID, fmt.Sprintf(
		"✅ *Mailbox Created!*\n\n📧 `%s`\n🔑 `%s`\n⏰ Expires: %s\n\n🔗 [Open Inbox](%s)\n\nNew emails will be pushed here automatically.",
		email, password, expiresAt[:10], loginURL))
}

func handleCheck(token string, chatID int64) {
	email := getChatEmail(chatID)

	if email == "" {
		sendMessage(token, chatID, "No mailbox linked. Send /new to create one.")
		return
	}

	rows, err := database.DB.Query(
		`SELECT sender, subject, extracted_code, received_at FROM emails WHERE recipient = ? ORDER BY id DESC LIMIT 5`, email)
	if err != nil {
		sendMessage(token, chatID, "❌ Query error")
		return
	}
	defer rows.Close()

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📬 *Latest emails for* `%s`\n\n", email))

	count := 0
	for rows.Next() {
		var sender, subject, code, receivedAt string
		rows.Scan(&sender, &subject, &code, &receivedAt)
		count++
		codeStr := ""
		if code != "" {
			codeStr = fmt.Sprintf(" 🔑 `%s`", code)
		}
		msg.WriteString(fmt.Sprintf("%d. *%s*%s\n   From: %s\n   %s\n\n",
			count, truncate(subject, 40), codeStr, truncate(sender, 30), receivedAt[:16]))
	}

	if count == 0 {
		msg.WriteString("No emails yet.")
	}

	sendMessage(token, chatID, msg.String())
}

func handleCode(token string, chatID int64) {
	email := getChatEmail(chatID)

	if email == "" {
		sendMessage(token, chatID, "No mailbox linked. Send /new to create one.")
		return
	}

	var code, subject string
	err := database.DB.QueryRow(
		`SELECT extracted_code, subject FROM emails WHERE recipient = ? AND extracted_code != '' ORDER BY id DESC LIMIT 1`, email,
	).Scan(&code, &subject)

	if err != nil || code == "" {
		sendMessage(token, chatID, "No verification code found.")
		return
	}

	sendMessage(token, chatID, fmt.Sprintf("🔑 *Verification Code:* `%s`\n📧 From: %s", code, truncate(subject, 50)))
}

func handleStatus(token string, chatID int64) {
	email := getChatEmail(chatID)

	if email == "" {
		sendMessage(token, chatID, "No mailbox linked. Send /new to create one.")
		return
	}

	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE recipient = ?", email).Scan(&count)

	sendMessage(token, chatID, fmt.Sprintf("📧 Mailbox: `%s`\n📬 Emails: %d", email, count))
}



func sendMessage(token string, chatID int64, text string) {
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*2) * time.Second) // 指数退避: 2s, 4s
		}
		resp, err := http.Post(
			fmt.Sprintf("%s%s/sendMessage", tgAPI, token),
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			lastErr = err
			continue
		}
		io.ReadAll(resp.Body) // drain
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}
	log.Printf("[telegram] send failed after 3 retries: %v", lastErr)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"mailer/config"
	"mailer/database"
	"mailer/service"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ReceiveEmail 通过 HTTP POST 接收原始邮件（Cloudflare Email Worker 调用）
// 鉴权通过 API Key 中间件完成
func ReceiveEmail(c *gin.Context) {
	// 限制请求体大小 10MB
	const maxBodySize = 10 << 20
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
		return
	}
	if int64(len(raw)) > maxBodySize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "邮件过大，超过10MB限制"})
		return
	}
	if len(raw) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "空邮件数据"})
		return
	}

	// 解析邮件
	parsed, err := service.ParseEmail(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮件解析失败: " + err.Error()})
		return
	}

	// 优先使用信封收件人（Worker 传的 X-Envelope-To），解决多 To/CC 问题
	envelopeTo := c.GetHeader("X-Envelope-To")
	if envelopeTo != "" {
		envelopeTo = strings.TrimSpace(strings.ToLower(envelopeTo))
		// Override parsed.To with envelope recipient for correct storage
		parsed.To = envelopeTo
	}

	// 提取收件人域名
	domain := extractEmailDomain(parsed.To)
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法提取收件人域名"})
		return
	}

	// 查询域名是否存在且启用
	var domainID int64
	var isActive int
	err = database.DB.QueryRow("SELECT id, is_active FROM domains WHERE name = ?", domain).Scan(&domainID, &isActive)
	if err != nil {
		log.Printf("[receive] 域名未注册: %s, 丢弃邮件", domain)
		c.JSON(http.StatusOK, gin.H{"status": "dropped", "reason": "domain not registered"})
		return
	}

	if isActive == 0 {
		log.Printf("[receive] 域名已禁用: %s, 丢弃邮件", domain)
		c.JSON(http.StatusOK, gin.H{"status": "dropped", "reason": "domain disabled"})
		return
	}

	// 检查发件人黑名单
	if isBlockedSender(parsed.From, domainID) {
		log.Printf("[receive] 发件人被封锁: %s -> %s", parsed.From, parsed.To)
		c.JSON(http.StatusOK, gin.H{"status": "blocked", "reason": "sender blocked"})
		return
	}

	// 使用统一的存储函数
	emailID, err := service.StoreEmail(parsed, domainID)
	if err != nil {
		log.Printf("[receive] 存储失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储失败"})
		return
	}

	// 自动创建邮箱（如果不存在）
	recipientAddr := extractEmailAddr(parsed.To)
	if recipientAddr != "" {
		var exists int
		database.DB.QueryRow("SELECT COUNT(*) FROM mailboxes WHERE email = ?", recipientAddr).Scan(&exists)
		if exists == 0 {
			pwdBytes := make([]byte, 6)
			rand.Read(pwdBytes)
			pwd := hex.EncodeToString(pwdBytes)
			hash, _ := bcrypt.GenerateFromPassword([]byte(pwd), 10)
			expiresMonths := database.GetSettingInt("temp_mailbox_expiry_months", 3)
			expiresAt := time.Now().AddDate(0, expiresMonths, 0)
			encryptedPwd, _ := config.Encrypt(pwd)
			database.DB.Exec(
				"INSERT OR IGNORE INTO mailboxes (email, password_plain, password_hash, domain_id, is_temp, expires_at) VALUES (?, ?, ?, ?, 1, ?)",
				recipientAddr, encryptedPwd, string(hash), domainID, expiresAt,
			)
			log.Printf("[receive] 自动创建邮箱: %s (过期: %s)", recipientAddr, expiresAt.Format("2006-01-02"))
		}
	}

	// 事件驱动推送（Webhook + TG）
	if recipientAddr != "" {
		go service.NotifyMailbox(recipientAddr, parsed)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "stored",
		"email_id": emailID,
	})
}

func extractEmailAddr(addr string) string {
	if idx := strings.Index(addr, "<"); idx >= 0 {
		addr = addr[idx+1:]
		if idx2 := strings.Index(addr, ">"); idx2 >= 0 {
			addr = addr[:idx2]
		}
	}
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, "@") {
		return strings.ToLower(addr)
	}
	return ""
}

func extractEmailDomain(addr string) string {
	if idx := strings.Index(addr, "<"); idx >= 0 {
		addr = addr[idx+1:]
		if idx2 := strings.Index(addr, ">"); idx2 >= 0 {
			addr = addr[:idx2]
		}
	}
	parts := strings.Split(addr, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

func isBlockedSender(sender string, domainID int64) bool {
	senderLower := strings.ToLower(sender)

	// 域名级白名单
	var count int
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM sender_rules WHERE domain_id = ? AND rule_type = 'whitelist'", domainID,
	).Scan(&count)
	if count > 0 {
		var matched int
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM sender_rules 
			 WHERE domain_id = ? AND rule_type = 'whitelist' AND ? LIKE REPLACE(sender_pattern, '*', '%')`,
			domainID, senderLower,
		).Scan(&matched)
		if matched > 0 {
			return false
		}
	}

	// 域名级黑名单
	var blocked int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM sender_rules 
		 WHERE domain_id = ? AND rule_type = 'blacklist' AND ? LIKE REPLACE(sender_pattern, '*', '%')`,
		domainID, senderLower,
	).Scan(&blocked)
	if blocked > 0 {
		return true
	}

	// 全局黑名单
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM sender_rules 
		 WHERE domain_id IS NULL AND rule_type = 'blacklist' AND ? LIKE REPLACE(sender_pattern, '*', '%')`,
		senderLower,
	).Scan(&blocked)

	return blocked > 0
}

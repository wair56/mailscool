package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"mailer/service"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	// Postfix pipe 入口：从 stdin 读取原始邮件
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read from stdin: %v", err)
	}

	if len(raw) == 0 {
		log.Fatal("Empty email data")
	}

	// 初始化数据库连接
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/mailer.db"
	}

	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 解析邮件
	parsed, err := service.ParseEmail(raw)
	if err != nil {
		log.Fatalf("Failed to parse email: %v", err)
	}

	// 提取收件人域名
	domain := extractDomain(parsed.To)
	if domain == "" {
		log.Fatalf("Cannot extract domain from recipient: %s", parsed.To)
	}

	// 查询域名是否存在且启用
	var domainID int64
	var isActive int
	err = db.QueryRow("SELECT id, is_active FROM domains WHERE name = ?", domain).Scan(&domainID, &isActive)
	if err != nil {
		log.Printf("Domain not found: %s, dropping email", domain)
		os.Exit(0)
	}

	if isActive == 0 {
		log.Printf("Domain disabled: %s, dropping email", domain)
		os.Exit(0)
	}

	// 检查发件人黑白名单
	if isBlocked(parsed.From, domainID) {
		log.Printf("Sender blocked: %s -> %s", parsed.From, parsed.To)
		os.Exit(0)
	}

	// 存储邮件（直接用本地 db 实例）
	linksJSON := "[]"
	if len(parsed.ExtractedLinks) > 0 {
		linksJSON = toJSON(parsed.ExtractedLinks)
	}

	result, err := db.Exec(
		`INSERT INTO emails (domain_id, recipient, sender, subject, body_text, body_html,
		 extracted_code, extracted_links, has_attachments, raw_size, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		domainID, parsed.To, parsed.From, parsed.Subject,
		parsed.BodyText, parsed.BodyHTML,
		parsed.ExtractedCode, linksJSON,
		boolToInt(parsed.HasAttachments), parsed.RawSize,
	)
	if err != nil {
		log.Fatalf("Failed to store email: %v", err)
	}

	emailID, _ := result.LastInsertId()
	log.Printf("Email stored: #%d %s -> %s [%s]", emailID, parsed.From, parsed.To, parsed.Subject)
}

func extractDomain(addr string) string {
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

func isBlocked(sender string, domainID int64) bool {
	senderLower := strings.ToLower(sender)

	// 域名级白名单
	var count int
	db.QueryRow(
		"SELECT COUNT(*) FROM sender_rules WHERE domain_id = ? AND rule_type = 'whitelist'", domainID,
	).Scan(&count)
	if count > 0 {
		var matched int
		db.QueryRow(
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
	db.QueryRow(
		`SELECT COUNT(*) FROM sender_rules 
		 WHERE domain_id = ? AND rule_type = 'blacklist' AND ? LIKE REPLACE(sender_pattern, '*', '%')`,
		domainID, senderLower,
	).Scan(&blocked)
	if blocked > 0 {
		return true
	}

	// 全局黑名单
	db.QueryRow(
		`SELECT COUNT(*) FROM sender_rules 
		 WHERE domain_id IS NULL AND rule_type = 'blacklist' AND ? LIKE REPLACE(sender_pattern, '*', '%')`,
		senderLower,
	).Scan(&blocked)

	return blocked > 0
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func toJSON(links []string) string {
	b, err := json.Marshal(links)
	if err != nil {
		return "[]"
	}
	return string(b)
}

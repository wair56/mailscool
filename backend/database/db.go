package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"mailer/config"
	"math/big"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func Init() {
	var err error
	DB, err = sql.Open("sqlite3", config.C.DBPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(4) // WAL 模式支持并发读，允许多连接
	DB.SetMaxIdleConns(4)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	migrate()
	ensureDefaultAdmin()
	initJWTSecret()
	log.Println("Database initialized successfully")
}

func migrate() {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			is_active INTEGER DEFAULT 1,
			note TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER NOT NULL,
			recipient TEXT NOT NULL,
			sender TEXT NOT NULL,
			subject TEXT,
			body_text TEXT,
			body_html TEXT,
			extracted_code TEXT,
			extracted_links TEXT,
			has_attachments INTEGER DEFAULT 0,
			raw_size INTEGER,
			is_read INTEGER DEFAULT 0,
			received_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_recipient ON emails(recipient, received_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_domain ON emails(domain_id, received_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_received ON emails(received_at)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			key_plain TEXT,
			name TEXT,
			ip_whitelist TEXT,
			rate_limit INTEGER DEFAULT 100,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS api_key_domains (
			api_key_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			PRIMARY KEY (api_key_id, domain_id),
			FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS sender_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain_id INTEGER,
			sender_pattern TEXT NOT NULL,
			rule_type TEXT NOT NULL,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'admin',
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS admin_domains (
			admin_id INTEGER NOT NULL,
			domain_id INTEGER NOT NULL,
			PRIMARY KEY (admin_id, domain_id),
			FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE,
			FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER,
			action TEXT NOT NULL,
			target TEXT,
			detail TEXT,
			ip TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range tables {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("Migration failed: %v\nQuery: %s", err, q)
		}
	}

	// 兼容旧表：添加 key_plain 列
	DB.Exec("ALTER TABLE api_keys ADD COLUMN key_plain TEXT")
	// 兼容旧表：添加 is_starred 列
	DB.Exec("ALTER TABLE emails ADD COLUMN is_starred INTEGER DEFAULT 0")
	// 兼容旧表：添加 is_system 列（系统自动创建的 key 不可删除）
	DB.Exec("ALTER TABLE api_keys ADD COLUMN is_system INTEGER DEFAULT 0")
	DB.Exec("ALTER TABLE api_keys ADD COLUMN created_by INTEGER DEFAULT 0")

	// mailboxes 表
	DB.Exec(`CREATE TABLE IF NOT EXISTS mailboxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_plain TEXT,
		password_hash TEXT NOT NULL,
		domain_id INTEGER,
		is_temp INTEGER DEFAULT 0,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
	)`)
	// 兼容旧 mailboxes 表
	DB.Exec("ALTER TABLE mailboxes ADD COLUMN is_temp INTEGER DEFAULT 0")
	DB.Exec("ALTER TABLE mailboxes ADD COLUMN expires_at DATETIME")
	DB.Exec("ALTER TABLE mailboxes ADD COLUMN created_ip TEXT")
	DB.Exec("ALTER TABLE mailboxes ADD COLUMN created_ua TEXT")

	// system_settings KV 表
	DB.Exec(`CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// 初始化默认配置
	DB.Exec("INSERT OR IGNORE INTO system_settings (key, value) VALUES ('mail_retention_days', '7')")
	DB.Exec("INSERT OR IGNORE INTO system_settings (key, value) VALUES ('temp_mailbox_expiry_months', '3')")

	// === 性能优化：索引 ===
	performanceIndexes := []string{
		// 邮件查询加速
		"CREATE INDEX IF NOT EXISTS idx_emails_sender ON emails(sender)",
		"CREATE INDEX IF NOT EXISTS idx_emails_subject ON emails(subject)",
		"CREATE INDEX IF NOT EXISTS idx_emails_starred ON emails(is_starred) WHERE is_starred = 1",
		"CREATE INDEX IF NOT EXISTS idx_emails_domain_received ON emails(domain_id, received_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_emails_recipient ON emails(recipient)",
		// 邮箱查询加速
		"CREATE INDEX IF NOT EXISTS idx_mailboxes_email ON mailboxes(email)",
		"CREATE INDEX IF NOT EXISTS idx_mailboxes_domain ON mailboxes(domain_id)",
		"CREATE INDEX IF NOT EXISTS idx_mailboxes_temp_expires ON mailboxes(is_temp, expires_at) WHERE is_temp = 1",
		// 权限查询加速
		"CREATE INDEX IF NOT EXISTS idx_admin_domains_admin ON admin_domains(admin_id)",
		"CREATE INDEX IF NOT EXISTS idx_admin_domains_domain ON admin_domains(domain_id)",
		"CREATE INDEX IF NOT EXISTS idx_api_key_domains_domain ON api_key_domains(domain_id)",
		// 审计日志
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)",
	}
	for _, idx := range performanceIndexes {
		DB.Exec(idx)
	}

	// === 性能优化：PRAGMA ===
	DB.Exec("PRAGMA journal_mode=WAL")
	DB.Exec("PRAGMA synchronous=NORMAL")
	DB.Exec("PRAGMA cache_size=-64000") // 64MB cache
	DB.Exec("PRAGMA busy_timeout=5000")
	DB.Exec("PRAGMA temp_store=MEMORY")

	// === 一次性迁移：加密现有明文数据 ===
	migrateEncryptPlaintext()

	// === 一次性迁移：统一 recipient 为纯邮箱地址 ===
	migrateNormalizeRecipients()

	// === 安全列迁移：Webhook + TG ===
	safeAddColumn("mailboxes", "webhook_url", "TEXT DEFAULT ''")
	safeAddColumn("mailboxes", "telegram_chat_id", "INTEGER DEFAULT 0")
}

func safeAddColumn(table, column, colDef string) {
	_, err := DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colDef))
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		log.Printf("[migrate] add column %s.%s: %v", table, column, err)
	}
}

func ensureDefaultAdmin() {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count > 0 {
		return
	}

	// 生成随机强密码 16 位
	password := generateRandomPassword(16)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatalf("Failed to hash default admin password: %v", err)
	}

	username := "maileradm"
	_, err = DB.Exec(
		"INSERT INTO admins (username, password_hash, role) VALUES (?, ?, 'super_admin')",
		username, string(hash),
	)
	if err != nil {
		log.Fatalf("Failed to create default admin: %v", err)
	}

	log.Println("╔══════════════════════════════════════════════════╗")
	log.Println("║          🔑 初始超级管理员已创建                ║")
	log.Println("╠══════════════════════════════════════════════════╣")
	log.Printf("║  用户名: %-40s║\n", username)
	log.Printf("║  密  码: %-40s║\n", password)
	log.Println("╠══════════════════════════════════════════════════╣")
	log.Println("║  ⚠️  请立即登录并修改密码！此密码仅显示一次！    ║")
	log.Println("╚══════════════════════════════════════════════════╝")
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// initJWTSecret 初始化 JWT 密钥：环境变量 > 数据库 > 生成并持久化
func initJWTSecret() {
	// 1. 环境变量已设置，直接使用
	if config.C.JWTSecret != "" && config.C.JWTSecret != "change-me-in-production" {
		return
	}

	// 2. 尝试从数据库读取
	dbSecret := GetSetting("jwt_secret", "")
	if dbSecret != "" {
		config.C.JWTSecret = dbSecret
		log.Println("✅ JWT_SECRET 从数据库加载")
		return
	}

	// 3. 首次启动：生成随机密钥并持久化到数据库
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		config.C.JWTSecret = "fallback-secret-please-change"
		log.Println("⚠️  JWT_SECRET 生成失败，请手动设置环境变量 JWT_SECRET")
		return
	}
	secret := hex.EncodeToString(b)
	config.C.JWTSecret = secret
	SetSetting("jwt_secret", secret)
	log.Println("🔑 JWT_SECRET 已自动生成并持久化到数据库（重启不会变化）")
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

// GetSetting 获取系统配置值
func GetSetting(key, fallback string) string {
	var val string
	err := DB.QueryRow("SELECT value FROM system_settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return fallback
	}
	return val
}

// GetSettingInt 获取整数配置值
func GetSettingInt(key string, fallback int) int {
	val := GetSetting(key, "")
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

// SetSetting 设置系统配置值
func SetSetting(key, value string) error {
	_, err := DB.Exec(
		"INSERT OR REPLACE INTO system_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		key, value,
	)
	return err
}

// migrateEncryptPlaintext 一次性迁移：将现有 password_plain 和 key_plain 明文加密
func migrateEncryptPlaintext() {
	// 检查是否已迁移
	migrated := GetSetting("plaintext_encrypted", "0")
	if migrated == "1" {
		return
	}

	// 加密 mailboxes.password_plain
	rows, err := DB.Query("SELECT id, password_plain FROM mailboxes WHERE password_plain IS NOT NULL AND password_plain != ''")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var plain string
			if err := rows.Scan(&id, &plain); err != nil {
				continue
			}
			// 跳过已加密的数据（hex 编码的密文很长）
			if len(plain) > 50 {
				continue
			}
			encrypted, err := config.Encrypt(plain)
			if err != nil {
				log.Printf("[migrate] failed to encrypt mailbox password %d: %v", id, err)
				continue
			}
			DB.Exec("UPDATE mailboxes SET password_plain = ? WHERE id = ?", encrypted, id)
		}
	}

	// 加密 api_keys.key_plain
	rows2, err := DB.Query("SELECT id, key_plain FROM api_keys WHERE key_plain IS NOT NULL AND key_plain != ''")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id int64
			var plain string
			if err := rows2.Scan(&id, &plain); err != nil {
				continue
			}
			if len(plain) > 50 {
				continue
			}
			encrypted, err := config.Encrypt(plain)
			if err != nil {
				log.Printf("[migrate] failed to encrypt api key %d: %v", id, err)
				continue
			}
			DB.Exec("UPDATE api_keys SET key_plain = ? WHERE id = ?", encrypted, id)
		}
	}

	SetSetting("plaintext_encrypted", "1")
	log.Println("Plaintext encryption migration completed")
}

func migrateNormalizeRecipients() {
	if GetSetting("recipients_normalized", "") == "1" {
		return
	}
	// Fix recipients that contain <email> format: extract pure email address
	result, err := DB.Exec(`UPDATE emails SET recipient = 
		LOWER(TRIM(SUBSTR(recipient, INSTR(recipient, '<') + 1, INSTR(recipient, '>') - INSTR(recipient, '<') - 1)))
		WHERE recipient LIKE '%<%>%'`)
	if err != nil {
		log.Printf("[migrate] normalize recipients error: %v", err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("[migrate] normalized %d recipient addresses", affected)
	}
	SetSetting("recipients_normalized", "1")
}

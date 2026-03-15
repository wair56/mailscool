package handler

import (
	"mailer/database"
	"mailer/middleware"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 登录失败锁定（Admin + Mailbox 共用）
var loginLock = struct {
	mu       sync.Mutex
	failures map[string]*loginAttempt
}{failures: make(map[string]*loginAttempt)}

type loginAttempt struct {
	count    int
	lockedAt time.Time
	lastSeen time.Time
}

const maxLoginAttempts = 5
const lockDuration = 15 * time.Minute

func init() {
	// 定期清理过期的登录锁定条目，防止内存泄漏
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			loginLock.mu.Lock()
			for k, a := range loginLock.failures {
				if time.Since(a.lastSeen) > 30*time.Minute {
					delete(loginLock.failures, k)
				}
			}
			loginLock.mu.Unlock()
		}
	}()
}

// CheckLoginLock 检查 IP 是否被锁定，返回 true 表示已锁定
func CheckLoginLock(ip string) bool {
	loginLock.mu.Lock()
	defer loginLock.mu.Unlock()
	attempt, exists := loginLock.failures[ip]
	if !exists {
		return false
	}
	if attempt.count >= maxLoginAttempts {
		if time.Since(attempt.lockedAt) < lockDuration {
			return true
		}
		// 锁定期过，重置
		delete(loginLock.failures, ip)
	}
	return false
}

// RecordLoginFailure 记录登录失败
func RecordLoginFailure(ip string) {
	loginLock.mu.Lock()
	defer loginLock.mu.Unlock()
	attempt, exists := loginLock.failures[ip]
	if !exists {
		attempt = &loginAttempt{}
		loginLock.failures[ip] = attempt
	}
	attempt.count++
	attempt.lastSeen = time.Now()
	if attempt.count >= maxLoginAttempts {
		attempt.lockedAt = time.Now()
	}
}

// ClearLoginFailure 清除登录失败记录
func ClearLoginFailure(ip string) {
	loginLock.mu.Lock()
	delete(loginLock.failures, ip)
	loginLock.mu.Unlock()
}

func Login(c *gin.Context) {
	var req database.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	clientIP := c.ClientIP()

	// 检查 IP 锁定
	if CheckLoginLock(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
		return
	}

	// 查询管理员
	var admin database.Admin
	var passwordHash string
	var isActive int
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, role, is_active, created_at FROM admins WHERE username = ?",
		req.Username,
	).Scan(&admin.ID, &admin.Username, &passwordHash, &admin.Role, &isActive, &admin.CreatedAt)

	if err != nil || isActive == 0 {
		RecordLoginFailure(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		RecordLoginFailure(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	admin.IsActive = true

	// 生成 JWT
	token, err := middleware.GenerateToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 清除登录失败记录
	ClearLoginFailure(clientIP)

	// 记录审计日志
	LogAudit(c, admin.ID, "login", "admin", "login success")

	c.JSON(http.StatusOK, database.LoginResponse{
		Token: token,
		Admin: admin,
	})
}

func GetCurrentAdmin(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	var admin database.Admin
	var isActive int
	err := database.DB.QueryRow(
		"SELECT id, username, role, is_active, created_at FROM admins WHERE id = ?",
		adminID,
	).Scan(&admin.ID, &admin.Username, &admin.Role, &isActive, &admin.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	admin.IsActive = isActive == 1
	c.JSON(http.StatusOK, admin)
}

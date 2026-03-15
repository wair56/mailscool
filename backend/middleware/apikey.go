package middleware

import (
	"encoding/json"
	"mailer/database"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ApiKeyAuth API Key 鉴权中间件
func ApiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		key := strings.TrimPrefix(authHeader, "Bearer ")
		if key == authHeader || !strings.HasPrefix(key, "sk_") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key format"})
			c.Abort()
			return
		}

		prefix := key[:11] // "sk_" + 8 chars

		// 查找匹配的 API Key
		rows, err := database.DB.Query(
			"SELECT id, key_hash, COALESCE(ip_whitelist,''), rate_limit, is_active, expires_at FROM api_keys WHERE key_prefix = ?",
			prefix,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			c.Abort()
			return
		}
		defer rows.Close()

		var matched struct {
			id          int64
			ipWhitelist string
			rateLimit   int
		}
		found := false

		for rows.Next() {
			var id int64
			var keyHash, ipWhitelist string
			var rateLimit int
			var isActive int
			var expiresAt *time.Time

			if err := rows.Scan(&id, &keyHash, &ipWhitelist, &rateLimit, &isActive, &expiresAt); err != nil {
				continue
			}

			if isActive == 0 {
				continue
			}

			if expiresAt != nil && expiresAt.Before(time.Now()) {
				continue
			}

			if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(key)); err == nil {
				matched.id = id
				matched.ipWhitelist = ipWhitelist
				matched.rateLimit = rateLimit
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired api key"})
			c.Abort()
			return
		}

		// IP 白名单验证
		if matched.ipWhitelist != "" {
			clientIP := c.ClientIP()
			if !isIPAllowed(clientIP, matched.ipWhitelist) {
				c.JSON(http.StatusForbidden, gin.H{"error": "ip not allowed"})
				c.Abort()
				return
			}
		}

		// 获取 API Key 绑定的域名 ID 列表
		domainIDs, err := getApiKeyDomains(matched.id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			c.Abort()
			return
		}

		c.Set("api_key_id", matched.id)
		c.Set("rate_limit", matched.rateLimit)
		c.Set("allowed_domain_ids", domainIDs)
		c.Next()
	}
}

func isIPAllowed(clientIP, whitelist string) bool {
	var ips []string
	if err := json.Unmarshal([]byte(whitelist), &ips); err != nil {
		// 尝试逗号分隔
		ips = strings.Split(whitelist, ",")
	}
	for _, allowed := range ips {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		// CIDR 支持
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err == nil && ipNet.Contains(net.ParseIP(clientIP)) {
				return true
			}
		} else if allowed == clientIP {
			return true
		}
	}
	return false
}

func getApiKeyDomains(apiKeyID int64) ([]int64, error) {
	rows, err := database.DB.Query("SELECT domain_id FROM api_key_domains WHERE api_key_id = ?", apiKeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

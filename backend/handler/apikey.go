package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"mailer/config"
	"mailer/database"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func ListApiKeys(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	var query string
	var args []interface{}

	if role == "super_admin" {
		query = `SELECT ak.id, ak.key_prefix, COALESCE(ak.key_plain,''), ak.name, COALESCE(ak.ip_whitelist,''), ak.rate_limit, ak.is_active, COALESCE(ak.is_system,0), ak.created_at, ak.expires_at, COALESCE(ak.created_by,0), COALESCE(adm.username,'') 
		 FROM api_keys ak LEFT JOIN admins adm ON ak.created_by = adm.id ORDER BY ak.id DESC`
	} else {
		// 普通管理员只能看到与其管理域名关联的 API Key
		query = `SELECT DISTINCT ak.id, ak.key_prefix, COALESCE(ak.key_plain,''), ak.name, COALESCE(ak.ip_whitelist,''), ak.rate_limit, ak.is_active, COALESCE(ak.is_system,0), ak.created_at, ak.expires_at, COALESCE(ak.created_by,0), COALESCE(adm.username,'') 
		 FROM api_keys ak
		 LEFT JOIN admins adm ON ak.created_by = adm.id
		 INNER JOIN api_key_domains akd ON ak.id = akd.api_key_id
		 INNER JOIN admin_domains ad ON akd.domain_id = ad.domain_id
		 WHERE ad.admin_id = ?
		 ORDER BY ak.id DESC`
		args = append(args, adminID)
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 第一步：收集所有 key（避免嵌套查询导致 SQLite 死锁）
	var keys []database.ApiKey
	for rows.Next() {
		var k database.ApiKey
		var isActive, isSystem int
		var keyPlain string
		if err := rows.Scan(&k.ID, &k.KeyPrefix, &keyPlain, &k.Name, &k.IPWhitelist, &k.RateLimit, &isActive, &isSystem, &k.CreatedAt, &k.ExpiresAt, &k.CreatedBy, &k.CreatedByName); err != nil {
			continue
		}
		k.IsActive = isActive == 1
		k.IsSystem = isSystem == 1
		if keyPlain != "" {
			// 解密 key_plain
			if decrypted, err := config.Decrypt(keyPlain); err == nil {
				k.KeyPrefix = decrypted
			} else {
				k.KeyPrefix = keyPlain
			}
		}
		keys = append(keys, k)
	}
	rows.Close()

	// 第二步：为每个 key 加载关联域名及统计数据（rows 已关闭，不会死锁）
	for i := range keys {
		domainRows, _ := database.DB.Query(
			`SELECT d.id, d.name, d.is_active FROM domains d 
			 INNER JOIN api_key_domains akd ON d.id = akd.domain_id 
			 WHERE akd.api_key_id = ?`, keys[i].ID,
		)
		if domainRows != nil {
			for domainRows.Next() {
				var d database.Domain
				var da int
				domainRows.Scan(&d.ID, &d.Name, &da)
				d.IsActive = da == 1
				keys[i].Domains = append(keys[i].Domains, d)
			}
			domainRows.Close()
		}
		if keys[i].Domains == nil {
			keys[i].Domains = []database.Domain{}
		}

		// 统计关联域名下的邮件数和邮箱数
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM emails WHERE domain_id IN (SELECT domain_id FROM api_key_domains WHERE api_key_id = ?)`,
			keys[i].ID,
		).Scan(&keys[i].TotalEmails)
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM mailboxes WHERE domain_id IN (SELECT domain_id FROM api_key_domains WHERE api_key_id = ?)`,
			keys[i].ID,
		).Scan(&keys[i].TotalMailboxes)
	}
	if keys == nil {
		keys = []database.ApiKey{}
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func CreateApiKey(c *gin.Context) {
	var req database.CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.RateLimit <= 0 {
		req.RateLimit = 100
	}

	// 非超管校验域名权限
	role, _ := c.Get("role")
	if role != "super_admin" {
		for _, domainID := range req.DomainIDs {
			if !hasDomainAccess(c, domainID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权关联此域名"})
				return
			}
		}
	}

	// 生成 API Key: sk_ + 32 random hex chars
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	key := "sk_" + hex.EncodeToString(randomBytes)
	prefix := key[:11] // sk_ + 8 chars

	hash, err := bcrypt.GenerateFromPassword([]byte(key), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash key"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	// AES 加密 key_plain
	encryptedKey, _ := config.Encrypt(key)

	// 事务：创建 key + 绑定域名
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	adminID, _ := c.Get("admin_id")

	result, err := tx.Exec(
		`INSERT INTO api_keys (key_prefix, key_hash, key_plain, name, ip_whitelist, rate_limit, expires_at, created_by) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		prefix, string(hash), encryptedKey, req.Name, req.IPWhitelist, req.RateLimit, expiresAt, adminID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	keyID, _ := result.LastInsertId()

	// 绑定域名
	for _, domainID := range req.DomainIDs {
		if _, err := tx.Exec(
			"INSERT INTO api_key_domains (api_key_id, domain_id) VALUES (?, ?)",
			keyID, domainID,
		); err != nil {
			log.Printf("[warn] failed to bind domain %d to key %d: %v", domainID, keyID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
		return
	}

	LogAudit(c, adminID.(int64), "create_api_key", req.Name, "")

	c.JSON(http.StatusCreated, database.CreateApiKeyResponse{
		Key: key,
		ApiKey: database.ApiKey{
			ID:        keyID,
			KeyPrefix: prefix,
			Name:      req.Name,
			RateLimit: req.RateLimit,
			IsActive:  true,
		},
	})
}

// adminCanAccessApiKey 检查非超管是否有权访问指定 API Key
func adminCanAccessApiKey(adminID int64, keyID int64) bool {
	var count int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM api_key_domains akd
		 INNER JOIN admin_domains ad ON akd.domain_id = ad.domain_id
		 WHERE akd.api_key_id = ? AND ad.admin_id = ?`, keyID, adminID,
	).Scan(&count)
	return count > 0
}

func ToggleApiKey(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	// 非超管权限检查
	if role != "super_admin" && !adminCanAccessApiKey(adminID.(int64), id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此 API Key"})
		return
	}

	var isActive int
	err := database.DB.QueryRow("SELECT is_active FROM api_keys WHERE id = ?", id).Scan(&isActive)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}

	newState := 1 - isActive
	database.DB.Exec("UPDATE api_keys SET is_active = ? WHERE id = ?", newState, id)

	LogAudit(c, adminID.(int64), "toggle_api_key", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"is_active": newState == 1})
}

// UpdateApiKey 编辑 API Key（名称、速率、IP白名单、有效期、域名）
func UpdateApiKey(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	if role != "super_admin" && !adminCanAccessApiKey(adminID.(int64), id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此 API Key"})
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		RateLimit   *int     `json:"rate_limit"`
		IPWhitelist *string  `json:"ip_whitelist"`
		ExpiresAt   *string  `json:"expires_at"` // ISO 8601 or empty to clear
		DomainIDs   *[]int64 `json:"domain_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 更新基本字段
	if req.Name != nil {
		database.DB.Exec("UPDATE api_keys SET name = ? WHERE id = ?", *req.Name, id)
	}
	if req.RateLimit != nil {
		database.DB.Exec("UPDATE api_keys SET rate_limit = ? WHERE id = ?", *req.RateLimit, id)
	}
	if req.IPWhitelist != nil {
		database.DB.Exec("UPDATE api_keys SET ip_whitelist = ? WHERE id = ?", *req.IPWhitelist, id)
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			database.DB.Exec("UPDATE api_keys SET expires_at = NULL WHERE id = ?", id)
		} else {
			t, err := time.Parse("2006-01-02", *req.ExpiresAt)
			if err != nil {
				t, err = time.Parse(time.RFC3339, *req.ExpiresAt)
			}
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format"})
				return
			}
			database.DB.Exec("UPDATE api_keys SET expires_at = ? WHERE id = ?", t, id)
		}
	}

	// 更新域名绑定（用事务保护）
	if req.DomainIDs != nil {
		tx, txErr := database.DB.Begin()
		if txErr == nil {
			defer tx.Rollback()
			tx.Exec("DELETE FROM api_key_domains WHERE api_key_id = ?", id)
			for _, did := range *req.DomainIDs {
				tx.Exec("INSERT INTO api_key_domains (api_key_id, domain_id) VALUES (?, ?)", id, did)
			}
			if err := tx.Commit(); err != nil {
				log.Printf("[warn] failed to update api key domains: %v", err)
			}
		}
	}

	LogAudit(c, adminID.(int64), "update_api_key", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteApiKey(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	// 非超管权限检查
	if role != "super_admin" && !adminCanAccessApiKey(adminID.(int64), id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此 API Key"})
		return
	}

	result, _ := database.DB.Exec("DELETE FROM api_keys WHERE id = ?", id)
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "api key not found"})
		return
	}

	LogAudit(c, adminID.(int64), "delete_api_key", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

package handler

import (
	"mailer/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func ListAdmins(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT a.id, a.username, a.role, a.is_active, a.created_at,
		  (SELECT COUNT(*) FROM admin_domains WHERE admin_id = a.id) as domain_count
		 FROM admins a ORDER BY a.id`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var admins []database.Admin
	for rows.Next() {
		var a database.Admin
		var isActive int
		var domainCount int
		if err := rows.Scan(&a.ID, &a.Username, &a.Role, &isActive, &a.CreatedAt, &domainCount); err != nil {
			continue
		}
		a.IsActive = isActive == 1
		a.DomainCount = domainCount
		admins = append(admins, a)
	}
	if admins == nil {
		admins = []database.Admin{}
	}

	c.JSON(http.StatusOK, gin.H{"data": admins})
}

func CreateAdmin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Role == "" {
		req.Role = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO admins (username, password_hash, role) VALUES (?, ?, ?)",
		req.Username, string(hash), req.Role,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	id, _ := result.LastInsertId()
	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "create_admin", req.Username, "role: "+req.Role)

	c.JSON(http.StatusCreated, gin.H{"id": id, "username": req.Username, "role": req.Role})
}

func DeleteAdmin(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 不能删除自己
	currentID, _ := c.Get("admin_id")
	if id == currentID.(int64) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	result, _ := database.DB.Exec("DELETE FROM admins WHERE id = ?", id)
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "delete_admin", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ChangePassword(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var passwordHash string
	database.DB.QueryRow("SELECT password_hash FROM admins WHERE id = ?", adminID).Scan(&passwordHash)

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "old password incorrect"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	database.DB.Exec("UPDATE admins SET password_hash = ? WHERE id = ?", string(hash), adminID)

	LogAudit(c, adminID.(int64), "change_password", "", "")
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// 获取管理员已分配的域名 ID 列表
func GetAdminDomains(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.DB.Query("SELECT domain_id FROM admin_domains WHERE admin_id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var domainIDs []int64
	for rows.Next() {
		var did int64
		rows.Scan(&did)
		domainIDs = append(domainIDs, did)
	}
	if domainIDs == nil {
		domainIDs = []int64{}
	}

	c.JSON(http.StatusOK, gin.H{"domain_ids": domainIDs})
}

// 更新管理员域名分配（全量替换）
func UpdateAdminDomains(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		DomainIDs []int64 `json:"domain_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 用事务保护先删后插
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM admin_domains WHERE admin_id = ?", id)
	for _, did := range req.DomainIDs {
		tx.Exec("INSERT OR IGNORE INTO admin_domains (admin_id, domain_id) VALUES (?, ?)", id, did)
	}

	// 同步清理：该管理员创建的 API Key 中，不再有权限的域名绑定
	// 删除 api_key_domains 中 domain_id 已不在新 domain_ids 列表里的记录
	if len(req.DomainIDs) > 0 {
		// 构建占位符
		placeholders := ""
		cleanArgs := []interface{}{id}
		for i, did := range req.DomainIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			cleanArgs = append(cleanArgs, did)
		}
		tx.Exec(
			`DELETE FROM api_key_domains WHERE api_key_id IN (SELECT id FROM api_keys WHERE created_by = ?) AND domain_id NOT IN (`+placeholders+`)`,
			cleanArgs...,
		)
	} else {
		// 如果没有任何域名权限了，清除该管理员所有 API Key 的域名绑定
		tx.Exec(`DELETE FROM api_key_domains WHERE api_key_id IN (SELECT id FROM api_keys WHERE created_by = ?)`, id)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
		return
	}

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "update_admin_domains", c.Param("id"), "")

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

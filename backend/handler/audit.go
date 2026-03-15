package handler

import (
	"mailer/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LogAudit 记录操作审计日志
func LogAudit(c *gin.Context, adminID int64, action, target, detail string) {
	ip := c.ClientIP()
	database.DB.Exec(
		"INSERT INTO audit_logs (admin_id, action, target, detail, ip) VALUES (?, ?, ?, ?, ?)",
		adminID, action, target, detail, ip,
	)
}

func ListAuditLogs(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	offset := (page - 1) * size

	var where string
	var args []interface{}

	if role != "super_admin" {
		where = " WHERE al.admin_id = ?"
		args = append(args, adminID)
	}

	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM audit_logs al"+where, args...).Scan(&total)

	queryArgs := append(args, size, offset)
	rows, err := database.DB.Query(
		`SELECT al.id, al.admin_id, COALESCE(a.username, ''), al.action, al.target, al.detail, al.ip, al.created_at 
		 FROM audit_logs al LEFT JOIN admins a ON al.admin_id = a.id`+where+`
		 ORDER BY al.id DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type AuditLogWithAdmin struct {
		database.AuditLog
		Username string `json:"username"`
	}

	var logs []AuditLogWithAdmin
	for rows.Next() {
		var l AuditLogWithAdmin
		if err := rows.Scan(&l.ID, &l.AdminID, &l.Username, &l.Action, &l.Target, &l.Detail, &l.IP, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AuditLogWithAdmin{}
	}

	c.JSON(http.StatusOK, database.PaginatedResponse{
		Total: total,
		Page:  page,
		Size:  size,
		Data:  logs,
	})
}

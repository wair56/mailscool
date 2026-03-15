package handler

import (
	"mailer/config"
	"mailer/database"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	// 检查数据库连通性
	err := database.DB.Ping()
	status := "ok"
	if err != nil {
		status = "error"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"version": "1.0.0",
	})
}

func Dashboard(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")
	var totalEmails, todayEmails, totalDomains, activeDomains, totalApiKeys int64

	if role == "super_admin" {
		database.DB.QueryRow("SELECT COUNT(*) FROM emails").Scan(&totalEmails)
		database.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE received_at >= date('now')").Scan(&todayEmails)
		database.DB.QueryRow("SELECT COUNT(*) FROM domains").Scan(&totalDomains)
		database.DB.QueryRow("SELECT COUNT(*) FROM domains WHERE is_active = 1").Scan(&activeDomains)
		database.DB.QueryRow("SELECT COUNT(*) FROM api_keys WHERE is_active = 1").Scan(&totalApiKeys)
	} else {
		database.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)", adminID).Scan(&totalEmails)
		database.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE received_at >= date('now') AND domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)", adminID).Scan(&todayEmails)
		database.DB.QueryRow("SELECT COUNT(*) FROM admin_domains WHERE admin_id = ?", adminID).Scan(&totalDomains)
		database.DB.QueryRow(`SELECT COUNT(*) FROM domains WHERE is_active = 1 AND id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)`, adminID).Scan(&activeDomains)
		database.DB.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE is_active = 1 AND id IN (SELECT api_key_id FROM api_key_domains WHERE domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?))`, adminID).Scan(&totalApiKeys)
	}

	// 最近 7 天每日收件量
	var dailyQuery string
	var dailyArgs []interface{}
	if role == "super_admin" {
		dailyQuery = `SELECT date(received_at) as day, COUNT(*) as cnt 
		 FROM emails WHERE received_at >= date('now', '-7 days') 
		 GROUP BY day ORDER BY day`
	} else {
		dailyQuery = `SELECT date(received_at) as day, COUNT(*) as cnt 
		 FROM emails WHERE received_at >= date('now', '-7 days') 
		 AND domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)
		 GROUP BY day ORDER BY day`
		dailyArgs = append(dailyArgs, adminID)
	}

	rows, _ := database.DB.Query(dailyQuery, dailyArgs...)
	var dailyStats []gin.H
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var cnt int64
			rows.Scan(&day, &cnt)
			dailyStats = append(dailyStats, gin.H{"date": day, "count": cnt})
		}
	}
	if dailyStats == nil {
		dailyStats = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_emails":   totalEmails,
		"today_emails":   todayEmails,
		"total_domains":  totalDomains,
		"active_domains": activeDomains,
		"total_api_keys": totalApiKeys,
		"daily_stats":    dailyStats,
	})
}

func DownloadDatabase(c *gin.Context) {
	dbPath := config.C.DBPath

	// 使用 VACUUM INTO 创建一致性快照
	tmpPath := dbPath + ".download"
	_, err := database.DB.Exec("VACUUM INTO ?", tmpPath)
	if err != nil {
		// fallback: 直接发送原文件
		c.Header("Content-Disposition", "attachment; filename=mailer.db")
		c.File(dbPath)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=mailer.db")
	c.File(tmpPath)

	// 清理临时文件
	go func() {
		// 等一会儿再删，确保文件已发送完毕
		<-c.Request.Context().Done()
		os.Remove(tmpPath)
	}()
}

package service

import (
	"fmt"
	"log"
	"mailer/config"
	"mailer/database"
	"time"
)

// StartCleanupWorker 启动定期清理过期邮件的后台任务
func StartCleanupWorker() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// 启动时执行一次
		cleanupExpiredEmails()

		for range ticker.C {
			cleanupExpiredEmails()
		}
	}()
	log.Printf("Cleanup worker started")
}

func cleanupExpiredEmails() {
	days := database.GetSettingInt("mail_retention_days", config.C.MailRetentionDays)
	if days <= 0 {
		return // 0 或负数表示不清理
	}

	result, err := database.DB.Exec(
		"DELETE FROM emails WHERE received_at < datetime('now', ?) AND (is_starred = 0 OR is_starred IS NULL)",
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		log.Printf("Cleanup error: %v", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("Cleaned up %d expired emails (older than %d days)", affected, days)
	}

	// 清理过期临时邮箱及其邮件
	expiredRows, err := database.DB.Query("SELECT email FROM mailboxes WHERE is_temp = 1 AND expires_at IS NOT NULL AND expires_at < datetime('now')")
	var expiredEmails []string
	if err == nil && expiredRows != nil {
		func() {
			defer expiredRows.Close()
			for expiredRows.Next() {
				var email string
				if err := expiredRows.Scan(&email); err != nil {
					continue
				}
				expiredEmails = append(expiredEmails, email)
			}
		}()
	}

	for _, email := range expiredEmails {
		// 精确匹配：recipient 已统一为纯邮箱地址
		database.DB.Exec("DELETE FROM emails WHERE recipient = ?", email)
	}

	expResult, _ := database.DB.Exec("DELETE FROM mailboxes WHERE is_temp = 1 AND expires_at IS NOT NULL AND expires_at < datetime('now')")
	if expResult != nil {
		expAffected, _ := expResult.RowsAffected()
		if expAffected > 0 {
			log.Printf("Cleaned up %d expired temp mailboxes", expAffected)
		}
	}
}

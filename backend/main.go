package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"mailer/config"
	"mailer/database"
	"mailer/handler"
	"mailer/middleware"
	"mailer/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var frontendFS embed.FS

type ginLogWriter struct{}

func (w *ginLogWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimRight(string(p), "\n")
	if line != "" {
		handler.AppendLog(line)
	}
	return os.Stderr.Write(p)
}

func main() {
	config.Init()
	database.Init()
	defer database.Close()

	// 初始化日志捕获到内存环形缓冲
	handler.InitLogCapture()

	// 启动邮件清理后台任务
	service.StartCleanupWorker()

	// 启动 Telegram Bot（如果配置了 token）
	go service.StartTelegramBot()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// Gin 请求日志也写入环形缓冲
	r.Use(gin.LoggerWithWriter(&ginLogWriter{}))

	// CORS
	configuredOrigins := config.C.CORSOrigins
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// 允许浏览器扩展
			if strings.HasPrefix(origin, "chrome-extension://") || strings.HasPrefix(origin, "moz-extension://") {
				return true
			}
			for _, o := range configuredOrigins {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// ===== 公开接口 =====
	r.GET("/api/health", handler.HealthCheck)
	r.POST("/admin/login", handler.Login)
	r.POST("/mailbox/login", handler.MailboxLogin)
	r.POST("/mailbox/register", handler.RegisterTempMailbox)
	r.GET("/public/config", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"turnstile_site_key": database.GetSetting("turnstile_site_key", ""),
		})
	})

	// ===== API Key 鉴权接口（供脚本调用）=====
	api := r.Group("/api")
	api.Use(middleware.ApiKeyAuth())
	api.Use(middleware.RateLimit())
	{
		api.GET("/emails", handler.ApiListEmails)
		api.GET("/emails/:id", handler.ApiGetEmail)
		api.GET("/emails/latest", handler.ApiGetLatestEmail)
		api.DELETE("/emails/:id", handler.ApiDeleteEmail)
		api.GET("/domains", handler.ApiListDomains)
		api.GET("/domains/:id/stats", handler.GetDomainStats)
		api.POST("/receive", handler.ReceiveEmail)
		api.POST("/mailboxes", handler.ApiCreateTempMailbox)
	}

	// ===== JWT 鉴权接口（Web 管理后台）=====
	admin := r.Group("/admin")
	admin.Use(middleware.JWTAuth())
	{
		admin.GET("/me", handler.GetCurrentAdmin)
		admin.PUT("/password", handler.ChangePassword)
		admin.GET("/dashboard", handler.Dashboard)

		// 域名管理
		admin.GET("/domains", handler.ListDomains)
		admin.POST("/domains", handler.CreateDomain)
		admin.PUT("/domains/:id", handler.UpdateDomain)
		admin.DELETE("/domains/:id", handler.DeleteDomain)
		admin.PUT("/domains/:id/toggle", handler.ToggleDomain)
		admin.POST("/domains/:id/check-dns", handler.CheckDomainDNS)
		admin.POST("/domains/:id/cf-setup", handler.CloudflareSetup)
		admin.GET("/domains/:id/stats", handler.GetDomainStats)

		// 邮件浏览
		admin.GET("/emails", handler.AdminListEmails)
		admin.GET("/emails/:id", handler.AdminGetEmail)
		admin.PUT("/emails/:id/star", handler.AdminToggleStar)

		// 邮箱管理
		admin.GET("/mailboxes", handler.AdminListMailboxes)
		admin.POST("/mailboxes", handler.AdminCreateMailbox)
		admin.PUT("/mailboxes/:id", handler.AdminUpdateMailbox)
		admin.DELETE("/mailboxes/:id", handler.AdminDeleteMailbox)

		// API Key 管理
		admin.GET("/api-keys", handler.ListApiKeys)
		admin.POST("/api-keys", handler.CreateApiKey)
		admin.PUT("/api-keys/:id/toggle", handler.ToggleApiKey)
		admin.PUT("/api-keys/:id", handler.UpdateApiKey)
		admin.DELETE("/api-keys/:id", handler.DeleteApiKey)

		// 管理员管理（仅超管）
		superAdmin := admin.Group("")
		superAdmin.Use(middleware.RequireSuperAdmin())
		{
			superAdmin.GET("/admins", handler.ListAdmins)
			superAdmin.POST("/admins", handler.CreateAdmin)
			superAdmin.DELETE("/admins/:id", handler.DeleteAdmin)
			superAdmin.GET("/admins/:id/domains", handler.GetAdminDomains)
			superAdmin.PUT("/admins/:id/domains", handler.UpdateAdminDomains)
			superAdmin.GET("/download-db", handler.DownloadDatabase)
			superAdmin.GET("/system-status", handler.GetSystemStatus)
			superAdmin.GET("/system-logs", handler.GetSystemLogs)
			superAdmin.POST("/clean-cache", handler.CleanCache)
		}

		// 审计日志
		admin.GET("/audit-logs", handler.ListAuditLogs)

		// 系统配置
		admin.GET("/settings", handler.GetSystemSettings)
		admin.PUT("/settings", handler.UpdateSystemSettings)
	}

	// ===== 邮箱用户接口 =====
	mailbox := r.Group("/mailbox")
	mailbox.Use(handler.MailboxAuth())
	{
		mailbox.GET("/me", handler.MailboxGetMe)
		mailbox.POST("/renew", handler.MailboxRenew)
		mailbox.GET("/emails", handler.MailboxListEmails)
		mailbox.GET("/emails/:id", handler.MailboxGetEmail)
		mailbox.GET("/export", handler.MailboxExportData)
	}

	// ===== 前端静态文件 =====
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Printf("Warning: frontend dist not found, skipping static file serving")
	} else {
		// 读取 index.html 用于 SPA fallback
		indexHTML, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			log.Printf("Warning: index.html not found in dist")
		}

		fileServer := http.FileServer(http.FS(distFS))

		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// 跳过 API 路径（只匹配 /api/ 前缀，避免 /api-keys 等前端路由误匹配）
			if strings.HasPrefix(path, "/api/") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			if strings.HasPrefix(path, "/admin/") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			if strings.HasPrefix(path, "/mailbox/") {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}

			// 尝试作为静态文件提供
			filePath := path[1:] // 去掉前导 /
			if filePath != "" {
				if f, err := distFS.Open(filePath); err == nil {
					f.Close()
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}

			// SPA fallback: 返回 index.html（禁缓存以确保每次获取最新版）
			if indexHTML != nil {
				c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
				c.Data(200, "text/html; charset=utf-8", indexHTML)
				return
			}
			c.String(404, "Not Found")
		})
	}

	log.Printf("Server starting on %s", config.C.ListenAddr)
	if err := r.Run(config.C.ListenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

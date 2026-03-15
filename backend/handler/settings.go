package handler

import (
	"fmt"
	"log"
	"mailer/database"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GetSystemSettings 获取所有系统配置
func GetSystemSettings(c *gin.Context) {
	rows, err := database.DB.Query("SELECT key, value FROM system_settings")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		settings[k] = v
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSystemSettings 批量更新系统配置
func UpdateSystemSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 只允许修改已知的配置项
	allowed := map[string]bool{
		"mail_retention_days":         true,
		"temp_mailbox_expiry_months":  true,
		"temp_email_domains":          true,
		"temp_mailbox_per_ip_daily":   true,
		"temp_mailbox_daily_total":    true,
		"telegram_bot_token":          true,
		"turnstile_site_key":          true,
		"turnstile_secret_key":        true,
	}

	adminID, _ := c.Get("admin_id")
	for k, v := range req {
		if !allowed[k] {
			continue
		}
		database.SetSetting(k, v)
		LogAudit(c, adminID.(int64), "update_setting", k, v)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetSystemStatus 获取服务器状态（CPU/内存/磁盘/运行时间）
func GetSystemStatus(c *gin.Context) {
	status := make(map[string]interface{})

	// 运行时间 - 直接读 /proc/uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			if secs, err := strconv.ParseFloat(parts[0], 64); err == nil {
				days := int(secs) / 86400
				hours := (int(secs) % 86400) / 3600
				mins := (int(secs) % 3600) / 60
				if days > 0 {
					status["uptime"] = fmt.Sprintf("%d days %d hours %d min", days, hours, mins)
				} else if hours > 0 {
					status["uptime"] = fmt.Sprintf("%d hours %d min", hours, mins)
				} else {
					status["uptime"] = fmt.Sprintf("%d min", mins)
				}
			}
		}
	}

	// CPU 负载
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			status["load_1m"] = parts[0]
			status["load_5m"] = parts[1]
			status["load_15m"] = parts[2]
		}
	}

	// 内存
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		mem := make(map[string]int64)
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				switch {
				case strings.HasPrefix(line, "MemTotal:"):
					mem["total"] = val
				case strings.HasPrefix(line, "MemAvailable:"):
					mem["available"] = val
				}
			}
		}
		if mem["total"] > 0 {
			status["mem_total_mb"] = mem["total"] / 1024
			status["mem_available_mb"] = mem["available"] / 1024
			status["mem_used_mb"] = (mem["total"] - mem["available"]) / 1024
			status["mem_percent"] = fmt.Sprintf("%.1f", float64(mem["total"]-mem["available"])/float64(mem["total"])*100)
		}
	}

	// 磁盘
	if out, err := exec.Command("df", "-h", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 5 {
				status["disk_total"] = parts[1]
				status["disk_used"] = parts[2]
				status["disk_avail"] = parts[3]
				status["disk_percent"] = parts[4]
			}
		}
	}

	// Go runtime info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	status["go_alloc_mb"] = fmt.Sprintf("%.1f", float64(m.Alloc)/1024/1024)
	status["go_goroutines"] = runtime.NumGoroutine()

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// ===== 内置日志环形缓冲 =====

var (
	logBuffer   []string
	logBufferMu sync.Mutex
	logMaxLines = 500
)

// AppendLog 添加一行日志到内存缓冲
func AppendLog(line string) {
	logBufferMu.Lock()
	defer logBufferMu.Unlock()
	logBuffer = append(logBuffer, line)
	if len(logBuffer) > logMaxLines {
		logBuffer = logBuffer[len(logBuffer)-logMaxLines:]
	}
}

// InitLogCapture 启动日志捕获（从 stdout 接管）
func InitLogCapture() {
	// 从 Gin 的日志和 Go log 中捕获
	log.SetOutput(&logWriter{})
}

type logWriter struct{}

func (w *logWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimRight(string(p), "\n")
	if line != "" {
		AppendLog(time.Now().Format("2006-01-02 15:04:05") + " " + line)
	}
	// 同时输出到标准错误
	os.Stderr.WriteString(string(p))
	return len(p), nil
}

// GetSystemLogs 获取最近 N 行应用日志
func GetSystemLogs(c *gin.Context) {
	n, _ := strconv.Atoi(c.DefaultQuery("lines", "200"))
	if n <= 0 || n > 500 {
		n = 200
	}

	logBufferMu.Lock()
	defer logBufferMu.Unlock()

	start := 0
	if len(logBuffer) > n {
		start = len(logBuffer) - n
	}

	var result strings.Builder
	for _, line := range logBuffer[start:] {
		result.WriteString(line)
		result.WriteString("\n")
	}

	c.JSON(http.StatusOK, gin.H{"data": result.String()})
}

// CleanCache 清理 Docker 无用镜像和缓存 + Go GC
func CleanCache(c *gin.Context) {
	// Force Go GC
	runtime.GC()

	// 执行 docker system prune -f（清理无用镜像、容器、网络）
	var pruneResult string
	if out, err := exec.Command("docker", "system", "prune", "-f").CombinedOutput(); err != nil {
		pruneResult = fmt.Sprintf("docker prune failed: %v\n%s", err, string(out))
	} else {
		pruneResult = strings.TrimSpace(string(out))
	}

	// 获取清理后的内存状态
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "clean_cache", "docker_prune", pruneResult)

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"prune_result": pruneResult,
		"go_alloc_mb":  fmt.Sprintf("%.1f", float64(m.Alloc)/1024/1024),
	})
}

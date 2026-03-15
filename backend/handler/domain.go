package handler

import (
	"database/sql"
	"mailer/database"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ===== Admin 域名管理接口 =====

func ListDomains(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	baseSelect := `SELECT d.id, d.name, d.is_active, COALESCE(d.note,''), d.created_at,
		(SELECT COUNT(*) FROM api_key_domains akd WHERE akd.domain_id = d.id) AS total_api_keys,
		(SELECT COUNT(*) FROM mailboxes m WHERE m.domain_id = d.id) AS total_mailboxes,
		(SELECT COUNT(*) FROM emails e WHERE e.domain_id = d.id) AS total_emails`

	var rows *sql.Rows
	var err error

	if role == "super_admin" {
		rows, err = database.DB.Query(baseSelect + " FROM domains d ORDER BY d.id DESC")
	} else {
		rows, err = database.DB.Query(
			baseSelect+` FROM domains d
			 INNER JOIN admin_domains ad ON d.id = ad.domain_id WHERE ad.admin_id = ? ORDER BY d.id DESC`,
			adminID,
		)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var domains []database.Domain
	for rows.Next() {
		var d database.Domain
		var isActive int
		if err := rows.Scan(&d.ID, &d.Name, &isActive, &d.Note, &d.CreatedAt,
			&d.TotalApiKeys, &d.TotalMailboxes, &d.TotalEmails); err != nil {
			continue
		}
		d.IsActive = isActive == 1
		domains = append(domains, d)
	}

	if domains == nil {
		domains = []database.Domain{}
	}

	c.JSON(http.StatusOK, gin.H{"data": domains})
}

func CreateDomain(c *gin.Context) {
	var req database.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO domains (name, note) VALUES (?, ?)",
		req.Name, req.Note,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "domain already exists or invalid"})
		return
	}

	id, _ := result.LastInsertId()
	adminID, _ := c.Get("admin_id")

	// 自动关联域名到当前管理员
	database.DB.Exec("INSERT OR IGNORE INTO admin_domains (admin_id, domain_id) VALUES (?, ?)", adminID, id)

	LogAudit(c, adminID.(int64), "create_domain", req.Name, "")

	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func UpdateDomain(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !hasDomainAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此域名"})
		return
	}
	var req database.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Note != nil {
		database.DB.Exec("UPDATE domains SET note = ? WHERE id = ?", *req.Note, id)
	}
	if req.IsActive != nil {
		active := 0
		if *req.IsActive {
			active = 1
		}
		database.DB.Exec("UPDATE domains SET is_active = ? WHERE id = ?", active, id)
	}

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "update_domain", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func DeleteDomain(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !hasDomainAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此域名"})
		return
	}

	result, err := database.DB.Exec("DELETE FROM domains WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "delete_domain", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ToggleDomain(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !hasDomainAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此域名"})
		return
	}

	var isActive int
	err := database.DB.QueryRow("SELECT is_active FROM domains WHERE id = ?", id).Scan(&isActive)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	newState := 1 - isActive
	database.DB.Exec("UPDATE domains SET is_active = ? WHERE id = ?", newState, id)

	adminID, _ := c.Get("admin_id")
	LogAudit(c, adminID.(int64), "toggle_domain", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"is_active": newState == 1})
}

// getServerIPs 获取本服务器的公网 IP（通过反向解析 hostname，或直接用请求 Host）
func getServerIPs() []string {
	// 尝试获取本机所有 IP
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP.String())
				}
			}
		}
	}
	return ips
}

// resolveMXToIPs 将 MX 主机名解析为 IP 地址
func resolveMXToIPs(mxHost string) []string {
	// 去除末尾的点
	mxHost = strings.TrimSuffix(mxHost, ".")
	ips, err := net.LookupHost(mxHost)
	if err != nil {
		return nil
	}
	return ips
}

func CheckDomainDNS(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !hasDomainAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此域名"})
		return
	}

	var name string
	err := database.DB.QueryRow("SELECT name FROM domains WHERE id = ?", id).Scan(&name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	type CheckItem struct {
		Name    string   `json:"name"`
		Status  string   `json:"status"` // "pass", "fail", "warn"
		Detail  string   `json:"detail"`
		Records []string `json:"records,omitempty"`
	}

	checks := []CheckItem{}

	// 获取服务器 IP（用于后续比对）
	serverIPs := getServerIPs()
	// 也从请求上下文取 Host 对应 IP
	requestHost := c.Request.Host
	if colonIdx := strings.LastIndex(requestHost, ":"); colonIdx > 0 {
		requestHost = requestHost[:colonIdx]
	}
	// 如果 requestHost 是 IP，加入比对列表
	if ip := net.ParseIP(requestHost); ip != nil {
		serverIPs = append(serverIPs, requestHost)
	}

	// 1. MX 记录检查
	mxRecords, _ := net.LookupMX(name)
	mxCheck := CheckItem{Name: "MX 记录", Records: []string{}}
	hasCloudflare := false
	if len(mxRecords) > 0 {
		for _, mx := range mxRecords {
			mxCheck.Records = append(mxCheck.Records, mx.Host)
			host := strings.ToLower(mx.Host)
			if strings.Contains(host, "cloudflare") ||
				strings.Contains(host, "cfmail") ||
				strings.Contains(host, "route") {
				hasCloudflare = true
			}
		}
		if hasCloudflare {
			mxCheck.Status = "pass"
			mxCheck.Detail = "MX 指向 Cloudflare Email Routing，通过 Worker 转发到本系统"
		} else {
			mxCheck.Status = "fail"
			mxCheck.Detail = "MX 记录未指向 Cloudflare。本系统通过 Cloudflare Email Worker 接收邮件，请启用 Cloudflare Email Routing"
		}
	} else {
		mxCheck.Status = "fail"
		mxCheck.Detail = "未找到 MX 记录，请在 Cloudflare 启用 Email Routing"
	}
	checks = append(checks, mxCheck)

	// 2. Cloudflare Email Routing 检测
	cfCheck := CheckItem{Name: "Cloudflare Email Routing"}
	if hasCloudflare {
		cfCheck.Status = "pass"
		cfCheck.Detail = "检测到 Cloudflare Email Routing 的 MX 记录"
	} else if len(mxRecords) > 0 {
		cfCheck.Status = "warn"
		cfCheck.Detail = "MX 记录存在但不是 Cloudflare，如果使用其他邮件转发也可正常工作"
	} else {
		cfCheck.Status = "fail"
		cfCheck.Detail = "未检测到，请在 Cloudflare 控制台启用 Email Routing 并配置 Catch-all 规则"
	}
	checks = append(checks, cfCheck)

	// 3. SPF 记录检查
	spfCheck := CheckItem{Name: "SPF 记录", Records: []string{}}
	txtRecords, _ := net.LookupTXT(name)
	hasSPF := false
	for _, txt := range txtRecords {
		if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
			hasSPF = true
			spfCheck.Records = append(spfCheck.Records, txt)
		}
	}
	if hasSPF {
		spfCheck.Status = "pass"
		spfCheck.Detail = "SPF 记录已配置"
	} else {
		spfCheck.Status = "warn"
		spfCheck.Detail = "未找到 SPF 记录（纯收件模式下非必需）"
	}
	checks = append(checks, spfCheck)

	// 4. DMARC 记录检查
	dmarcCheck := CheckItem{Name: "DMARC 记录", Records: []string{}}
	dmarcRecords, _ := net.LookupTXT("_dmarc." + name)
	hasDMARC := false
	for _, txt := range dmarcRecords {
		if strings.HasPrefix(strings.ToLower(txt), "v=dmarc") {
			hasDMARC = true
			dmarcCheck.Records = append(dmarcCheck.Records, txt)
		}
	}
	if hasDMARC {
		dmarcCheck.Status = "pass"
		dmarcCheck.Detail = "DMARC 记录已配置"
	} else {
		dmarcCheck.Status = "warn"
		dmarcCheck.Detail = "未找到 DMARC 记录（纯收件模式下非必需）"
	}
	checks = append(checks, dmarcCheck)

	// 总体状态：MX 必须通过
	overallStatus := "pass"
	for _, ch := range checks {
		if ch.Status == "fail" {
			overallStatus = "fail"
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"domain": name,
		"status": overallStatus,
		"checks": checks,
	})
}

// ===== API 域名接口（API Key 鉴权）=====

func ApiListDomains(c *gin.Context) {
	domainIDs, _ := c.Get("allowed_domain_ids")
	ids := domainIDs.([]int64)

	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []database.Domain{}})
		return
	}

	// 构建 IN 查询
	query := "SELECT id, name, is_active, COALESCE(note,''), created_at FROM domains WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var domains []database.Domain
	for rows.Next() {
		var d database.Domain
		var isActive int
		if err := rows.Scan(&d.ID, &d.Name, &isActive, &d.Note, &d.CreatedAt); err != nil {
			// fallback: domain listing without note
			continue
		}
		d.IsActive = isActive == 1
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []database.Domain{}
	}

	c.JSON(http.StatusOK, gin.H{"data": domains})
}

func GetDomainStats(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 验证域名权限（API Key 或 Admin）
	if domainIDs, exists := c.Get("allowed_domain_ids"); exists {
		ids := domainIDs.([]int64)
		allowed := false
		for _, did := range ids {
			if did == id {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	} else if !hasDomainAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var stats database.DomainStats
	stats.DomainID = id

	database.DB.QueryRow("SELECT name FROM domains WHERE id = ?", id).Scan(&stats.DomainName)
	database.DB.QueryRow("SELECT COUNT(*) FROM emails WHERE domain_id = ?", id).Scan(&stats.TotalMails)
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE domain_id = ? AND received_at >= date('now')",
		id,
	).Scan(&stats.TodayMails)
	database.DB.QueryRow(
		"SELECT COUNT(DISTINCT recipient) FROM emails WHERE domain_id = ?", id,
	).Scan(&stats.UniqueAddr)

	c.JSON(http.StatusOK, stats)
}

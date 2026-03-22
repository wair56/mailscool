package handler

import (
	"database/sql"
	"mailer/database"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ===== API 邮件查询接口（API Key 鉴权）=====

func ApiListEmails(c *gin.Context) {
	domainIDs, _ := c.Get("allowed_domain_ids")
	ids, ok := domainIDs.([]int64)
	if !ok || len(ids) == 0 {
		c.JSON(http.StatusOK, database.PaginatedResponse{Data: []database.EmailListItem{}, Total: 0, Page: 1, Size: 20})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	// 构建查询
	where := "WHERE domain_id IN (" + buildPlaceholders(len(ids)) + ")"
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	// 可选筛选条件
	if to := c.Query("to"); to != "" {
		where += " AND recipient = ?"
		args = append(args, to)
	}
	if from := c.Query("from"); from != "" {
		where += " AND sender LIKE ?"
		args = append(args, "%"+from+"%")
	}
	if subject := c.Query("subject"); subject != "" {
		where += " AND subject LIKE ?"
		args = append(args, "%"+subject+"%")
	}
	if since := c.Query("since"); since != "" {
		where += " AND received_at >= ?"
		args = append(args, since)
	}
	if until := c.Query("until"); until != "" {
		where += " AND received_at <= ?"
		args = append(args, until)
	}

	// 总数
	var total int64
	database.DB.QueryRow("SELECT COUNT(*) FROM emails e "+where, args...).Scan(&total)

	// 分页查询
	queryArgs := append(args, size, offset)
	rows, err := database.DB.Query(
		"SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(extracted_code,''), has_attachments, is_read, COALESCE(is_starred,0), received_at FROM emails "+
			where+" ORDER BY received_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var emails []database.EmailListItem
	for rows.Next() {
		var e database.EmailListItem
		var hasAtt, isRead, isStarred int
		if err := rows.Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject, &e.ExtractedCode, &hasAtt, &isRead, &isStarred, &e.ReceivedAt); err != nil {
			continue
		}
		e.HasAttachments = hasAtt == 1
		e.IsRead = isRead == 1
		e.IsStarred = isStarred == 1
		emails = append(emails, e)
	}
	if emails == nil {
		emails = []database.EmailListItem{}
	}

	c.JSON(http.StatusOK, database.PaginatedResponse{
		Total: total,
		Page:  page,
		Size:  size,
		Data:  emails,
	})
}

func ApiGetEmail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	domainIDs, _ := c.Get("allowed_domain_ids")
	ids := domainIDs.([]int64)

	var e database.Email
	var hasAtt, isRead, isStarred int
	err := database.DB.QueryRow(
		`SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(body_text,''), COALESCE(body_html,''), 
		 COALESCE(extracted_code,''), COALESCE(extracted_links,''), has_attachments, raw_size, is_read, COALESCE(is_starred,0), received_at
		 FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject,
		&e.BodyText, &e.BodyHTML, &e.ExtractedCode, &e.ExtractedLinks,
		&hasAtt, &e.RawSize, &isRead, &isStarred, &e.ReceivedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查域名权限
	allowed := false
	for _, did := range ids {
		if did == e.DomainID {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	e.HasAttachments = hasAtt == 1
	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1

	// 标记已读
	database.DB.Exec("UPDATE emails SET is_read = 1 WHERE id = ?", id)

	c.JSON(http.StatusOK, e)
}

func ApiGetLatestEmail(c *gin.Context) {
	to := c.Query("to")
	if to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parameter 'to' is required"})
		return
	}

	domainIDs, _ := c.Get("allowed_domain_ids")
	ids := domainIDs.([]int64)

	where := "WHERE recipient = ? AND domain_id IN (" + buildPlaceholders(len(ids)) + ")"
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, to)
	for _, id := range ids {
		args = append(args, id)
	}

	// 可选：过滤时间范围（只看最近 N 分钟的邮件）
	if since := c.Query("since"); since != "" {
		where += " AND received_at >= ?"
		args = append(args, since)
	}

	var e database.Email
	var hasAtt, isRead, isStarred int
	err := database.DB.QueryRow(
		`SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(body_text,''), COALESCE(body_html,''),
		 COALESCE(extracted_code,''), COALESCE(extracted_links,''), has_attachments, raw_size, is_read, COALESCE(is_starred,0), received_at
		 FROM emails `+where+` ORDER BY received_at DESC LIMIT 1`,
		args...,
	).Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject,
		&e.BodyText, &e.BodyHTML, &e.ExtractedCode, &e.ExtractedLinks,
		&hasAtt, &e.RawSize, &isRead, &isStarred, &e.ReceivedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "no email found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	e.HasAttachments = hasAtt == 1
	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1

	c.JSON(http.StatusOK, e)
}

func ApiDeleteEmail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	domainIDs, _ := c.Get("allowed_domain_ids")
	ids := domainIDs.([]int64)

	// 检查邮件是否存在且有权限
	var domainID int64
	err := database.DB.QueryRow("SELECT domain_id FROM emails WHERE id = ?", id).Scan(&domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}

	allowed := false
	for _, did := range ids {
		if did == domainID {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	database.DB.Exec("DELETE FROM emails WHERE id = ?", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ===== Admin 邮件浏览接口 =====

func AdminListEmails(c *gin.Context) {
	role, _ := c.Get("role")
	adminID, _ := c.Get("admin_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	where := "WHERE 1=1"
	var args []interface{}

	// 非超管只能看自己管理的域名
	if role != "super_admin" {
		where += " AND e.domain_id IN (SELECT domain_id FROM admin_domains WHERE admin_id = ?)"
		args = append(args, adminID)
	}

	if domainID := c.Query("domain_id"); domainID != "" {
		where += " AND e.domain_id = ?"
		args = append(args, domainID)
	}
	if to := c.Query("to"); to != "" {
		where += " AND e.recipient = ?"
		args = append(args, to)
	}
	if from := c.Query("from"); from != "" {
		where += " AND e.sender LIKE ?"
		args = append(args, "%"+from+"%")
	}
	// 发件域名黑名单：排除特定域名的邮件
	if excludeDomains := c.Query("exclude_domains"); excludeDomains != "" {
		for _, d := range strings.Split(excludeDomains, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				where += " AND e.sender NOT LIKE ?"
				args = append(args, "%@"+d)
			}
		}
	}
	if c.Query("has_code") == "1" {
		where += " AND e.extracted_code != ''"
	}

	var total int64 = -1
	if c.Query("skip_count") != "1" {
		database.DB.QueryRow("SELECT COUNT(*) FROM emails e "+where, args...).Scan(&total)
	}

	queryArgs := append(args, size, offset)
	rows, err := database.DB.Query(
		`SELECT e.id, e.domain_id, e.recipient, e.sender, COALESCE(e.subject,''), COALESCE(e.extracted_code,''), 
		 e.has_attachments, e.is_read, COALESCE(e.is_starred,0), e.received_at 
		 FROM emails e `+where+` ORDER BY e.received_at DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var emails []database.EmailListItem
	for rows.Next() {
		var e database.EmailListItem
		var hasAtt, isRead, isStarred int
		if err := rows.Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject, &e.ExtractedCode, &hasAtt, &isRead, &isStarred, &e.ReceivedAt); err != nil {
			continue
		}
		e.HasAttachments = hasAtt == 1
		e.IsRead = isRead == 1
		e.IsStarred = isStarred == 1
		emails = append(emails, e)
	}
	if emails == nil {
		emails = []database.EmailListItem{}
	}

	c.JSON(http.StatusOK, database.PaginatedResponse{
		Total: total,
		Page:  page,
		Size:  size,
		Data:  emails,
	})
}

func AdminGetEmail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var e database.Email
	var hasAtt, isRead, isStarred int
	err := database.DB.QueryRow(
		`SELECT id, domain_id, recipient, sender, COALESCE(subject,''), COALESCE(body_text,''), COALESCE(body_html,''), 
		 COALESCE(extracted_code,''), COALESCE(extracted_links,''), has_attachments, raw_size, is_read, COALESCE(is_starred,0), received_at
		 FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.DomainID, &e.Recipient, &e.Sender, &e.Subject,
		&e.BodyText, &e.BodyHTML, &e.ExtractedCode, &e.ExtractedLinks,
		&hasAtt, &e.RawSize, &isRead, &isStarred, &e.ReceivedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}

	// 域名权限校验：非超管需检查邮件所属域名
	if !hasDomainAccess(c, e.DomainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	e.HasAttachments = hasAtt == 1
	e.IsRead = isRead == 1
	e.IsStarred = isStarred == 1

	database.DB.Exec("UPDATE emails SET is_read = 1 WHERE id = ?", id)
	c.JSON(http.StatusOK, e)
}

// helpers

func buildPlaceholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// ===== Toggle Star =====

func AdminToggleStar(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 域名权限校验
	var domainID int64
	err := database.DB.QueryRow("SELECT domain_id FROM emails WHERE id = ?", id).Scan(&domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}
	if !hasDomainAccess(c, domainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	database.DB.Exec("UPDATE emails SET is_starred = CASE WHEN COALESCE(is_starred,0) = 0 THEN 1 ELSE 0 END WHERE id = ?", id)

	var starred int
	database.DB.QueryRow("SELECT COALESCE(is_starred,0) FROM emails WHERE id = ?", id).Scan(&starred)
	c.JSON(http.StatusOK, gin.H{"is_starred": starred == 1})
}

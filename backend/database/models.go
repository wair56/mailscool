package database

import "time"

type Domain struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	IsActive       bool      `json:"is_active"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"created_at"`
	TotalApiKeys   int64     `json:"total_api_keys"`
	TotalMailboxes int64     `json:"total_mailboxes"`
	TotalEmails    int64     `json:"total_emails"`
}

type Email struct {
	ID             int64     `json:"id"`
	DomainID       int64     `json:"domain_id"`
	Recipient      string    `json:"recipient"`
	Sender         string    `json:"sender"`
	Subject        string    `json:"subject"`
	BodyText       string    `json:"body_text,omitempty"`
	BodyHTML       string    `json:"body_html,omitempty"`
	ExtractedCode  string    `json:"code,omitempty"`
	ExtractedLinks string    `json:"links,omitempty"`
	HasAttachments bool      `json:"has_attachments"`
	RawSize        int64     `json:"raw_size"`
	IsRead         bool      `json:"is_read"`
	IsStarred      bool      `json:"is_starred"`
	ReceivedAt     time.Time `json:"received_at"`
}

type EmailListItem struct {
	ID             int64     `json:"id"`
	DomainID       int64     `json:"domain_id"`
	Recipient      string    `json:"recipient"`
	Sender         string    `json:"sender"`
	Subject        string    `json:"subject"`
	ExtractedCode  string    `json:"code,omitempty"`
	HasAttachments bool      `json:"has_attachments"`
	IsRead         bool      `json:"is_read"`
	IsStarred      bool      `json:"is_starred"`
	ReceivedAt     time.Time `json:"received_at"`
}

type ApiKey struct {
	ID             int64      `json:"id"`
	KeyPrefix      string     `json:"key_prefix"`
	KeyHash        string     `json:"-"`
	Name           string     `json:"name"`
	IPWhitelist    string     `json:"ip_whitelist"`
	RateLimit      int        `json:"rate_limit"`
	IsActive       bool       `json:"is_active"`
	IsSystem       bool       `json:"is_system"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedBy      int64      `json:"created_by"`
	CreatedByName  string     `json:"created_by_name"`
	Domains        []Domain   `json:"domains,omitempty"`
	TotalEmails    int64      `json:"total_emails"`
	TotalMailboxes int64      `json:"total_mailboxes"`
}

type SenderRule struct {
	ID            int64  `json:"id"`
	DomainID      *int64 `json:"domain_id,omitempty"`
	SenderPattern string `json:"sender_pattern"`
	RuleType      string `json:"rule_type"` // "whitelist" or "blacklist"
}

type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "super_admin" or "admin"
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	DomainCount  int       `json:"domain_count"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	AdminID   *int64    `json:"admin_id,omitempty"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// Request/Response types

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Admin Admin  `json:"admin"`
}

type CreateDomainRequest struct {
	Name string `json:"name" binding:"required"`
	Note string `json:"note"`
}

type UpdateDomainRequest struct {
	Note     *string `json:"note"`
	IsActive *bool   `json:"is_active"`
}

type CreateApiKeyRequest struct {
	Name        string  `json:"name" binding:"required"`
	DomainIDs   []int64 `json:"domain_ids" binding:"required"`
	IPWhitelist string  `json:"ip_whitelist"`
	RateLimit   int     `json:"rate_limit"`
	ExpiresAt   string  `json:"expires_at"`
}

type CreateApiKeyResponse struct {
	Key    string `json:"key"` // 仅创建时返回明文
	ApiKey ApiKey `json:"api_key"`
}

type PaginatedResponse struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Data  interface{} `json:"data"`
}

type DomainStats struct {
	DomainID   int64  `json:"domain_id"`
	DomainName string `json:"domain_name"`
	TotalMails int64  `json:"total_emails"`
	TodayMails int64  `json:"today_emails"`
	UniqueAddr int64  `json:"unique_addresses"`
}

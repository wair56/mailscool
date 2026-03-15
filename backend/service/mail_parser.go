package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"

	"mailer/database"
)

// ParsedEmail 解析后的邮件结构
type ParsedEmail struct {
	From           string
	To             string
	Subject        string
	BodyText       string
	BodyHTML       string
	ExtractedCode  string
	ExtractedLinks []string
	HasAttachments bool
	RawSize        int64
}

// ParseEmail 从原始邮件数据解析邮件
func ParseEmail(raw []byte) (*ParsedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	parsed := &ParsedEmail{
		RawSize: int64(len(raw)),
	}

	// 解析头信息
	parsed.From = decodeHeader(msg.Header.Get("From"))
	parsed.To = decodeHeader(msg.Header.Get("To"))
	parsed.Subject = decodeHeader(msg.Header.Get("Subject"))

	// 解析正文
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// 尝试当纯文本处理
		body, _ := io.ReadAll(msg.Body)
		parsed.BodyText = string(body)
	} else if strings.HasPrefix(mediaType, "multipart/") {
		parseMultipart(msg.Body, params["boundary"], parsed)
	} else {
		cte := msg.Header.Get("Content-Transfer-Encoding")
		reader := decodeCTE(msg.Body, cte)
		body, _ := io.ReadAll(reader)
		decoded := decodeBody(body, contentType)
		if strings.Contains(mediaType, "html") {
			parsed.BodyHTML = decoded
		} else {
			parsed.BodyText = decoded
		}
	}

	// 提取验证码和链接
	textForExtraction := parsed.BodyText
	if textForExtraction == "" {
		textForExtraction = stripHTML(parsed.BodyHTML)
	}
	parsed.ExtractedCode = ExtractCode(textForExtraction)
	parsed.ExtractedLinks = ExtractLinks(textForExtraction)

	return parsed, nil
}

func parseMultipart(r io.Reader, boundary string, parsed *ParsedEmail) {
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}

		ct := part.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(ct)

		if strings.HasPrefix(mediaType, "multipart/") {
			parseMultipart(part, params["boundary"], parsed)
			continue
		}

		cte := part.Header.Get("Content-Transfer-Encoding")
		reader := decodeCTE(part, cte)
		body, err := io.ReadAll(reader)
		if err != nil {
			continue
		}

		disposition := part.Header.Get("Content-Disposition")
		if strings.Contains(disposition, "attachment") {
			parsed.HasAttachments = true
			continue
		}

		decoded := decodeBody(body, ct)
		switch {
		case strings.Contains(mediaType, "text/plain") && parsed.BodyText == "":
			parsed.BodyText = decoded
		case strings.Contains(mediaType, "text/html") && parsed.BodyHTML == "":
			parsed.BodyHTML = decoded
		}
	}
}

func decodeCTE(r io.Reader, cte string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, r)
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	default:
		return r
	}
}

func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

func decodeBody(body []byte, contentType string) string {
	// 使用 charset 包自动检测并转换编码
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		// charset.NewReader failed — try to detect encoding from BOM or content
		// If content looks like valid UTF-8, use as-is
		if isValidUTF8(body) {
			return string(body)
		}
		// Try common fallback: GBK (most common non-UTF-8 Chinese encoding)
		reader, err = charset.NewReader(bytes.NewReader(body), "text/plain; charset=gbk")
		if err != nil {
			return string(body)
		}
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return string(body)
	}
	// Verify result is valid UTF-8; if not, return raw
	if !isValidUTF8(decoded) {
		return string(body)
	}
	return string(decoded)
}

func isValidUTF8(b []byte) bool {
	return utf8.Valid(b)
}

func stripHTML(html string) string {
	// 在块级标签处插入换行，确保验证码等内容出现在独立行
	blockRe := regexp.MustCompile(`(?i)<\s*/?\s*(?:p|div|br|tr|td|th|li|h[1-6]|blockquote|section|article|header|footer)\b[^>]*>`)
	html = blockRe.ReplaceAllString(html, "\n")

	// Strip all remaining tags
	var result strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// StoreEmail 存储解析后的邮件到数据库
func StoreEmail(parsed *ParsedEmail, domainID int64) (int64, error) {
	linksJSON := "[]"
	if len(parsed.ExtractedLinks) > 0 {
		b, _ := json.Marshal(parsed.ExtractedLinks)
		linksJSON = string(b)
	}

	// Normalize recipient: extract pure email address from "Name <email>" format
	recipient := parsed.To
	if idx := strings.Index(recipient, "<"); idx >= 0 {
		recipient = recipient[idx+1:]
		if idx2 := strings.Index(recipient, ">"); idx2 >= 0 {
			recipient = recipient[:idx2]
		}
	}
	recipient = strings.TrimSpace(strings.ToLower(recipient))

	result, err := database.DB.Exec(
		`INSERT INTO emails (domain_id, recipient, sender, subject, body_text, body_html, 
		 extracted_code, extracted_links, has_attachments, raw_size, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		domainID, recipient, parsed.From, parsed.Subject,
		parsed.BodyText, parsed.BodyHTML,
		parsed.ExtractedCode, linksJSON,
		boolToInt(parsed.HasAttachments), parsed.RawSize,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to store email: %w", err)
	}

	id, _ := result.LastInsertId()
	log.Printf("Stored email #%d: %s -> %s [%s]", id, parsed.From, parsed.To, parsed.Subject)
	return id, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

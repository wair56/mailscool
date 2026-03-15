package service

import (
	"regexp"
	"strings"
)

// 纯数字验证码正则（4-8位）
var numericCodePatterns = []*regexp.Regexp{
	// 中文关键词（支持跨行）
	regexp.MustCompile(`(?i)(?:验证码|校验码|确认码|安全码|动态码|一次性验证码|一次性密码)[：:是为\s]*[\r\n\s]*(\d{4,8})`),
	// 日语关键词
	regexp.MustCompile(`(?i)(?:認証コード|確認コード|セキュリティコード|ワンタイムパスワード|パスコード)[：:は\s]*[\r\n\s]*(\d{4,8})`),
	// 韩语关键词
	regexp.MustCompile(`(?i)(?:인증\s*코드|확인\s*코드|보안\s*코드|일회용\s*비밀번호)[：:은는\s]*[\r\n\s]*(\d{4,8})`),
	// 英文关键词
	regexp.MustCompile(`(?i)(?:verification|confirm|security|auth)\s*(?:code|pin|number)(?:\s+is)?[:\s]*[\r\n\s]*(\d{4,8})`),
	regexp.MustCompile(`(?i)(?:otp|one[-\s]?time)\s*(?:password|code|pin)?(?:\s+is)?[:\s]*[\r\n\s]*(\d{4,8})`),
	regexp.MustCompile(`(?i)(?:code|pin|passcode)\s*(?:is|:)\s*[\r\n\s]*(\d{4,8})`),
	// 独立行纯数字
	regexp.MustCompile(`(?m)^\s*(\d{4,8})\s*$`),
}

// 字母+数字混合验证码正则（4-10位）— 紧邻模式（高优先级）
var alphanumCodeTightPatterns = []*regexp.Regexp{
	// 中文关键词 + 字母数字混合（紧邻）
	regexp.MustCompile(`(?i)(?:验证码|校验码|确认码|安全码|动态码|一次性验证码|一次性密码)[：:是为\s]*[\r\n\s]*([A-Z0-9]{4,10})`),
	// 日语关键词
	regexp.MustCompile(`(?i)(?:認証コード|確認コード|セキュリティコード|ワンタイムパスワード|パスコード)[：:は\s]*[\r\n\s]*([A-Z0-9]{4,10})`),
	// 韩语关键词
	regexp.MustCompile(`(?i)(?:인증\s*코드|확인\s*코드|보안\s*코드|일회용\s*비밀번호)[：:은는\s]*[\r\n\s]*([A-Z0-9]{4,10})`),
	// 英文关键词
	regexp.MustCompile(`(?i)(?:verification|confirm|security|auth)\s*(?:code|pin|number)(?:\s+is)?[:\s]*[\r\n\s]*([A-Z0-9]{4,10})`),
	regexp.MustCompile(`(?i)(?:otp|one[-\s]?time)\s*(?:password|code|pin)?(?:\s+is)?[:\s]*[\r\n\s]*([A-Z0-9]{4,10})`),
	regexp.MustCompile(`(?i)(?:code|pin|passcode)\s*(?:is|:)\s*[\r\n\s]*([A-Z0-9]{4,10})`),
}

// 独立行字母数字混合正则（兜底，单独处理）
var alphanumStandaloneLine = regexp.MustCompile(`(?mi)^\s*([A-Za-z0-9]{4,10})\s*$`)

// 宽松模式：关键词和验证码之间有间隔文本（低优先级）
var alphanumCodeLoosePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?si)(?:验证码|校验码|确认码|安全码|动态码|一次性验证码|一次性密码).{0,300}?\b([A-Za-z0-9]{4,10})\b`),
	regexp.MustCompile(`(?si)(?:verification|confirm|security|auth)\s*(?:code|pin|number).{0,300}?\b([A-Za-z0-9]{4,10})\b`),
}

// 常见假阳性，排除这些值
var falsePositives = map[string]bool{
	"http": true, "https": true, "html": true, "text": true,
	"from": true, "date": true, "subject": true, "email": true,
	"mail": true, "spam": true, "inbox": true, "sent": true,
	"your": true, "this": true, "that": true, "with": true,
	"have": true, "will": true, "been": true, "please": true,
	"click": true, "here": true, "link": true, "button": true,
	"font": true, "size": true, "color": true, "style": true,
	"width": true, "height": true, "table": true, "image": true,
	"spaceship": true, "alert": true, "account": true,
	"phoenix": true, "street": true, "suite": true,
	"minute": true, "after": true, "before": true, "about": true,
	"just": true, "copy": true, "paste": true, "form": true,
	"wair56": true, "rosetta": true,
}

// isLikelyCode 判断提取出的值是否像验证码
func isLikelyCode(s string) bool {
	lower := strings.ToLower(s)
	if falsePositives[lower] {
		return false
	}
	if strings.Contains(lower, "@") || strings.Contains(lower, ".") {
		return false
	}
	return hasLetterAndDigit(s)
}

// hasLetterAndDigit 检查字符串是否同时包含字母和数字
func hasLetterAndDigit(s string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// URL 提取正则
var urlPattern = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `\x{200b}\x{200c}\x{200d}\x{feff}]+`)

// ExtractCode 从邮件正文中提取验证码
func ExtractCode(text string) string {
	// 1. 优先匹配纯数字验证码（最常见）
	for _, p := range numericCodePatterns {
		if m := p.FindStringSubmatch(text); len(m) > 1 {
			return m[1]
		}
	}
	// 2. 紧邻模式匹配字母+数字混合（关键词后直接跟验证码）
	for _, p := range alphanumCodeTightPatterns {
		if m := p.FindStringSubmatch(text); len(m) > 1 {
			code := m[1]
			if isLikelyCode(code) {
				return code
			}
		}
	}
	// 3. 独立行兜底（优于宽松模式，独立行是很强的信号）
	if matches := alphanumStandaloneLine.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, m := range matches {
			code := m[1]
			if isLikelyCode(code) {
				return code
			}
		}
	}
	// 4. 宽松模式（关键词和验证码中间有间隔文本）
	for _, p := range alphanumCodeLoosePatterns {
		allMatches := p.FindAllStringSubmatch(text, 5)
		for _, m := range allMatches {
			if len(m) > 1 {
				code := m[1]
				if isLikelyCode(code) {
					return code
				}
			}
		}
	}
	return ""
}

// ExtractLinks 从邮件正文中提取链接
func ExtractLinks(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	seen := make(map[string]bool)
	var result []string
	for _, u := range matches {
		u = strings.TrimRight(u, ".,;:!?)")
		if !seen[u] {
			seen[u] = true
			result = append(result, u)
		}
	}
	return result
}

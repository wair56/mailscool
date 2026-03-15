package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mailer/config"
	"mailer/database"
	"net/http"
	"strings"
	"time"
)

// WebhookPayload 推送到用户 Webhook 的数据结构
type WebhookPayload struct {
	Event   string       `json:"event"`
	Mailbox string       `json:"mailbox"`
	Email   WebhookEmail `json:"email"`
}

type WebhookEmail struct {
	From       string `json:"from"`
	Subject    string `json:"subject"`
	Code       string `json:"code,omitempty"`
	BodyText   string `json:"body_text,omitempty"`
	ReceivedAt string `json:"received_at"`
}

// NotifyMailbox 收到新邮件时，查询邮箱的 webhook_url 和 telegram_chat_id 并推送
func NotifyMailbox(email string, parsed *ParsedEmail) {
	var webhookURL string
	var telegramChatID int64
	err := database.DB.QueryRow(
		"SELECT COALESCE(webhook_url,''), COALESCE(telegram_chat_id,0) FROM mailboxes WHERE email = ?", email,
	).Scan(&webhookURL, &telegramChatID)
	
	log.Printf("[webhook] NotifyMailbox for %s: url=%s, tg=%d, err=%v", email, webhookURL, telegramChatID, err)
	
	if err != nil {
		return // 邮箱不存在或查询失败
	}

	// Webhook 推送
	if webhookURL != "" {
		go pushWebhook(webhookURL, email, parsed)
	}

	// TG 推送
	if telegramChatID != 0 {
		tgToken := database.GetSetting("telegram_bot_token", "")
		if tgToken != "" {
			go pushTelegram(tgToken, telegramChatID, email, parsed)
		}
	}
}

func pushWebhook(url, email string, parsed *ParsedEmail) {
	var body []byte
	
	codeStr := ""
	if parsed.ExtractedCode != "" {
		codeStr = fmt.Sprintf("\n🔑 Code: %s", parsed.ExtractedCode)
	}
	// Build login URL
	baseURL := database.GetSetting("base_url", "")
	if baseURL == "" {
		baseURL = "https://mails.cool"
	}
	var encPwd string
	pwd := ""
	if e := database.DB.QueryRow("SELECT password_plain FROM mailboxes WHERE email = ?", email).Scan(&encPwd); e == nil {
		pwd, _ = config.Decrypt(encPwd)
	}
	loginURL := fmt.Sprintf("%s/login?email=%s&password=%s", baseURL, email, pwd)

	if strings.Contains(url, "open.feishu.cn") || strings.Contains(url, "open.larksuite.com") {
		// 飞书机器人格式
		feishuPayload := map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": fmt.Sprintf("📨 新邮件到达\n📧 收件人: %s\n🧑 发件人: %s\n📝 主题: %s%s\n\n🔗 网页端查看: %s", 
					email, parsed.From, parsed.Subject, codeStr, loginURL),
			},
		}
		body, _ = json.Marshal(feishuPayload)
	} else if strings.Contains(url, "oapi.dingtalk.com") {
		// 钉钉机器人格式
		dingPayload := map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": fmt.Sprintf("📨 新邮件到达\n📧 收件人: %s\n🧑 发件人: %s\n📝 主题: %s%s\n\n🔗 网页端查看: %s", 
					email, parsed.From, parsed.Subject, codeStr, loginURL),
			},
		}
		body, _ = json.Marshal(dingPayload)
	} else if strings.Contains(url, "qyapi.weixin.qq.com") {
		// 企业微信机器人格式
		wecomPayload := map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": fmt.Sprintf("📨 新邮件到达\n📧 收件人: %s\n🧑 发件人: %s\n📝 主题: %s%s\n\n🔗 网页端查看: %s", 
					email, parsed.From, parsed.Subject, codeStr, loginURL),
			},
		}
		body, _ = json.Marshal(wecomPayload)
	} else if strings.Contains(url, "hooks.slack.com") {
		// Slack 机器人格式
		slackPayload := map[string]interface{}{
			"text": fmt.Sprintf("📨 *New Email*\n📧 To: %s\n🧑 From: %s\n📝 Subject: %s%s\n\n🔗 [Open Inbox](%s)", 
				email, parsed.From, parsed.Subject, codeStr, loginURL),
		}
		body, _ = json.Marshal(slackPayload)
	} else if strings.Contains(url, "discord.com/api/webhooks") {
		// Discord 机器人格式
		discordPayload := map[string]interface{}{
			"content": fmt.Sprintf("📨 **New Email**\n📧 To: `%s`\n🧑 From: `%s`\n📝 Subject: %s%s\n\n🔗 [Open Inbox](%s)", 
				email, parsed.From, parsed.Subject, codeStr, loginURL),
		}
		body, _ = json.Marshal(discordPayload)
	} else {
		// 标准格式
		payload := WebhookPayload{
			Event:   "new_email",
			Mailbox: email,
			Email: WebhookEmail{
				From:       parsed.From,
				Subject:    parsed.Subject,
				Code:       parsed.ExtractedCode,
				BodyText:   parsed.BodyText,
				ReceivedAt: time.Now().Format(time.RFC3339),
			},
		}
		body, _ = json.Marshal(payload)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[webhook] push failed for %s -> %s: %v", email, url, err)
		return
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("[webhook] push %s -> %s failed %d: %s", email, url, resp.StatusCode, string(respBody))
	} else {
		log.Printf("[webhook] push %s -> %s success: %s", email, url, string(respBody))
	}
}

func pushTelegram(token string, chatID int64, email string, parsed *ParsedEmail) {
	codeStr := ""
	if parsed.ExtractedCode != "" {
		codeStr = fmt.Sprintf("\n🔑 Code: `%s`", parsed.ExtractedCode)
	}

	// Build login URL
	baseURL := database.GetSetting("base_url", "")
	if baseURL == "" {
		baseURL = "https://mails.cool"
	}
	var encPwd string
	pwd := ""
	if e := database.DB.QueryRow("SELECT password_plain FROM mailboxes WHERE email = ?", email).Scan(&encPwd); e == nil {
		pwd, _ = config.Decrypt(encPwd)
	}
	loginURL := fmt.Sprintf("%s/login?email=%s&password=%s", baseURL, email, pwd)

	sendMessage(token, chatID, fmt.Sprintf(
		"📨 *New Email*\n📧 To: `%s`\n✉️ From: %s\n📝 %s%s\n\n🔗 [Open Inbox](%s)",
		email, truncate(parsed.From, 40), truncate(parsed.Subject, 50), codeStr, loginURL))
}

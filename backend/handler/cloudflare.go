package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mailer/config"
	"mailer/database"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ===== Cloudflare 一键配置 =====

const cfAPI = "https://api.cloudflare.com/client/v4"

// cfRequest performs a CF API request
func cfRequest(method, path, token string, body io.Reader, contentType string) (map[string]interface{}, error) {
	req, err := http.NewRequest(method, cfAPI+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("CF API returned invalid JSON: %s", string(respBody[:min(200, len(respBody))]))
	}

	success, _ := result["success"].(bool)
	if !success {
		errorsField, _ := result["errors"].([]interface{})
		if len(errorsField) > 0 {
			errMsg, _ := errorsField[0].(map[string]interface{})
			msg, _ := errMsg["message"].(string)
			return nil, fmt.Errorf("CF API error: %s", msg)
		}
		return nil, fmt.Errorf("CF API failed: %s", string(respBody[:min(300, len(respBody))]))
	}

	return result, nil
}



// CloudflareSetup handles one-click CF Email Routing + Worker setup
func CloudflareSetup(c *gin.Context) {
	domainID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// Check domain access
	if !hasDomainAccess(c, domainID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access to this domain"})
		return
	}

	// Get domain name
	var domainName string
	err := database.DB.QueryRow("SELECT name FROM domains WHERE id = ?", domainID).Scan(&domainName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Parse request
	var req struct {
		CFToken    string `json:"cf_token"`
		ReceiveURL string `json:"receive_url"` // optional override
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CFToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cf_token is required"})
		return
	}

	// Default receive URL - prefer mailer-api subdomain (via Tunnel, bypasses WAF)
	receiveURL := req.ReceiveURL
	if receiveURL == "" {
		host := c.Request.Host
		// Strip port if present
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}
		// Use mailer-api. prefix to route via Tunnel instead of CF proxy
		// e.g. mails.cool -> mailer-api.mails.cool
		receiveURL = fmt.Sprintf("https://mailer-api.%s/api/receive", host)
	}

	steps := []gin.H{}
	addStep := func(name, status string, err error) {
		s := gin.H{"step": name, "status": status}
		if err != nil {
			s["error"] = err.Error()
		}
		steps = append(steps, s)
	}

	// Step 1: Get account ID
	result, err := cfRequest("GET", "/accounts?per_page=1", req.CFToken, nil, "")
	if err != nil {
		addStep("verify_token", "failed", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CF API Token", "steps": steps})
		return
	}
	accounts, _ := result["result"].([]interface{})
	if len(accounts) == 0 {
		addStep("verify_token", "failed", fmt.Errorf("no accounts found"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "No CF accounts found", "steps": steps})
		return
	}
	account, _ := accounts[0].(map[string]interface{})
	accountID, _ := account["id"].(string)
	addStep("verify_token", "ok", nil)

	// Step 2: Get zone ID for domain
	result, err = cfRequest("GET", fmt.Sprintf("/zones?name=%s", domainName), req.CFToken, nil, "")
	if err != nil {
		addStep("find_zone", "failed", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find zone", "steps": steps})
		return
	}
	zones, _ := result["result"].([]interface{})
	if len(zones) == 0 {
		addStep("find_zone", "failed", fmt.Errorf("domain %s not found in Cloudflare", domainName))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Domain %s not found in your CF account", domainName), "steps": steps})
		return
	}
	zone, _ := zones[0].(map[string]interface{})
	zoneID, _ := zone["id"].(string)
	zoneStatus, _ := zone["status"].(string)

	// Preflight: Check zone NS status
	if zoneStatus == "pending" {
		addStep("find_zone", "warning", fmt.Errorf("NS not yet active, Email Routing may not work until NS or MX are configured"))
	} else if zoneStatus != "active" {
		addStep("find_zone", "warning", fmt.Errorf("zone status: %s", zoneStatus))
	} else {
		addStep("find_zone", "ok", nil)
	}

	// Preflight: Check if Email Routing DNS records exist
	emailRoutingResult, _ := cfRequest("GET", fmt.Sprintf("/zones/%s/email/routing", zoneID), req.CFToken, nil, "")
	if emailRoutingResult != nil {
		if erResult, ok := emailRoutingResult["result"].(map[string]interface{}); ok {
			if enabled, ok := erResult["enabled"].(bool); ok && enabled {
				addStep("check_email_routing", "ok", nil)
			} else {
				addStep("check_email_routing", "ok", fmt.Errorf("will be enabled"))
			}
		}
	}

	// Step 3: Enable Email Routing (skip if already enabled)
	alreadyEnabled := false
	if emailRoutingResult != nil {
		if erResult, ok := emailRoutingResult["result"].(map[string]interface{}); ok {
			if enabled, ok := erResult["enabled"].(bool); ok && enabled {
				alreadyEnabled = true
			}
		}
	}
	if alreadyEnabled {
		addStep("enable_email_routing", "ok", nil)
	} else {
		_, err = cfRequest("POST", fmt.Sprintf("/zones/%s/email/routing/enable", zoneID), req.CFToken, nil, "application/json")
		if err != nil {
			// Not fatal - might need manual enable via CF dashboard, or already enabled
			addStep("enable_email_routing", "warning", fmt.Errorf("%s (please enable Email Routing manually in CF dashboard if needed)", err.Error()))
		} else {
			addStep("enable_email_routing", "ok", nil)
		}
	}

	// Step 4: Create/Reuse API Key for this domain's worker
	keyName := fmt.Sprintf("CF Worker - %s", domainName)
	var existingKeyPlain string
	err = database.DB.QueryRow("SELECT key_plain FROM api_keys WHERE name = ?", keyName).Scan(&existingKeyPlain)
	var apiKey string
	if err == nil && existingKeyPlain != "" {
		if decrypted, err := config.Decrypt(existingKeyPlain); err == nil {
			apiKey = decrypted
		} else {
			apiKey = existingKeyPlain
		}
		// Ensure domain binding exists for reused key
		var existingKeyID int64
		database.DB.QueryRow("SELECT id FROM api_keys WHERE name = ?", keyName).Scan(&existingKeyID)
		if existingKeyID > 0 {
			database.DB.Exec("INSERT OR IGNORE INTO api_key_domains (api_key_id, domain_id) VALUES (?, ?)", existingKeyID, domainID)
		}
		addStep("create_api_key", "ok", fmt.Errorf("reused existing key"))
	} else {
		randomBytes := make([]byte, 16)
		rand.Read(randomBytes)
		apiKey = "sk_" + hex.EncodeToString(randomBytes)
		prefix := apiKey[:11]
		hash, _ := bcrypt.GenerateFromPassword([]byte(apiKey), 10)
		encryptedKey, _ := config.Encrypt(apiKey)
		adminID, _ := c.Get("admin_id")
		result2, err := database.DB.Exec(
			`INSERT INTO api_keys (key_prefix, key_hash, key_plain, name, rate_limit, is_system, created_by) VALUES (?, ?, ?, ?, ?, 1, ?)`,
			prefix, string(hash), encryptedKey, keyName, 100, adminID,
		)
		if err != nil {
			addStep("create_api_key", "failed", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create API key", "steps": steps})
			return
		}
		keyID, _ := result2.LastInsertId()
		database.DB.Exec("INSERT INTO api_key_domains (api_key_id, domain_id) VALUES (?, ?)", keyID, domainID)
		addStep("create_api_key", "ok", nil)
	}

	// Step 5: Create/Update Email Worker
	workerName := fmt.Sprintf("mailer-email-%s", strings.ReplaceAll(domainName, ".", "-"))
	workerScript := fmt.Sprintf(`export default {
  async email(message, env) {
    const to = message.to;
    const from = message.from;
    console.log("--- Email received / 收到邮件 ---");
    console.log("From / 发件人: " + from + " → To / 收件人: " + to);
    const raw = new Response(message.raw);
    const body = await raw.arrayBuffer();
    const resp = await fetch("%s", {
      method: "POST",
      headers: {
        "Authorization": "Bearer %s",
        "Content-Type": "application/octet-stream",
        "X-Envelope-To": to
      },
      body: body
    });
    const text = await resp.text();
    if (resp.ok) {
      console.log("✅ Forwarded / 转发成功 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
    } else {
      console.log("⚠ Forward failed / 转发失败 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
      console.log("Server: " + (resp.headers.get("server") || "unknown"));
      if (resp.status === 401 || resp.status === 403) {
        console.log("Diagnosis / 诊断: Invalid or expired API Key / API Key 无效或过期");
      } else if (text.includes("<!DOCTYPE") || text.includes("cloudflare")) {
        console.log("Diagnosis / 诊断: Blocked by Cloudflare WAF / 被 CF WAF 拦截，未到达服务器");
      }
    }
  }
}`, receiveURL, apiKey)

	// Use multipart form for worker upload
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// metadata part
	metadata := map[string]interface{}{
		"main_module":       "worker.js",
		"compatibility_date": "2024-01-01",
	}
	metaJSON, _ := json.Marshal(metadata)
	metaPart, _ := w.CreateFormField("metadata")
	metaPart.Write(metaJSON)

	// script part
	scriptHeader := make(map[string][]string)
	scriptHeader["Content-Disposition"] = []string{`form-data; name="worker.js"; filename="worker.js"`}
	scriptHeader["Content-Type"] = []string{"application/javascript+module"}
	scriptPart, _ := w.CreatePart(scriptHeader)
	scriptPart.Write([]byte(workerScript))
	w.Close()

	_, err = cfRequest("PUT",
		fmt.Sprintf("/accounts/%s/workers/scripts/%s", accountID, workerName),
		req.CFToken, &buf, w.FormDataContentType())
	if err != nil {
		addStep("create_worker", "failed", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Worker", "steps": steps})
		return
	}
	addStep("create_worker", "ok", nil)

	// Step 5: Configure catch-all rule to send to worker
	catchAllBody := map[string]interface{}{
		"enabled": true,
		"actions": []map[string]interface{}{
			{
				"type":  "worker",
				"value": []string{workerName},
			},
		},
		"matchers": []map[string]interface{}{
			{
				"type": "all",
			},
		},
	}
	catchAllJSON, _ := json.Marshal(catchAllBody)

	_, err = cfRequest("PUT",
		fmt.Sprintf("/zones/%s/email/routing/rules/catch_all", zoneID),
		req.CFToken, bytes.NewReader(catchAllJSON), "application/json")
	if err != nil {
		addStep("configure_catchall", "failed", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure catch-all", "steps": steps})
		return
	}
	addStep("configure_catchall", "ok", nil)

	// Step 6: Create DMARC record if not exists
	dmarcResult, _ := cfRequest("GET",
		fmt.Sprintf("/zones/%s/dns_records?type=TXT&name=_dmarc.%s", zoneID, domainName),
		req.CFToken, nil, "application/json")
	dmarcExists := false
	if dmarcResult != nil {
		if records, ok := dmarcResult["result"].([]interface{}); ok && len(records) > 0 {
			dmarcExists = true
		}
	}
	if dmarcExists {
		addStep("create_dmarc", "ok", fmt.Errorf("already exists"))
	} else {
		dmarcBody := map[string]interface{}{
			"type":    "TXT",
			"name":    fmt.Sprintf("_dmarc.%s", domainName),
			"content": "v=DMARC1; p=none;",
			"ttl":     1,
		}
		dmarcJSON, _ := json.Marshal(dmarcBody)
		_, err = cfRequest("POST",
			fmt.Sprintf("/zones/%s/dns_records", zoneID),
			req.CFToken, bytes.NewReader(dmarcJSON), "application/json")
		if err != nil {
			addStep("create_dmarc", "warning", err)
		} else {
			addStep("create_dmarc", "ok", nil)
		}
	}

	// Save worker name to domain record for reference
	database.DB.Exec("UPDATE domains SET note = ? WHERE id = ?",
		fmt.Sprintf("CF Worker: %s", workerName), domainID)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Cloudflare Email Routing configured successfully",
		"worker_name": workerName,
		"api_key":     apiKey,
		"api_key_name": keyName,
		"steps":       steps,
	})
}

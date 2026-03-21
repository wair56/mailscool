# 📮 MailsCool

**The Self-Hosted Email API You Actually Own** | **真正属于你的自托管邮件 API**

> Stop renting email inboxes. Deploy once, own every email that hits your domain — forever.
>
> 不再租用邮箱。一次部署，永久拥有命中你域名的每一封邮件。

[English](#-why-mailscool) | [中文](#-为什么选择-mailscool)

---

## 🎯 Why MailsCool?

**The Problem**: You need to receive emails programmatically — for account verification, email-based automation, or disposable inboxes. Existing solutions force you to:

| Pain Point             | Existing Services                      | MailsCool                                       |
| ---------------------- | -------------------------------------- | ----------------------------------------------- |
| **Pricing**            | $15-50+/mo per domain, pay-per-email   | **Free forever** (self-hosted)                  |
| **Rate Limits**        | 100-500 emails/day on free tiers       | **Unlimited** — your server, your rules         |
| **Data Privacy**       | Emails stored on 3rd-party servers     | **Your server, your data, zero logging**        |
| **Vendor Lock-in**     | Proprietary APIs, migration nightmares | **Open source**, standard REST API              |
| **Verification Codes** | Manual parsing or regex hacks          | **Auto-extracted** in 10+ languages             |
| **Setup Complexity**   | MX records + SMTP + TLS certs          | **One-click** Cloudflare setup, zero SMTP       |
| **Catch-all**          | Often paid add-on or unsupported       | **Built-in** — any address, no pre-registration |
| **Multi-tenant**       | Separate accounts per domain           | **One instance** manages all domains            |

### 🚀 Core Value / 核心价值

1. **Zero-SMTP Architecture** — No Postfix, no MX records to debug. Cloudflare Email Routing + Worker handles everything. Your server only receives HTTP POST.

2. **Instant Verification Codes** — Every incoming email is auto-parsed for codes (SMS-style 4-8 digit, alphanumeric, and link-based). Perfect for automated account registration flows.

3. **One API Call = One Temp Mailbox** — `POST /api/mailbox/create` returns a ready-to-use email + password in milliseconds. No waiting, no pre-provisioning.

4. **True Catch-all** — `anything@your-domain.com` works instantly. No mailbox creation needed. Every email is captured and queryable via API.

5. **Self-Hosted = Unlimited** — No per-email fees, no daily limits, no "upgrade to pro" walls. Run it on a $5 VPS.

---

## 🎯 为什么选择 MailsCool？

**痛点**：你需要通过程序接收邮件 — 用于账号验证、邮件自动化或一次性邮箱。但现有方案让你不得不面对：

| 痛点           | 现有服务（Mailgun/Temp-mail等） | MailsCool                         |
| -------------- | ------------------------------- | --------------------------------- |
| **费用**       | 每域名 $15-50+/月，按信计费     | **永久免费**（自托管）            |
| **速率限制**   | 免费版每天 100-500 封           | **无限制** — 你的服务器，你做主   |
| **数据隐私**   | 邮件存储在第三方服务器          | **你的服务器，你的数据，零日志**  |
| **供应商锁定** | 专有 API，迁移噩梦              | **开源**，标准 REST API           |
| **验证码提取** | 手动解析或正则                  | **自动提取** 10+ 种语言           |
| **配置复杂度** | MX 记录 + SMTP + TLS 证书       | **一键** Cloudflare 配置，零 SMTP |
| **Catch-all**  | 通常是付费附加功能              | **内置** — 任意地址，无需预创建   |
| **多租户**     | 每个域名独立账号                | **一个实例** 管理所有域名         |

### 🚀 核心价值

1. **零 SMTP 架构** — 无需 Postfix，无需调试 MX 记录。Cloudflare Email Routing + Worker 搞定一切。你的服务器只接收 HTTP POST。

2. **秒级验证码提取** — 自动解析每封来信中的验证码（4-8位数字、字母数字混合、链接型）。完美适配自动注册场景。

3. **一个 API = 一个临时邮箱** — `POST /api/mailbox/create` 毫秒级返回可用邮箱+密码。无等待，无预配置。

4. **真正的 Catch-all** — `任意名@你的域名.com` 即时生效。无需创建邮箱，每封邮件可通过 API 查询。

5. **自托管 = 无限制** — 无按邮件收费，无每日限额，无"升级 Pro"付费墙。$5 VPS 即可运行。

---

## ✨ Full Features / 完整功能列表

<details>
<summary>📋 Click to expand / 点击展开</summary>

### English

- **Catch-all Receiving** — Any address under your domain receives email automatically, no pre-registration needed
- **Code Extraction** — Auto-detect verification codes in emails (supports multilingual formats)
- **Link Extraction** — Auto-extract all URLs from email body
- **Multi-Domain** — One instance manages multiple domains with DNS health checks
- **Cloudflare Auto Setup** — One-click Worker + Catch-all configuration with CF API Token, fully idempotent & retryable
- **API Key Auth** — Multiple keys, IP whitelist, rate limiting, domain-level permissions
- **System API Keys** — CF Worker keys marked as 🔒 system keys with delete protection warnings
- **Create Mailbox via API** — Programmatically create temp mailboxes with realistic human-like names
- **Mailbox System** — Permanent & temporary mailboxes, users can log in to view inbox
- **Temp Mailbox Domain Config** — Admin selects which domains are available for temp mailbox creation
- **Mailbox Renewal** — Temp mailbox users can extend expiry by 3 months with one click
- **Email Starring** — Star important emails, auto-cleanup skips starred messages
- **Multi-Admin** — Super admin + domain admins, domain-based permission isolation
- **Audit Logging** — Track all critical operations
- **Auto Cleanup** — Configurable email retention, starred emails exempted
- **Bilingual UI** — Full Chinese/English interface with one-click switch
- **Public API Docs** — Interactive API documentation accessible without login
- **Privacy First** — No tracking, no third-party services
- **Google Analytics** — Optional GA4 integration

### 中文

- **Catch-all 收件** — 域名下任意地址自动收件，无需预创建邮箱
- **验证码提取** — 自动识别邮件中的验证码（中英日韩德法西等多语言）
- **链接提取** — 自动提取邮件正文中的 URL
- **多域名管理** — 一个实例管理多个域名，支持 DNS 状态检测
- **Cloudflare 一键配置** — 填入 CF API Token，自动创建 Worker + 配置 Catch-all
- **API Key 鉴权** — 多 Key、IP 白名单、速率限制、域名级权限
- **系统 API Key** — CF Worker 的 Key 标记为 🔒 系统 Key
- **API 创建邮箱** — 毫秒级创建临时邮箱，真人风格邮箱名
- **邮箱系统** — 永久邮箱 + 临时邮箱，用户可登录查看收件箱
- **临时邮箱域名配置** — 管理员选定可用域名
- **邮箱续期** — 一键续期 3 个月
- **邮件星标** — 自动清理时跳过星标邮件
- **多管理员** — 超管 + 域名管理员权限隔离
- **审计日志** — 记录所有关键操作
- **自动清理** — 可配置保留天数
- **双语界面** — 中英文一键切换
- **公开 API 文档** — 无需登录即可查看
- **隐私优先** — 无追踪、无第三方服务
- **Google Analytics** — 可选 GA4 集成

</details>

## 🏗 Tech Stack / 技术栈

| Component | Technology                                      |
| --------- | ----------------------------------------------- |
| Backend   | Go + Gin (single binary with embedded frontend) |
| Frontend  | Vue 3 + Naive UI (bilingual i18n)               |
| Database  | SQLite (WAL mode, 64MB cache)                   |
| Receiving | Cloudflare Email Routing + Worker (HTTP POST)   |
| Deploy    | Docker (single container) / Bare metal          |
| HTTPS     | Cloudflare Tunnel                               |

## 🚀 Quick Start / 快速开始

### 1. Requirements / 准备

- A Linux server (1C1G is sufficient) / 一台 Linux 服务器
- Docker + Docker Compose
- Domain hosted on Cloudflare / 域名托管在 Cloudflare

### 2. Deploy / 部署

```bash
git clone https://github.com/wair56/mailer.git
cd mailer
cp .env.example .env
vim .env  # Set JWT_SECRET at minimum / 至少设置 JWT_SECRET

docker compose up -d --build
```

First-time startup creates a super admin (password shown in logs only once):

```bash
docker logs mailer-backend 2>&1 | grep -A 5 "admin"
```

### 3. Configure HTTPS (Cloudflare Tunnel)

```bash
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
  -o /usr/bin/cloudflared && chmod +x /usr/bin/cloudflared

cloudflared tunnel login
cloudflared tunnel create mailer
cloudflared tunnel route dns mailer your-domain.com
cloudflared tunnel route dns mailer mailer-api.your-domain.com
```

Create `/root/.cloudflared/config.yml`:

```yaml
tunnel: <TUNNEL_ID>
credentials-file: /root/.cloudflared/<TUNNEL_ID>.json

ingress:
  - hostname: your-domain.com
    service: http://localhost:8080
  - hostname: mailer-api.your-domain.com
    service: http://localhost:8080
  - service: http_status:404
```

> ⚠️ **Important**: The `mailer-api.` subdomain is **required**. Email Workers use this URL to forward emails to your backend via Tunnel, bypassing Cloudflare WAF/Bot protection that would block Worker-to-origin requests.

```bash
cloudflared service install
cp /root/.cloudflared/config.yml /etc/cloudflared/config.yml
systemctl enable --now cloudflared
```

### 4. Configure Email Receiving / 配置邮件接收

#### Option A: One-Click Auto Setup (Recommended / 推荐)

1. Open `https://your-domain.com`, log in with admin credentials
2. **Domains** → Add your domain
3. Click the **☁️ CF** button → Paste your CF API Token → Click **🚀 Start**
4. All 8 steps execute automatically (verify token → find zone → enable Email Routing → create API Key → deploy Worker → configure Catch-all → create DMARC record)
5. The setup is **fully idempotent** — safe to retry if any step fails

**Required CF API Token Permissions (6 permissions):**

| Scope   | Permission              | Level |
| ------- | ----------------------- | ----- |
| Zone    | Zone                    | Read  |
| Zone    | Zone Settings           | Edit  |
| Zone    | DNS                     | Edit  |
| Zone    | Email Routing Rules     | Edit  |
| Account | Workers Scripts         | Edit  |
| Account | Email Routing Addresses | Edit  |

> 💡 Create token at [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens) → **Create Custom Token**. Set Zone/Account Resources to **All zones / All accounts** (or specific domain).

#### Option B: Manual Setup / 手动配置

Create an **Email Worker** in [Cloudflare Dashboard](https://dash.cloudflare.com/?to=/:account/workers-and-pages):

```javascript
export default {
  async email(message, env) {
    const to = message.to;
    const from = message.from;
    console.log("--- Email received / 收到邮件 ---");
    console.log("From / 发件人: " + from + " → To / 收件人: " + to);
    const raw = new Response(message.raw);
    const body = await raw.arrayBuffer();
    const resp = await fetch("https://mailer-api.your-domain.com/api/receive", {
      method: "POST",
      headers: {
        Authorization: "Bearer sk_your_api_key",
        "Content-Type": "application/octet-stream",
        "X-Envelope-To": to,
      },
      body: body,
    });
    const text = await resp.text();
    if (resp.ok) {
      console.log("✅ Forwarded / 转发成功 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
    } else {
      console.log("⚠ Forward failed / 转发失败 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
    }
  },
};
```

> ⚠️ Use `mailer-api.your-domain.com` (Tunnel URL), **NOT** `your-domain.com`, to avoid WAF blocking.

Then set **Email Routing → Routes** Catch-all to "Send to Worker".

### 5. DNS Records / DNS 记录

After enabling Email Routing (via one-click or manually), Cloudflare automatically creates:

| Record | Name                    | Value                                       | Status              |
| ------ | ----------------------- | ------------------------------------------- | ------------------- |
| MX     | your-domain.com         | route{1,2,3}.mx.cloudflare.net.             | Auto ✅             |
| TXT    | your-domain.com         | v=spf1 include:\_spf.mx.cloudflare.net ~all | Auto ✅             |
| TXT    | \_dmarc.your-domain.com | v=DMARC1; p=none;                           | Auto (one-click) ✅ |

> The one-click setup also creates the DMARC record. If using manual setup, add it yourself via CF Dashboard → DNS.

## 📡 API Reference / API 参考

All API endpoints require `Authorization: Bearer <api_key>` header.

### Get Latest Email (Verification Code)

```bash
curl -H "Authorization: Bearer sk_xxx" \
  "https://your-domain.com/api/emails/latest?to=user@your-domain.com"
```

```json
{
  "id": 1,
  "recipient": "user@your-domain.com",
  "sender": "noreply@example.com",
  "subject": "Your verification code",
  "code": "483921",
  "received_at": "2026-01-01T00:00:00Z"
}
```

### List Emails / 邮件列表

```bash
curl -H "Authorization: Bearer sk_xxx" \
  "https://your-domain.com/api/emails?to=user@domain.com&page=1&size=10"
```

Filter params: `to`, `from`, `subject`, `since`, `until`

### Email Detail / Delete

```bash
# Get detail
curl -H "Authorization: Bearer sk_xxx" "https://your-domain.com/api/emails/123"

# Delete
curl -X DELETE -H "Authorization: Bearer sk_xxx" "https://your-domain.com/api/emails/123"
```

### Domains

```bash
# List domains
curl -H "Authorization: Bearer sk_xxx" "https://your-domain.com/api/domains"

# Domain stats
curl -H "Authorization: Bearer sk_xxx" "https://your-domain.com/api/domains/1/stats"
```

### Create Temporary Mailbox (via API)

```bash
curl -X POST -H "Authorization: Bearer sk_xxx" \
  "https://your-domain.com/api/mailboxes?domain=your-domain.com"
```

```json
{
  "email": "sarah.chen29@your-domain.com",
  "password": "e5f6a7b8c9d0",
  "domain": "your-domain.com",
  "expires_at": "2026-06-14"
}
```

> Generates realistic human-like email addresses (170+ first names × 160+ last names × 16 patterns)

## 📬 Mailbox System / 邮箱用户系统

### Temporary Mailbox (Public Registration)

```bash
curl -X POST "https://your-domain.com/mailbox/register"
# Returns: { "email": "emma.silva.dev@domain.com", "password": "xxx", "expires_at": "2026-06-01" }
```

### Mailbox Login

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"email":"user@domain.com","password":"xxx"}' \
  "https://your-domain.com/mailbox/login"
```

### Admin Create Mailbox

```bash
curl -X POST -H "Authorization: Bearer <admin_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@domain.com","password":"123456"}' \
  "https://your-domain.com/admin/mailboxes"
```

## ⚙️ Environment Variables / 环境变量

| Variable              | Default           | Description                                            |
| --------------------- | ----------------- | ------------------------------------------------------ |
| `JWT_SECRET`          | —                 | JWT signing key (**required**)                         |
| `DB_PATH`             | `/data/mailer.db` | SQLite database path                                   |
| `LISTEN_ADDR`         | `:8080`           | Listen address                                         |
| `MAIL_RETENTION_DAYS` | `7`               | Email retention days (0 = no cleanup, starred exempt)  |
| `DEFAULT_ADMIN_USER`  | `admin`           | Initial super admin username                           |
| `DEFAULT_ADMIN_PASS`  | —                 | Initial super admin password (auto-generated if empty) |
| `CORS_ORIGINS`        | `*`               | Allowed CORS origins (comma-separated)                 |

## 📁 Project Structure / 项目结构

```
mailer/
├── backend/
│   ├── config/          # Environment config
│   ├── database/        # SQLite init + auto migration
│   ├── handler/         # Route handlers (admin/api/mailbox/email)
│   ├── middleware/       # JWT auth / API Key / Rate limiting
│   ├── service/         # Mail parsing / Code extraction / Cleanup
│   ├── pipe/            # Postfix pipe receiver (optional)
│   └── main.go          # Entry (embedded frontend + routing)
├── frontend/
│   └── src/
│       ├── views/       # All pages (14 Vue components)
│       ├── i18n/        # Bilingual language packs (zh/en)
│       ├── api/         # Axios API layer
│       ├── stores/      # Pinia auth store
│       └── router/      # Frontend routing
├── docker-compose.yml
├── Dockerfile           # Multi-stage build (frontend + backend → single binary)
├── .env.example
└── README.md
```

## 🔐 Security / 安全说明

- API Keys stored as **bcrypt hashes**
- Admin passwords bcrypt hashed (cost=10)
- Auto-generated 16-char strong passwords on first run
- JWT authentication with configurable secret
- Per-API-Key IP whitelist + rate limiting
- Domain-level permission isolation (API Keys & admins can be bound to specific domains)
- System API Keys (auto-created by CF setup) protected with 🔒 badge and delete warnings
- **No telemetry, no analytics, no third-party dependencies** — your data never leaves your server

## ☕ Support / 支持

If you find this tool useful, consider buying me a coffee! 
如果觉得本项目好用，请我喝杯咖啡吧！让创造力持续燃烧！🔥

<p align="center">
  <img src="./BMC.png" width="260" alt="Buy Me A Coffee QR Code" style="margin-right: 20px" />
  <img src="./wechat.png" width="260" alt="WeChat Pay QR Code" />
</p>

---

## License

MIT

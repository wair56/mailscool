<template>
  <div class="page-container">
    <n-grid :cols="1" :y-gap="20">
      <n-gi>
        <div class="config-card">
          <n-alert type="success" :bordered="false" style="margin-bottom: 16px">
            <template v-if="locale === 'zh'">
              <strong>Catch-all 模式：</strong>无需提前创建邮箱账号，域名下任意地址（如 <code>anything@yourdomain.com</code>）均可直接收件。发信方直接发送，然后通过 API 查询即可获取邮件和验证码。
            </template>
            <template v-else>
              <strong>Catch-all Mode:</strong> No need to create mailbox accounts in advance. Any address under your domain (e.g. <code>anything@yourdomain.com</code>) can receive mail. Just send to it, then query via API to get emails and verification codes.
            </template>
          </n-alert>

          <div class="auth-hint">
            <span class="auth-label">🔑 {{ locale === 'zh' ? '鉴权方式' : 'Auth' }}</span>
            <span class="auth-value">{{ locale === 'zh' ? '所有接口均需在 Header 中携带' : 'All endpoints require header' }} <code>Authorization: Bearer sk_your_api_key</code></span>
          </div>

          <n-tabs type="segment" animated>
            <!-- 获取最新验证码 -->
            <n-tab-pane name="latest" :tab="locale === 'zh' ? '获取最新验证码' : 'Get Latest Code'">
              <div class="api-method"><span class="method-tag get">GET</span> <code>/api/emails/latest</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '返回指定收件人的最新一封邮件，包含自动提取的验证码。最常用的接口。' : 'Returns the latest email for a recipient with auto-extracted verification code. Most commonly used endpoint.' }}</div>

              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? 'Query 参数' : 'Query Params' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '参数' : 'Param' }}</th><th>{{ locale === 'zh' ? '必填' : 'Required' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>to</code></td><td class="required">✅</td><td>string</td><td>{{ locale === 'zh' ? '收件人地址，如' : 'Recipient, e.g.' }} <code>user@yourdomain.com</code></td></tr>
                    <tr><td><code>since</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>{{ locale === 'zh' ? '只查该时间之后的邮件，ISO 8601 格式，如' : 'Only emails after this time, ISO 8601, e.g.' }} <code>2026-03-13T00:00:00Z</code></td></tr>
                  </tbody>
                </table>
              </div>

              <pre class="code-block">curl -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/emails/latest?to=user@yourdomain.com"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应字段' : 'Response' }} <span class="status-tag s200">200</span></div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '字段' : 'Field' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>id</code></td><td>number</td><td>{{ locale === 'zh' ? '邮件 ID' : 'Email ID' }}</td></tr>
                    <tr><td><code>recipient</code></td><td>string</td><td>{{ locale === 'zh' ? '收件人地址' : 'Recipient address' }}</td></tr>
                    <tr><td><code>sender</code></td><td>string</td><td>{{ locale === 'zh' ? '发件人地址' : 'Sender address' }}</td></tr>
                    <tr><td><code>subject</code></td><td>string</td><td>{{ locale === 'zh' ? '邮件主题' : 'Email subject' }}</td></tr>
                    <tr class="highlight-row"><td><code>code</code></td><td>string</td><td>⭐ {{ locale === 'zh' ? '自动提取的验证码（如有）' : 'Auto-extracted verification code (if any)' }}</td></tr>
                    <tr><td><code>body_text</code></td><td>string</td><td>{{ locale === 'zh' ? '纯文本正文' : 'Plain text body' }}</td></tr>
                    <tr><td><code>body_html</code></td><td>string</td><td>{{ locale === 'zh' ? 'HTML 正文' : 'HTML body' }}</td></tr>
                    <tr><td><code>links</code></td><td>string</td><td>{{ locale === 'zh' ? '提取的链接（JSON 数组字符串）' : 'Extracted links (JSON array string)' }}</td></tr>
                    <tr><td><code>has_attachments</code></td><td>boolean</td><td>{{ locale === 'zh' ? '是否有附件' : 'Has attachments' }}</td></tr>
                    <tr><td><code>raw_size</code></td><td>number</td><td>{{ locale === 'zh' ? '原始邮件大小（字节）' : 'Raw email size (bytes)' }}</td></tr>
                    <tr><td><code>is_read</code></td><td>boolean</td><td>{{ locale === 'zh' ? '是否已读' : 'Read status' }}</td></tr>
                    <tr><td><code>is_starred</code></td><td>boolean</td><td>{{ locale === 'zh' ? '是否已标星' : 'Starred status' }}</td></tr>
                    <tr><td><code>received_at</code></td><td>string</td><td>{{ locale === 'zh' ? '接收时间（ISO 8601）' : 'Received time (ISO 8601)' }}</td></tr>
                  </tbody>
                </table>
              </div>
              <div class="param-section">
                <div class="param-title">📋 {{ locale === 'zh' ? '响应示例' : 'Response Example' }}</div>
                <pre class="code-block json">{
  "id": 42,
  "domain_id": 1,
  "recipient": "user@yourdomain.com",
  "sender": "noreply@example.com",
  "subject": "Your verification code",
  "code": "583921",
  "body_text": "Your code is 583921",
  "body_html": "&lt;p&gt;Your code is &lt;b&gt;583921&lt;/b&gt;&lt;/p&gt;",
  "links": "[\"https://example.com/verify\"]",
  "has_attachments": false,
  "raw_size": 2048,
  "is_read": false,
  "is_starred": false,
  "received_at": "2026-03-13T08:30:00Z"
}</pre>
              </div>
              <div class="error-codes">
                <span class="status-tag s400">400</span> {{ locale === 'zh' ? '缺少' : 'Missing' }} <code>to</code> &nbsp;
                <span class="status-tag s404">404</span> {{ locale === 'zh' ? '未找到邮件' : 'Email not found' }} &nbsp;
                <span class="status-tag s401">401</span> {{ locale === 'zh' ? 'API Key 无效' : 'Invalid API Key' }}
              </div>
            </n-tab-pane>

            <!-- 查询邮件列表 -->
            <n-tab-pane name="list" :tab="locale === 'zh' ? '查询邮件列表' : 'List Emails'">
              <div class="api-method"><span class="method-tag get">GET</span> <code>/api/emails</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '分页查询邮件列表，支持多条件筛选。' : 'Paginated email list with multi-condition filtering.' }}</div>

              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? 'Query 参数' : 'Query Params' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '参数' : 'Param' }}</th><th>{{ locale === 'zh' ? '必填' : 'Req' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '默认值' : 'Default' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>to</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>—</td><td>{{ locale === 'zh' ? '精确匹配收件人地址' : 'Exact match recipient' }}</td></tr>
                    <tr><td><code>from</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>—</td><td>{{ locale === 'zh' ? '模糊匹配发件人（包含即可）' : 'Fuzzy match sender (contains)' }}</td></tr>
                    <tr><td><code>subject</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>—</td><td>{{ locale === 'zh' ? '模糊匹配邮件主题' : 'Fuzzy match subject' }}</td></tr>
                    <tr><td><code>since</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>—</td><td>{{ locale === 'zh' ? '起始时间（ISO 8601）' : 'Start time (ISO 8601)' }}</td></tr>
                    <tr><td><code>until</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>—</td><td>{{ locale === 'zh' ? '截止时间（ISO 8601）' : 'End time (ISO 8601)' }}</td></tr>
                    <tr><td><code>page</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>number</td><td>1</td><td>{{ locale === 'zh' ? '页码，从 1 开始' : 'Page number, starts at 1' }}</td></tr>
                    <tr><td><code>size</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>number</td><td>20</td><td>{{ locale === 'zh' ? '每页条数，最大 100' : 'Items per page, max 100' }}</td></tr>
                  </tbody>
                </table>
              </div>

              <pre class="code-block">curl -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/emails?to=user@yourdomain.com&amp;page=1&amp;size=10"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应字段' : 'Response' }} <span class="status-tag s200">200</span></div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '字段' : 'Field' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>total</code></td><td>number</td><td>{{ locale === 'zh' ? '符合条件的总数' : 'Total matching count' }}</td></tr>
                    <tr><td><code>page</code></td><td>number</td><td>{{ locale === 'zh' ? '当前页码' : 'Current page' }}</td></tr>
                    <tr><td><code>size</code></td><td>number</td><td>{{ locale === 'zh' ? '每页条数' : 'Page size' }}</td></tr>
                    <tr><td><code>data</code></td><td>array</td><td>{{ locale === 'zh' ? '邮件列表（简要信息，不含正文）' : 'Email list (summary, no body)' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].id</code></td><td>number</td><td>{{ locale === 'zh' ? '邮件 ID' : 'Email ID' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].recipient</code></td><td>string</td><td>{{ locale === 'zh' ? '收件人' : 'Recipient' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].sender</code></td><td>string</td><td>{{ locale === 'zh' ? '发件人' : 'Sender' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].subject</code></td><td>string</td><td>{{ locale === 'zh' ? '主题' : 'Subject' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].code</code></td><td>string</td><td>{{ locale === 'zh' ? '验证码（如有）' : 'Code (if any)' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].has_attachments</code></td><td>boolean</td><td>{{ locale === 'zh' ? '是否有附件' : 'Has attachments' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].is_read</code></td><td>boolean</td><td>{{ locale === 'zh' ? '是否已读' : 'Read status' }}</td></tr>
                    <tr><td>&nbsp;&nbsp;<code>data[].received_at</code></td><td>string</td><td>{{ locale === 'zh' ? '接收时间' : 'Received at' }}</td></tr>
                  </tbody>
                </table>
              </div>
              <div class="param-section">
                <div class="param-title">📋 {{ locale === 'zh' ? '响应示例' : 'Response Example' }}</div>
                <pre class="code-block json">{
  "total": 56,
  "page": 1,
  "size": 10,
  "data": [
    {
      "id": 42,
      "domain_id": 1,
      "recipient": "user@yourdomain.com",
      "sender": "noreply@example.com",
      "subject": "Your verification code",
      "code": "583921",
      "has_attachments": false,
      "is_read": false,
      "is_starred": false,
      "received_at": "2026-03-13T08:30:00Z"
    }
  ]
}</pre>
              </div>
            </n-tab-pane>

            <!-- 获取邮件详情 -->
            <n-tab-pane name="detail" :tab="locale === 'zh' ? '获取邮件详情' : 'Email Detail'">
              <div class="api-method"><span class="method-tag get">GET</span> <code>/api/emails/:id</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '获取单封邮件的完整内容，包含 HTML/纯文本正文、提取的链接和验证码。调用后自动标记为已读。' : 'Get full email content including HTML/text body, extracted links and code. Auto-marks as read.' }}</div>

              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? '路径参数' : 'Path Params' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '参数' : 'Param' }}</th><th>{{ locale === 'zh' ? '必填' : 'Req' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>id</code></td><td class="required">✅</td><td>number</td><td>{{ locale === 'zh' ? '邮件 ID' : 'Email ID' }}</td></tr>
                  </tbody>
                </table>
              </div>

              <pre class="code-block">curl -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/emails/42"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应字段' : 'Response' }} <span class="status-tag s200">200</span></div>
                <div class="api-desc">{{ locale === 'zh' ? '响应结构同「获取最新验证码」，包含完整的' : 'Same structure as Get Latest Code, includes full' }} <code>body_text</code>{{ locale === 'zh' ? '、' : ', ' }}<code>body_html</code>{{ locale === 'zh' ? '、' : ', ' }}<code>links</code></div>
              </div>
              <div class="error-codes">
                <span class="status-tag s404">404</span> {{ locale === 'zh' ? '邮件不存在' : 'Not found' }} &nbsp;
                <span class="status-tag s403">403</span> {{ locale === 'zh' ? '无权访问（API Key 未绑定该域名）' : 'No access (API Key not bound to domain)' }}
              </div>
            </n-tab-pane>

            <!-- 删除邮件 -->
            <n-tab-pane name="delete" :tab="locale === 'zh' ? '删除邮件' : 'Delete Email'">
              <div class="api-method"><span class="method-tag delete">DELETE</span> <code>/api/emails/:id</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '删除指定 ID 的邮件。仅能删除 API Key 所绑定域名下的邮件。' : 'Delete email by ID. Only emails under API Key bound domains.' }}</div>

              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? '路径参数' : 'Path Params' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '参数' : 'Param' }}</th><th>{{ locale === 'zh' ? '必填' : 'Req' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>id</code></td><td class="required">✅</td><td>number</td><td>{{ locale === 'zh' ? '邮件 ID' : 'Email ID' }}</td></tr>
                  </tbody>
                </table>
              </div>

              <pre class="code-block">curl -X DELETE -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/emails/42"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应' : 'Response' }} <span class="status-tag s200">200</span></div>
                <pre class="code-block json">{ "message": "deleted" }</pre>
              </div>
              <div class="error-codes">
                <span class="status-tag s404">404</span> {{ locale === 'zh' ? '邮件不存在' : 'Not found' }} &nbsp;
                <span class="status-tag s403">403</span> {{ locale === 'zh' ? '无权删除' : 'No permission' }}
              </div>
            </n-tab-pane>

            <n-tab-pane name="domains" :tab="locale === 'zh' ? '查询域名' : 'List Domains'">
              <div class="api-method"><span class="method-tag get">GET</span> <code>/api/domains</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '列出当前 API Key 所绑定的域名列表。' : 'List domains bound to current API Key.' }}</div>

              <pre class="code-block">curl -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/domains"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应示例' : 'Response Example' }} <span class="status-tag s200">200</span></div>
                <pre class="code-block json">{
  "data": [
    { "id": 1, "name": "yourdomain.com", "is_active": true }
  ]
}</pre>
              </div>
            </n-tab-pane>

            <n-tab-pane name="mailbox" :tab="locale === 'zh' ? '创建邮箱' : 'Create Mailbox'">
              <div class="api-method"><span class="method-tag post">POST</span> <code>/api/mailboxes</code></div>
              <div class="api-desc">{{ locale === 'zh' ? '创建一个临时邮箱，返回邮箱地址和密码。用户可用该凭据登录查看收件箱。' : 'Create a temporary mailbox. Returns email address and password. User can login to view inbox.' }}</div>
              <n-alert type="info" :bordered="false" style="margin: 8px 0">
                {{ locale === 'zh'
                  ? '💡 仅当需要用户登录收件箱查看邮件时才需要创建邮箱。如果只是通过 API 收取邮件，无需创建邮箱 —— 直接往域名下任意地址发信，然后用 API 查询即可。'
                  : '💡 Only create a mailbox if users need to login and view their inbox. For API-only workflows, no mailbox is needed — just send to any address under your domain and query via API.' }}
              </n-alert>
              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? 'Query 参数' : 'Query Params' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '参数' : 'Param' }}</th><th>{{ locale === 'zh' ? '必填' : 'Req' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>domain</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>{{ locale === 'zh' ? '指定域名，如' : 'Target domain, e.g.' }} <code>yourdomain.com</code>{{ locale === 'zh' ? '。默认随机' : '. Default random' }}</td></tr>
                    <tr><td><code>username</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>{{ locale === 'zh' ? '自定义用户名（@前的部分）。不填则随机生成' : 'Custom username (before @). Random if empty' }}</td></tr>
                    <tr><td><code>password</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>{{ locale === 'zh' ? '自定义密码。不填则随机生成' : 'Custom password. Random if empty' }}</td></tr>
                    <tr><td><code>webhook_url</code></td><td>{{ locale === 'zh' ? '否' : 'No' }}</td><td>string</td><td>{{ locale === 'zh' ? 'Webhook 推送地址。配置后收到新邮件时会自动 POST JSON 到该 URL' : 'Webhook URL. New emails will be POSTed as JSON to this URL' }}</td></tr>
                  </tbody>
                </table>
              </div>

              <pre class="code-block">curl -X POST -H "Authorization: Bearer sk_your_api_key" \
  "{{ apiBase }}/api/mailboxes?username=myuser&amp;domain=yourdomain.com"</pre>

              <div class="param-section">
                <div class="param-title">📤 {{ locale === 'zh' ? '响应' : 'Response' }} <span class="status-tag s200">200</span></div>
                <pre class="code-block json">{
  "email": "tmp_a1b2c3d4@yourdomain.com",
  "password": "e5f6a7b8c9d0",
  "domain": "yourdomain.com",
  "expires_at": "2026-06-14"
}</pre>
              </div>

              <div class="param-section">
                <div class="param-title">📥 {{ locale === 'zh' ? '响应字段' : 'Response Fields' }}</div>
                <table class="param-table">
                  <thead><tr><th>{{ locale === 'zh' ? '字段' : 'Field' }}</th><th>{{ locale === 'zh' ? '类型' : 'Type' }}</th><th>{{ locale === 'zh' ? '说明' : 'Description' }}</th></tr></thead>
                  <tbody>
                    <tr><td><code>email</code></td><td>string</td><td>{{ locale === 'zh' ? '生成的邮箱地址' : 'Generated email address' }}</td></tr>
                    <tr><td><code>password</code></td><td>string</td><td>{{ locale === 'zh' ? '登录密码（仅此次返回）' : 'Login password (returned only once)' }}</td></tr>
                    <tr><td><code>domain</code></td><td>string</td><td>{{ locale === 'zh' ? '所属域名' : 'Domain' }}</td></tr>
                    <tr><td><code>expires_at</code></td><td>string</td><td>{{ locale === 'zh' ? '过期日期' : 'Expiry date' }}</td></tr>
                  </tbody>
                </table>
              </div>
              <div class="error-codes">
                <span class="status-tag s400">400</span> {{ locale === 'zh' ? '域名不存在或未激活' : 'Domain not found or inactive' }} &nbsp;
                <span class="status-tag s403">403</span> {{ locale === 'zh' ? 'API Key 无此域名权限' : 'API Key not authorized for domain' }}
              </div>
            </n-tab-pane>
          </n-tabs>
        </div>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup>
import { useI18n } from '../i18n'
const { locale } = useI18n()
const apiBase = window.location.origin
</script>

<style scoped>
.config-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px 24px;
}

.code-block {
  background: #080c14;
  color: #00f0ff;
  padding: 16px;
  border-radius: 10px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 8px 0;
  border: 1px solid var(--border-color);
}

.api-desc {
  color: var(--text-secondary);
  font-size: 13px;
  margin-top: 8px;
}
.api-desc code {
  color: #0aff9d;
  background: rgba(10, 255, 157, 0.08);
  padding: 1px 6px;
  border-radius: 4px;
}

.auth-hint {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: rgba(0, 240, 255, 0.04);
  border: 1px solid rgba(0, 240, 255, 0.1);
  border-radius: 8px;
  margin-bottom: 16px;
  font-size: 13px;
}
.auth-label {
  font-weight: 600;
  white-space: nowrap;
  font-family: 'JetBrains Mono', monospace;
}
.auth-value {
  color: var(--text-secondary);
}
.auth-value code {
  color: #00f0ff;
  background: rgba(0, 240, 255, 0.08);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.api-method {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}
.api-method code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  font-weight: 600;
  color: #e0e0e0;
}

.method-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 700;
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.5px;
}
.method-tag.get { background: rgba(10, 255, 157, 0.15); color: #0aff9d; border: 1px solid rgba(10, 255, 157, 0.3); }
.method-tag.post { background: rgba(0, 150, 255, 0.15); color: #59b3ff; border: 1px solid rgba(0, 150, 255, 0.3); }
.method-tag.delete { background: rgba(255, 80, 80, 0.15); color: #ff6b6b; border: 1px solid rgba(255, 80, 80, 0.3); }

.param-section { margin: 12px 0; }
.param-title {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.param-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  margin-bottom: 4px;
}
.param-table th {
  text-align: left;
  padding: 8px 10px;
  background: rgba(0, 240, 255, 0.05);
  border-bottom: 1px solid var(--border-color);
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
}
.param-table td {
  padding: 7px 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
}
.param-table td code {
  color: #00f0ff;
  background: rgba(0, 240, 255, 0.06);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
}
.param-table td.required { color: #0aff9d; font-weight: 600; }
.param-table tr.highlight-row td { background: rgba(255, 215, 0, 0.04); }
.param-table tr.highlight-row td code { color: #ffd700; background: rgba(255, 215, 0, 0.1); }

.status-tag {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 700;
  font-family: 'JetBrains Mono', monospace;
}
.status-tag.s200 { background: rgba(10, 255, 157, 0.12); color: #0aff9d; }
.status-tag.s400 { background: rgba(255, 170, 0, 0.12); color: #ffaa00; }
.status-tag.s401 { background: rgba(255, 100, 100, 0.12); color: #ff6464; }
.status-tag.s403 { background: rgba(255, 80, 80, 0.12); color: #ff5050; }
.status-tag.s404 { background: rgba(150, 150, 150, 0.12); color: #999; }

.error-codes {
  margin-top: 10px;
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.error-codes code {
  color: #00f0ff;
  background: rgba(0, 240, 255, 0.06);
  padding: 1px 4px;
  border-radius: 3px;
}

.code-block.json { color: #b5e853; }
</style>

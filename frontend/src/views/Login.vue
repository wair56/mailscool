<template>
  <div class="login-page">
    <!-- Language switcher -->
    <select class="lang-switcher" :value="locale" @change="setLocale($event.target.value)">
      <option v-for="l in availableLocales" :key="l.code" :value="l.code">{{ l.label }}</option>
    </select>
    <!-- Animated background grid -->
    <div class="bg-grid"></div>
    <div class="bg-glow"></div>

    <div class="login-container">
      <!-- Left: Project intro + API docs -->
      <div class="intro-panel">
        <div class="intro-header">
          <div class="logo-mark">
            <span class="logo-bracket">[</span>
            <span class="logo-letter">M</span>
            <span class="logo-bracket">]</span>
          </div>
          <h1 class="brand-name">MAILS<span class="brand-accent">COOL</span></h1>
          <p class="intro-tagline">{{ t('login_brand_tagline') }}</p>
        </div>

        <div class="intro-features">
          <div class="feature-item stagger-1">
            <span class="feature-dot"></span>
            <div>
              <strong>{{ t('login_feature1_title') }}</strong>
              <span>{{ t('login_feature1_desc') }}</span>
            </div>
          </div>
          <div class="feature-item stagger-2">
            <span class="feature-dot"></span>
            <div>
              <strong>{{ t('login_feature2_title') }}</strong>
              <span>{{ t('login_feature2_desc') }}</span>
            </div>
          </div>
          <div class="feature-item stagger-3">
            <span class="feature-dot"></span>
            <div>
              <strong>{{ t('login_feature3_title') }}</strong>
              <span>{{ t('login_feature3_desc') }}</span>
            </div>
          </div>
        </div>

        <!-- API Reference - Terminal Style -->
        <div class="api-section stagger-4">
          <div class="api-header">
            <span class="terminal-dot red"></span>
            <span class="terminal-dot yellow"></span>
            <span class="terminal-dot green"></span>
            <span class="terminal-title">api_reference.sh</span>
            <a href="/api-docs" target="_blank" class="docs-link">{{ t('login_view_docs') }}</a>
          </div>
          <div class="api-body">
            <div class="api-line"><span class="prompt">$</span> <span class="cmd">GET</span> <code>/api/emails/latest?to=user@example.com</code></div>
            <div class="api-line"><span class="prompt">$</span> <span class="cmd">GET</span> <code>/api/emails?to=user@example.com&amp;page=1</code></div>
            <div class="api-line"><span class="prompt">$</span> <span class="cmd">GET</span> <code>/api/emails/:id</code></div>
            <div class="api-line"><span class="prompt">$</span> <span class="cmd">GET</span> <code>/api/domains</code></div>
            <div class="api-line comment"># Authorization: Bearer sk_your_api_key</div>
          </div>
        </div>
      </div>

      <!-- Right: Main content area -->
      <div class="login-card stagger-5">
        <!-- Mode Toggle -->
        <div class="mode-toggle">
          <button :class="{ active: mode === 'temp' }" @click="mode = 'temp'">{{ t('login_mode_temp') }}</button>
          <button :class="{ active: mode === 'mailbox' }" @click="mode = 'mailbox'">{{ t('login_mode_mailbox') }}</button>
          <button :class="{ active: mode === 'admin' }" @click="mode = 'admin'">{{ t('login_mode_admin') }}</button>
        </div>

        <!-- Mode 1: Quick Temp Email + Code Polling (主入口) -->
        <div v-if="mode === 'temp'" class="mode-content">
          <div class="mode-desc">{{ t('login_temp_desc') }}</div>
          <div v-if="turnstileSiteKey" id="turnstile-widget" style="margin-bottom: 12px; display: flex; justify-content: center;"></div>
          <button class="login-btn pulse" :disabled="tempLoading" @click="handleCreateTemp">
            <span v-if="!tempLoading">⚡ {{ t('login_temp_btn') }}</span>
            <span v-else>{{ t('login_temp_loading') }}</span>
          </button>

          <!-- Temp Result + Code Polling Panel -->
          <div v-if="tempResult" class="temp-result-panel">
            <div class="temp-info">
              <div class="temp-line">📧 <span class="mono cyan">{{ tempResult.email }}</span>
                <button class="copy-mini" @click="copyText(tempResult.email)">{{ t('login_copy') }}</button>
              </div>
              <div class="temp-line">🔑 <span class="mono green">{{ tempResult.password }}</span>
                <button class="copy-mini" @click="copyText(tempResult.password)">{{ t('login_copy') }}</button>
              </div>
              <div class="temp-line dim">{{ t('login_temp_expires') }} {{ tempResult.expires_at }}</div>
            </div>

            <!-- Code Polling -->
            <div class="code-poll-area">
              <div class="code-poll-header">
                <span class="poll-dot" :class="{ active: polling }"></span>
                <template v-if="polledCode && !polling">✅ {{ t('login_code_received') }}</template>
                <template v-else-if="polledEmailCount > 0 && !polledCode">📨 {{ t('login_email_arrived', { count: polledEmailCount }) }}</template>
                <template v-else>{{ t('login_waiting_code') }}</template>
              </div>
              <div v-if="polledCode" class="polled-code">
                <div class="code-value">{{ polledCode }}</div>
                <div class="code-subject">{{ polledSubject }}</div>
                <div v-if="polledSender" class="code-sender">✉️ {{ polledSender }}</div>
              </div>
              <div v-else-if="polledEmailCount > 0" class="polled-code">
                <div class="code-subject">{{ polledSubject }}</div>
                <div v-if="polledSender" class="code-sender">✉️ {{ polledSender }}</div>
                <div style="font-size: 11px; color: #5e7290; margin-top: 6px">{{ t('login_no_code_hint') }}</div>
              </div>
              <div v-else class="poll-spinner">
                {{ t('login_polling') }}
              </div>
            </div>

            <!-- Action Buttons - always visible -->
            <div class="temp-actions">
              <button class="action-btn inbox-btn" @click="enterInbox">
                📬 {{ t('login_enter_inbox') }}
              </button>
              <button v-if="polledCode" class="action-btn copy-btn" @click="copyText(polledCode)">
                📋 {{ t('login_copy_code') }}
              </button>
              <button v-if="!polling" class="action-btn wait-btn" @click="resumePolling">
                🔄 {{ t('login_continue_wait') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="mode === 'mailbox'" class="mode-content mode-center">
          <n-form ref="mailboxFormRef" :model="mailboxForm" @submit.prevent="handleMailboxLogin">
            <n-form-item :label="t('login_mailbox_email')">
              <n-input v-model:value="mailboxForm.email" :placeholder="t('login_mailbox_email_ph')" size="large" />
            </n-form-item>
            <n-form-item :label="t('login_mailbox_password')">
              <n-input v-model:value="mailboxForm.password" type="password" :placeholder="t('login_mailbox_password_ph')"
                size="large" show-password-on="click" @keyup.enter="handleMailboxLogin" />
            </n-form-item>
            <button type="button" class="login-btn" :disabled="mailboxLoading" @click="handleMailboxLogin">
              <span v-if="!mailboxLoading">{{ t('login_mailbox_btn') }}</span>
              <span v-else>{{ t('login_mailbox_loading') }}</span>
            </button>
          </n-form>
        </div>

        <!-- Mode 3: Admin Login -->
        <div v-if="mode === 'admin'" class="mode-content mode-center">
          <n-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
            <n-form-item path="username" :label="t('login_username')">
              <n-input v-model:value="form.username" :placeholder="t('login_input_username')" size="large" />
            </n-form-item>
            <n-form-item path="password" :label="t('login_password')">
              <n-input v-model:value="form.password" type="password" :placeholder="t('login_input_password')"
                size="large" show-password-on="click" @keyup.enter="handleLogin" />
            </n-form-item>
            <button type="button" class="login-btn" :disabled="loading" @click="handleLogin">
              <span v-if="!loading">{{ t('login_btn') }}</span>
              <span v-else>{{ t('login_btn_loading') }}</span>
            </button>
          </n-form>
        </div>
      </div>
    </div>

    <!-- Footer links -->
    <div class="login-footer">
      <a href="https://github.com/wair56/mailer" target="_blank">GitHub</a>
      <span class="footer-sep">·</span>
      <a href="https://www.buymeacoffee.com/399is" target="_blank">☕ Buy Me a Coffee</a>
      <span class="footer-sep">·</span>
      <a href="/privacy.html" target="_blank">Privacy Policy</a>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { login } from '../api'
import api from '../api'
import { useI18n } from '../i18n'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const auth = useAuthStore()
const loading = ref(false)
const formRef = ref(null)
const { t, locale, setLocale, availableLocales } = useI18n()

const mode = ref('temp') // 'temp' | 'mailbox' | 'admin'

const form = ref({ username: '', password: '' })
const rules = {
  username: { required: true, message: t('login_require_username') },
  password: { required: true, message: t('login_require_password') },
}

async function handleLogin() {
  try { await formRef.value?.validate() } catch { return }
  loading.value = true
  try {
    const { data } = await login(form.value)
    auth.setAuth(data.token, data.admin)
    message.success(t('login_auth_success'))
    router.push('/')
  } catch (e) {
    message.error(e.response?.data?.error || t('login_auth_fail'))
  } finally { loading.value = false }
}

// 邮箱登录
const mailboxForm = ref({ email: '', password: '' })
const mailboxLoading = ref(false)
const mailboxFormRef = ref(null)

async function handleMailboxLogin() {
  if (!mailboxForm.value.email || !mailboxForm.value.password) {
    message.warning(t('login_mailbox_require'))
    return
  }
  mailboxLoading.value = true
  try {
    const { data } = await api.post('/mailbox/login', mailboxForm.value)
    localStorage.setItem('mailbox_token', data.token)
    localStorage.setItem('mailbox_email', data.email)
    message.success(t('login_mailbox_success'))
    router.push('/inbox')
  } catch (e) {
    message.error(e.response?.data?.error || t('login_mailbox_fail'))
  } finally { mailboxLoading.value = false }
}

// Auto-login from URL params (?email=xxx&password=xxx)
// Turnstile support
const turnstileSiteKey = ref('')
const turnstileToken = ref('')

onMounted(async () => {
  const urlEmail = route.query.email
  const urlPass = route.query.password
  if (urlEmail && urlPass) {
    mode.value = 'mailbox'
    mailboxForm.value.email = urlEmail
    mailboxForm.value.password = urlPass
    setTimeout(() => handleMailboxLogin(), 100)
  }

  // Fetch public config for Turnstile
  try {
    const { data } = await api.get('/public/config')
    if (data.turnstile_site_key) {
      turnstileSiteKey.value = data.turnstile_site_key
      // Load Turnstile script
      const s = document.createElement('script')
      s.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onTurnstileLoad'
      s.async = true
      window.onTurnstileLoad = () => {
        if (document.getElementById('turnstile-widget')) {
          window.turnstile.render('#turnstile-widget', {
            sitekey: data.turnstile_site_key,
            callback: (token) => { turnstileToken.value = token },
            theme: 'dark'
          })
        }
      }
      document.head.appendChild(s)
    }
  } catch (e) { /* ignore */ }
})

// 创建临时邮箱 + 验证码轮询
const tempLoading = ref(false)
const tempResult = ref(null)
const polledCode = ref('')
const polledSubject = ref('')
const polledSender = ref('')
const polledEmailCount = ref(0)
const polledLastId = ref(null)
const polling = ref(false)
let pollTimer = null
let pollToken = null

async function handleCreateTemp() {
  // Turnstile check
  if (turnstileSiteKey.value && !turnstileToken.value) {
    message.warning(t('login_turnstile_required'))
    return
  }
  tempLoading.value = true
  tempResult.value = null
  polledCode.value = ''
  polledSubject.value = ''
  try {
    const params = {}
    if (turnstileToken.value) params.turnstile_token = turnstileToken.value
    const { data } = await api.post('/mailbox/register', params)
    tempResult.value = data
    message.success(t('login_temp_success'))
    // Auto-login for polling
    const loginResp = await api.post('/mailbox/login', { email: data.email, password: data.password })
    localStorage.setItem('mailbox_token', loginResp.data.token)
    localStorage.setItem('mailbox_email', data.email)
    startPolling(loginResp.data.token)
    // Reset turnstile for next use
    if (window.turnstile && document.getElementById('turnstile-widget')) {
      window.turnstile.reset('#turnstile-widget')
      turnstileToken.value = ''
    }
  } catch (e) {
    message.error(e.response?.data?.error || t('login_temp_fail'))
    // Reset turnstile on error
    if (window.turnstile && document.getElementById('turnstile-widget')) {
      window.turnstile.reset('#turnstile-widget')
      turnstileToken.value = ''
    }
  } finally { tempLoading.value = false }
}

function startPolling(token) {
  if (pollTimer) clearInterval(pollTimer)
  polling.value = true
  pollToken = token
  pollTimer = setInterval(() => pollForCode(token), 3000)
}

function resumePolling() {
  if (!pollToken) return
  polledCode.value = ''
  polledSubject.value = ''
  polledSender.value = ''
  polling.value = true
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(() => pollForCode(pollToken), 3000)
}

async function pollForCode(token) {
  try {
    const { data } = await api.get('/mailbox/emails', {
      params: { page: 1, size: 5 },
      headers: { Authorization: `Bearer ${token}` }
    })
    const emails = data.data || []
    const latest = emails[0]
    if (!latest) return

    // Check if new email arrived
    if (latest.id !== polledLastId.value) {
      polledLastId.value = latest.id
      polledSubject.value = latest.subject || ''
      polledSender.value = latest.sender || ''
      polledEmailCount.value = emails.length
    }

    // Check for code in any recent email
    for (const email of emails) {
      if (email.code && email.code !== polledCode.value) {
        polledCode.value = email.code
        polledSubject.value = email.subject || ''
        polledSender.value = email.sender || ''
        polling.value = false
        clearInterval(pollTimer)
        // Auto copy
        navigator.clipboard.writeText(email.code).catch(() => {})
        message.success(t('login_code_auto_copied'))
        return
      }
    }
  } catch {}
}

function enterInbox() {
  router.push('/inbox')
}

function copyText(text) {
  navigator.clipboard.writeText(text).catch(() => {})
  message.success(t('login_copied'))
}

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #080c14;
  padding: 24px;
  position: relative;
  overflow: hidden;
}
.bg-grid {
  position: absolute; inset: 0;
  background-image: linear-gradient(rgba(0,240,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(0,240,255,0.03) 1px, transparent 1px);
  background-size: 60px 60px;
  animation: gridScroll 20s linear infinite;
}
@keyframes gridScroll { to { background-position: 60px 60px; } }
.bg-glow {
  position: absolute; width: 600px; height: 600px; border-radius: 50%;
  background: radial-gradient(circle, rgba(0,240,255,0.06) 0%, transparent 60%);
  top: 50%; left: 30%; transform: translate(-50%, -50%); pointer-events: none;
}
.login-container {
  display: flex; max-width: 1000px; width: 100%; border-radius: 12px; overflow: hidden;
  border: 1px solid #1a2940; box-shadow: 0 0 40px rgba(0,240,255,0.05), 0 20px 60px rgba(0,0,0,0.6);
  position: relative; z-index: 1;
}
.intro-panel {
  flex: 1; padding: 40px 32px;
  background: linear-gradient(170deg, #0d1520 0%, #080c14 100%);
  border-right: 1px solid #1a2940;
}
.intro-header { text-align: center; margin-bottom: 28px; }
.logo-mark { font-family: 'JetBrains Mono', monospace; font-size: 36px; font-weight: 700; margin-bottom: 12px; }
.logo-bracket { color: #5e7290; }
.logo-letter { color: #00f0ff; text-shadow: 0 0 12px rgba(0,240,255,0.4); }
.brand-name { font-family: 'JetBrains Mono', monospace; font-size: 20px; font-weight: 700; color: #c8d6e5; letter-spacing: 4px; }
.brand-accent { color: #00f0ff; text-shadow: 0 0 8px rgba(0,240,255,0.3); }
.intro-tagline { color: #5e7290; font-size: 12px; margin-top: 6px; letter-spacing: 1px; }
.intro-features { display: flex; flex-direction: column; gap: 10px; margin-bottom: 24px; }
.feature-item {
  display: flex; gap: 12px; align-items: flex-start; padding: 10px 14px;
  background: rgba(0,240,255,0.02); border-radius: 6px; border: 1px solid rgba(0,240,255,0.06);
  transition: all 0.3s ease;
}
.feature-item:hover { border-color: rgba(0,240,255,0.15); background: rgba(0,240,255,0.04); }
.feature-dot {
  width: 8px; height: 8px; border-radius: 50%; background: #00f0ff;
  box-shadow: 0 0 6px rgba(0,240,255,0.5); flex-shrink: 0; margin-top: 5px;
}
.feature-item strong { display: block; font-family: 'JetBrains Mono', monospace; font-size: 12px; letter-spacing: 0.5px; margin-bottom: 2px; color: #c8d6e5; }
.feature-item span { color: #5e7290; font-size: 12px; line-height: 1.4; }
.api-section { border-radius: 8px; overflow: hidden; border: 1px solid #1a2940; }
.api-header { display: flex; align-items: center; gap: 6px; padding: 8px 12px; background: #0d1520; border-bottom: 1px solid #1a2940; }
.terminal-dot { width: 10px; height: 10px; border-radius: 50%; }
.terminal-dot.red { background: #ff3d71; }
.terminal-dot.yellow { background: #ffaa00; }
.terminal-dot.green { background: #0aff9d; }
.terminal-title { margin-left: 8px; font-family: 'JetBrains Mono', monospace; font-size: 11px; color: #5e7290; }
.api-body { padding: 12px 14px; background: #080c14; }
.api-line { font-family: 'JetBrains Mono', monospace; font-size: 11px; line-height: 2; color: #c8d6e5; }
.api-line .prompt { color: #0aff9d; }
.api-line .cmd { color: #00f0ff; font-weight: 600; }
.api-line code { color: #5e7290; }
.api-line.comment { color: #3a4f6a; font-style: italic; }

/* Right Panel */
.login-card {
  width: 380px; flex-shrink: 0; padding: 28px 28px;
  background: #0d1520; display: flex; flex-direction: column;
}

/* Mode Toggle */
.mode-toggle {
  display: flex; gap: 4px; margin-bottom: 20px;
  background: rgba(0,240,255,0.04); border-radius: 8px; padding: 3px;
  border: 1px solid #1a2940;
}
.mode-toggle button {
  flex: 1; padding: 8px 6px; border: none; background: transparent; color: #5e7290;
  font-family: 'JetBrains Mono', monospace; font-size: 11px; font-weight: 500;
  border-radius: 6px; cursor: pointer; transition: all 0.2s; letter-spacing: 0.5px;
}
.mode-toggle button.active {
  background: rgba(0,240,255,0.12); color: #00f0ff;
  box-shadow: 0 0 10px rgba(0,240,255,0.1);
}
.mode-toggle button:hover:not(.active) { color: #c8d6e5; }
.mode-content { flex: 1; }
.mode-content.mode-center { display: flex; flex-direction: column; justify-content: flex-start; padding-top: 15%; }
.mode-desc { color: #5e7290; font-size: 12px; margin-bottom: 14px; line-height: 1.5; }

/* Login Button */
.login-btn {
  width: 100%; padding: 14px; border: 1px solid #00f0ff; background: transparent;
  color: #00f0ff; font-family: 'JetBrains Mono', monospace; font-size: 13px;
  font-weight: 600; letter-spacing: 2px; border-radius: 6px; cursor: pointer; transition: all 0.3s ease;
}
.login-btn:hover:not(:disabled) { background: rgba(0,240,255,0.08); box-shadow: 0 0 20px rgba(0,240,255,0.15); }
.login-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.login-btn.pulse { animation: btnPulse 2s ease-in-out infinite; }
@keyframes btnPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(0,240,255,0.2); }
  50% { box-shadow: 0 0 20px 4px rgba(0,240,255,0.15); }
}

/* Temp Result Panel */
.temp-result-panel { margin-top: 14px; }
.temp-info {
  padding: 10px 12px; background: rgba(0,240,255,0.04); border: 1px solid rgba(0,240,255,0.12);
  border-radius: 8px; margin-bottom: 10px;
}
.temp-line { font-size: 12px; line-height: 2; display: flex; align-items: center; gap: 6px; }
.temp-line .dim { color: #5e7290; font-size: 10px; }
.mono { font-family: 'JetBrains Mono', monospace; }
.cyan { color: #00f0ff; }
.green { color: #0aff9d; }
.copy-mini {
  padding: 2px 8px; border: 1px solid #1a2940; background: transparent; color: #5e7290;
  font-size: 10px; border-radius: 4px; cursor: pointer; transition: all 0.2s;
  font-family: 'JetBrains Mono', monospace; margin-left: auto;
}
.copy-mini:hover { color: #00f0ff; border-color: #00f0ff; }

/* Code Polling Area */
.code-poll-area {
  padding: 12px; background: rgba(10,255,157,0.04); border: 1px solid rgba(10,255,157,0.12);
  border-radius: 8px; margin-bottom: 10px; min-height: 80px;
}
.code-poll-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 11px; color: #5e7290; font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.5px; margin-bottom: 8px;
}
.poll-dot {
  width: 8px; height: 8px; border-radius: 50%; background: #3a4f6a;
  transition: all 0.3s;
}
.poll-dot.active {
  background: #0aff9d; box-shadow: 0 0 8px rgba(10,255,157,0.5);
  animation: dotPulse 1s ease-in-out infinite;
}
@keyframes dotPulse {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}
.polled-code { text-align: center; }
.code-label { font-size: 10px; color: #5e7290; text-transform: uppercase; letter-spacing: 1px; }
.code-value {
  font-family: 'JetBrains Mono', monospace; font-size: 28px; font-weight: 700;
  color: #0aff9d; text-shadow: 0 0 12px rgba(10,255,157,0.3);
  margin: 4px 0; letter-spacing: 4px;
}
.code-subject { font-size: 11px; color: #5e7290; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.code-sender { font-size: 10px; color: #3a4f6a; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-top: 2px; }
.poll-spinner {
  display: flex; align-items: center; gap: 8px; justify-content: center;
  color: #5e7290; font-size: 12px; padding: 8px 0;
}

/* Mailbox Button */
.mailbox-btn {
  width: 100%; padding: 10px; border: 1px solid rgba(10,255,157,0.3); background: transparent;
  color: #0aff9d; font-family: 'JetBrains Mono', monospace; font-size: 12px;
  font-weight: 600; letter-spacing: 1px; border-radius: 6px; cursor: pointer; transition: all 0.3s ease;
}
.mailbox-btn:hover:not(:disabled) { background: rgba(10,255,157,0.06); box-shadow: 0 0 16px rgba(10,255,157,0.1); }
.mailbox-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* Temp Action Buttons */
.temp-actions {
  display: flex; flex-direction: column; gap: 8px; margin-top: 12px;
}
.action-btn {
  width: 100%; padding: 12px 16px; border-radius: 6px; cursor: pointer;
  font-family: 'JetBrains Mono', monospace; font-size: 13px; font-weight: 600;
  letter-spacing: 0.5px; transition: all 0.3s ease; border: 1px solid;
  text-align: center; white-space: nowrap;
}
.action-btn.inbox-btn {
  background: rgba(0,240,255,0.08); border-color: rgba(0,240,255,0.3); color: #00f0ff;
}
.action-btn.inbox-btn:hover {
  background: rgba(0,240,255,0.15); box-shadow: 0 0 16px rgba(0,240,255,0.15);
}
.action-btn.copy-btn {
  background: rgba(10,255,157,0.08); border-color: rgba(10,255,157,0.3); color: #0aff9d;
}
.action-btn.copy-btn:hover {
  background: rgba(10,255,157,0.15); box-shadow: 0 0 16px rgba(10,255,157,0.15);
}
.action-btn.wait-btn {
  background: rgba(255,170,0,0.08); border-color: rgba(255,170,0,0.3); color: #ffaa00;
}
.action-btn.wait-btn:hover {
  background: rgba(255,170,0,0.15); box-shadow: 0 0 16px rgba(255,170,0,0.15);
}

/* Responsive */
@media (max-width: 768px) {
  .login-container { flex-direction: column; }
  .intro-panel { border-right: none; border-bottom: 1px solid #1a2940; }
  .login-card { width: 100%; }
}

/* Lang Switcher */
.lang-switcher {
  position: absolute; top: 16px; right: 20px; z-index: 10; cursor: pointer;
  font-family: 'JetBrains Mono', monospace; font-size: 13px; font-weight: 600;
  color: #5e7290; padding: 4px 10px; border: 1px solid rgba(94,114,144,0.3);
  border-radius: 4px; transition: all 0.3s ease; background: transparent;
  -webkit-appearance: none; appearance: none;
}
.lang-switcher option { background: #0d1520; color: #c8d6e5; }
.lang-switcher:hover { color: #00f0ff; border-color: rgba(0,240,255,0.4); }

.docs-link {
  margin-left: auto; color: #5e7290; text-decoration: none;
  font-size: 11px; font-family: 'JetBrains Mono', monospace; transition: color 0.3s;
}
.docs-link:hover { color: #00f0ff; }

.login-footer {
  position: absolute; bottom: 16px; left: 0; right: 0;
  text-align: center; padding: 10px 0;
  font-family: 'JetBrains Mono', monospace; font-size: 11px;
  z-index: 10;
}
.login-footer a {
  color: #5e7290; text-decoration: none; transition: color 0.2s;
}
.login-footer a:hover { color: #00f0ff; }
.footer-sep { color: #3a4f6a; margin: 0 8px; }
</style>

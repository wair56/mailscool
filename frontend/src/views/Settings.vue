<template>
  <div class="page-container">
    <n-grid :cols="1" :y-gap="20">
      <!-- 服务状态 -->
      <n-gi>
        <div class="config-card">
          <div class="card-header">
            <span class="card-title">{{ t('settings_service_status') }}</span>
          </div>
          <n-descriptions bordered :column="2" label-placement="left" size="small">
            <n-descriptions-item :label="t('settings_api_addr')">
              <n-tag type="info" :bordered="false" size="small">{{ apiBase }}/api</n-tag>
            </n-descriptions-item>
            <n-descriptions-item :label="t('settings_version')">v1.0.0</n-descriptions-item>
            <n-descriptions-item :label="t('settings_receive_method')">
              <n-tag type="success" :bordered="false" size="small">Cloudflare Email Worker → HTTP POST</n-tag>
            </n-descriptions-item>
            <n-descriptions-item :label="t('settings_database')">
              SQLite (WAL)
              <n-button v-if="isSuperAdmin" size="tiny" type="info" quaternary @click="downloadDB" style="margin-left: 8px">
                {{ t('settings_download_backup') }}
              </n-button>
            </n-descriptions-item>
          </n-descriptions>
        </div>
      </n-gi>

      <!-- 系统配置 -->
      <n-gi v-if="isSuperAdmin">
        <div class="config-card">
          <div class="card-header">
            <span class="card-title">{{ t('settings_system_config') }}</span>
          </div>
          <div class="settings-grid">
            <!-- LEFT COLUMN: Email & Limits -->
            <div class="settings-col">
              <n-form :model="settings" label-placement="top" size="small">
                <n-form-item :label="t('settings_retention_days')">
                  <div style="width: 100%">
                    <n-input-number v-model:value="settings.mail_retention_days" :min="0" :max="365"
                      placeholder="0" style="width: 100%" />
                    <div class="setting-hint">{{ t('settings_retention_hint') }}</div>
                  </div>
                </n-form-item>
                <n-form-item :label="t('settings_temp_expiry')">
                  <n-input-number v-model:value="settings.temp_mailbox_expiry_months" :min="1" :max="60"
                    style="width: 100%" />
                </n-form-item>
                <n-form-item :label="t('settings_temp_domains')">
                  <div style="width: 100%">
                    <n-select v-model:value="settings.temp_email_domains" :options="domainOptions" multiple
                      :placeholder="t('settings_temp_domains_ph')" style="width: 100%" filterable clearable />
                    <div class="setting-hint">{{ t('settings_temp_domains_hint') }}</div>
                  </div>
                </n-form-item>
                <n-form-item :label="t('settings_per_ip_daily')">
                  <div style="width: 100%">
                    <n-input-number v-model:value="settings.temp_mailbox_per_ip_daily" :min="0" :max="100"
                      style="width: 100%" />
                    <div class="setting-hint">{{ t('settings_per_ip_daily_hint') }}</div>
                  </div>
                </n-form-item>
                <n-form-item :label="t('settings_daily_total')">
                  <div style="width: 100%">
                    <n-input-number v-model:value="settings.temp_mailbox_daily_total" :min="0" :max="10000"
                      style="width: 100%" />
                    <div class="setting-hint">{{ t('settings_daily_total_hint') }}</div>
                  </div>
                </n-form-item>
              </n-form>
            </div>
            <!-- RIGHT COLUMN: Telegram + Turnstile -->
            <div class="settings-col">
              <n-form :model="settings" label-placement="top" size="small">
                <n-form-item label="🤖 Telegram Bot Token">
                  <div style="width: 100%">
                    <n-input v-model:value="settings.telegram_bot_token" type="password"
                      show-password-on="click" placeholder="123456:ABC-DEF..." style="width: 100%" />
                    <div class="setting-hint">
                      <n-popover trigger="click" placement="bottom" style="max-width: 380px">
                        <template #trigger>
                          <a style="cursor: pointer; color: var(--primary-color)">{{ t('settings_telegram_guide_title') }}</a>
                        </template>
                        <ol style="padding-left: 18px; font-size: 12px; line-height: 2; margin: 0">
                          <li>{{ t('settings_telegram_step1') }} <a href="https://t.me/BotFather" target="_blank">@BotFather ↗</a></li>
                          <li>{{ t('settings_telegram_step2') }}</li>
                          <li>{{ t('settings_telegram_step3') }}</li>
                          <li>{{ t('settings_telegram_step4') }}</li>
                        </ol>
                        <div style="margin-top: 6px; font-size: 11px; color: var(--text-secondary)">
                          {{ t('settings_telegram_commands') }}:
                          <code>/new</code> <code>/check</code> <code>/code</code> <code>/status</code>
                        </div>
                      </n-popover>
                    </div>
                  </div>
                </n-form-item>
                <n-form-item label="🛡️ Turnstile Site Key">
                  <n-input v-model:value="settings.turnstile_site_key" placeholder="0x4AAAA..." style="width: 100%" />
                </n-form-item>
                <n-form-item label="🔐 Turnstile Secret Key">
                  <div style="width: 100%">
                    <n-input v-model:value="settings.turnstile_secret_key" type="password"
                      show-password-on="click" placeholder="0x4AAAA..." style="width: 100%" />
                    <div class="setting-hint">{{ t('settings_turnstile_hint') }}</div>
                  </div>
                </n-form-item>
              </n-form>
            </div>
          </div>
          <div style="margin-top: 8px">
            <n-button type="primary" size="small" @click="saveSettings" :loading="saving">
              {{ t('settings_save') }}
            </n-button>
          </div>
        </div>
      </n-gi>

      <!-- 服务器状态 -->
      <n-gi v-if="isSuperAdmin">
        <div class="config-card">
          <div class="card-header" style="display: flex; justify-content: space-between; align-items: center">
            <span class="card-title">{{ t('settings_server_status') }}</span>
            <n-button size="tiny" quaternary @click="fetchStatus" :loading="statusLoading">🔄 {{ t('settings_refresh') }}</n-button>
          </div>
          <n-spin :show="statusLoading">
            <n-grid :cols="3" :x-gap="16" :y-gap="12" v-if="sysStatus">
              <n-gi>
                <n-statistic :label="t('settings_uptime')">
                  <template #default>{{ sysStatus.uptime || '-' }}</template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic :label="t('settings_cpu_load')">
                  <template #default>{{ sysStatus.load_1m || '-' }} / {{ sysStatus.load_5m || '-' }} / {{ sysStatus.load_15m || '-' }}</template>
                  <template #suffix><span style="font-size: 11px; color: var(--text-secondary)">1m / 5m / 15m</span></template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic :label="t('settings_memory')">
                  <template #default>{{ sysStatus.mem_used_mb || 0 }} MB</template>
                  <template #suffix><span style="font-size: 11px; color: var(--text-secondary)"> / {{ sysStatus.mem_total_mb || 0 }} MB ({{ sysStatus.mem_percent || 0 }}%)</span></template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic :label="t('settings_disk')">
                  <template #default>{{ sysStatus.disk_used || '-' }}</template>
                  <template #suffix><span style="font-size: 11px; color: var(--text-secondary)"> / {{ sysStatus.disk_total || '-' }} ({{ sysStatus.disk_percent || '-' }})</span></template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic :label="t('settings_go_mem')">
                  <template #default>{{ sysStatus.go_alloc_mb || '-' }} MB</template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic :label="t('settings_goroutines')">
                  <template #default>{{ sysStatus.go_goroutines || '-' }}</template>
                </n-statistic>
              </n-gi>
            </n-grid>
          </n-spin>
        </div>
      </n-gi>

      <!-- 应用日志 -->
      <n-gi v-if="isSuperAdmin">
        <div class="config-card">
          <div class="card-header" style="display: flex; justify-content: space-between; align-items: center">
            <span class="card-title">{{ t('settings_logs') }} <span class="setting-hint">{{ t('settings_logs_hint') }}</span></span>
            <n-button size="tiny" quaternary @click="fetchLogs" :loading="logsLoading">🔄 {{ t('settings_load_logs') }}</n-button>
          </div>
          <n-spin :show="logsLoading">
            <pre class="log-box" v-if="logContent">{{ logContent }}</pre>
            <n-empty v-else :description="t('settings_load_logs')" />
          </n-spin>
        </div>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { getDashboard, getSystemSettings, updateSystemSettings, listDomains, getSystemStatus, getSystemLogs } from '../api'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const { t } = useI18n()

const auth = useAuthStore()
const message = useMessage()
const isSuperAdmin = computed(() => auth.admin?.role === 'super_admin')
const stats = ref({})
const apiBase = window.location.origin
const saving = ref(false)

const settings = ref({
  mail_retention_days: 7,
  temp_mailbox_expiry_months: 3,
  temp_email_domains: [],
  temp_mailbox_per_ip_daily: 3,
  temp_mailbox_daily_total: 0,
  telegram_bot_token: '',
  turnstile_site_key: '',
  turnstile_secret_key: '',
})
const domainOptions = ref([])

// 服务器状态
const sysStatus = ref(null)
const statusLoading = ref(false)

// 日志
const logContent = ref('')
const logsLoading = ref(false)

onMounted(async () => {
  try {
    const { data } = await getDashboard()
    stats.value = data
  } catch {}
  if (isSuperAdmin.value) {
    try {
      const { data: domData } = await listDomains()
      domainOptions.value = domData.data.map(d => ({ label: d.name, value: d.name }))
    } catch {}
    try {
      const { data } = await getSystemSettings()
      if (data.data) {
        settings.value.mail_retention_days = parseInt(data.data.mail_retention_days) || 7
        settings.value.temp_mailbox_expiry_months = parseInt(data.data.temp_mailbox_expiry_months) || 3
        const raw = data.data.temp_email_domains || ''
        settings.value.temp_email_domains = raw ? raw.split(',').map(s => s.trim()).filter(Boolean) : []
        settings.value.temp_mailbox_per_ip_daily = parseInt(data.data.temp_mailbox_per_ip_daily) || 3
        settings.value.temp_mailbox_daily_total = parseInt(data.data.temp_mailbox_daily_total) || 0
        settings.value.telegram_bot_token = data.data.telegram_bot_token || ''
        settings.value.turnstile_site_key = data.data.turnstile_site_key || ''
        settings.value.turnstile_secret_key = data.data.turnstile_secret_key || ''
      }
    } catch {}
    fetchStatus()
  }
})

async function fetchStatus() {
  statusLoading.value = true
  try {
    const { data } = await getSystemStatus()
    sysStatus.value = data.data
  } catch {} finally { statusLoading.value = false }
}

async function fetchLogs() {
  logsLoading.value = true
  try {
    const { data } = await getSystemLogs(200)
    logContent.value = data.data
  } catch {} finally { logsLoading.value = false }
}

async function saveSettings() {
  // Turnstile key validation
  const sk = settings.value.turnstile_site_key?.trim() || ''
  const sec = settings.value.turnstile_secret_key?.trim() || ''
  if ((sk && !sec) || (!sk && sec)) {
    message.warning(t('settings_turnstile_pair_error'))
    return
  }
  if (sk && !sk.startsWith('0x')) {
    message.warning(t('settings_turnstile_site_format'))
    return
  }
  if (sec && !sec.startsWith('0x')) {
    message.warning(t('settings_turnstile_secret_format'))
    return
  }

  saving.value = true
  try {
    await updateSystemSettings({
      mail_retention_days: String(settings.value.mail_retention_days),
      temp_mailbox_expiry_months: String(settings.value.temp_mailbox_expiry_months),
      temp_email_domains: settings.value.temp_email_domains.join(','),
      temp_mailbox_per_ip_daily: String(settings.value.temp_mailbox_per_ip_daily),
      temp_mailbox_daily_total: String(settings.value.temp_mailbox_daily_total),
      telegram_bot_token: settings.value.telegram_bot_token,
      turnstile_site_key: settings.value.turnstile_site_key,
      turnstile_secret_key: settings.value.turnstile_secret_key,
    })
    message.success(t('settings_saved'))
  } catch (e) {
    message.error(t('settings_save_fail'))
  } finally { saving.value = false }
}

const downloadDB = () => {
  const token = localStorage.getItem('token')
  const link = document.createElement('a')
  link.href = `${apiBase}/admin/download-db?token=${token}`
  link.download = 'mailer.db'
  link.click()
}

</script>

<style scoped>
.config-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px 24px;
}

.card-header {
  margin-bottom: 16px;
}

.card-title {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 1px;
}

.setting-hint {
  font-size: 11px;
  color: var(--text-secondary);
  display: block;
  width: 100%;
  margin-top: 4px;
  margin-left: 0;
}

.settings-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 32px;
  align-items: start;
}
.settings-col {
  min-width: 0;
}

.log-box {
  background: #0a0e14;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  font-family: 'JetBrains Mono', 'Courier New', monospace;
  font-size: 11px;
  line-height: 1.6;
  color: #a0d0c0;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>

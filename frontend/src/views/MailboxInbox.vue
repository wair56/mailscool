<template>
  <div class="inbox-page">
    <div class="inbox-header">
      <div class="inbox-title">
        <span class="inbox-icon">📬</span>
        <h2>{{ email }}</h2>
        <n-tag v-if="expiresAt" :type="isExpiringSoon ? 'warning' : 'info'" size="small" :bordered="false" style="margin-left: 8px">
          {{ locale === 'zh' ? '到期: ' : 'Expires: ' }}{{ expiresAt }}
        </n-tag>
        <n-button v-if="isTemp" size="tiny" type="success" secondary style="margin-left: 4px" @click="handleRenew" :loading="renewLoading">
          {{ locale === 'zh' ? '⭐ 续期 3 月' : '⭐ Renew 3mo' }}
        </n-button>
      </div>
      <div style="display: flex; align-items: center; gap: 8px">
        <n-input v-if="senderFilter" v-model:value="senderFilter" size="small" :placeholder="locale === 'zh' ? '筛选发件人' : 'Filter sender'" clearable style="width: 200px" @clear="senderFilter = ''; fetchEmails()" @keyup.enter="fetchEmails" />
        <n-button quaternary size="small" @click="fetchEmails" :loading="loading">🔄</n-button>
        <n-button quaternary size="small" @click="handleExport" :loading="exportLoading">📥 {{ locale === 'zh' ? '导出' : 'Export' }}</n-button>
        <select style="background: transparent; color: #5e7290; border: 1px solid rgba(94,114,144,0.3); border-radius: 4px; padding: 2px 6px; font-size: 12px; cursor: pointer" :value="locale" @change="setLocale($event.target.value)"><option v-for="l in availableLocales" :key="l.code" :value="l.code" style="background: #0d1520; color: #c8d6e5">{{ l.label }}</option></select>
        <n-button quaternary type="error" @click="handleLogout">{{ t('inbox_logout') }}</n-button>
      </div>
    </div>

    <n-data-table :columns="columns" :data="emails" :loading="loading" :bordered="false"
      :row-props="rowProps" size="small" />

    <div style="display: flex; justify-content: flex-end; margin-top: 16px">
      <n-pagination v-model:page="page" :page-count="Math.ceil(total / 20)" @update:page="fetchEmails" />
    </div>

    <!-- Email Detail Modal -->
    <n-modal v-model:show="showDetail" style="width: 95vw; max-width: 1400px; max-height: 90vh" preset="card"
      :title="currentEmail?.subject || t('inbox_email_detail')">
      <div v-if="currentEmail">
        <n-descriptions bordered :column="2" size="small" style="margin-bottom: 12px">
          <n-descriptions-item :label="t('inbox_from')">{{ currentEmail.sender }}</n-descriptions-item>
          <n-descriptions-item :label="t('inbox_time')">{{ new Date(currentEmail.received_at).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US') }}</n-descriptions-item>
        </n-descriptions>
        <div v-if="currentEmail.code" style="margin-bottom: 12px">
            <n-tag type="success" size="large">{{ t('inbox_code') }}: {{ currentEmail.code }}</n-tag>
        </div>
        <n-tabs type="line" :default-value="currentEmail.body_html ? 'html' : 'text'">
          <n-tab-pane name="text" :tab="t('inbox_text')">
            <pre class="email-text">{{ currentEmail.body_text || t('inbox_no_text') }}</pre>
          </n-tab-pane>
          <n-tab-pane name="html" :tab="t('inbox_html')" v-if="currentEmail.body_html">
            <div class="email-html" v-html="sanitizedHtml" />
          </n-tab-pane>
        </n-tabs>
      </div>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NTag, NInput, useMessage } from 'naive-ui'
import api from '../api'
import { useI18n } from '../i18n'

const { t, locale, setLocale, availableLocales } = useI18n()

const router = useRouter()
const message = useMessage()
const email = ref(localStorage.getItem('mailbox_email') || '')
const token = ref(localStorage.getItem('mailbox_token') || '')
const emails = ref([])
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const showDetail = ref(false)
const currentEmail = ref(null)
const senderFilter = ref('')
const expiresAt = ref('')
const isExpiringSoon = ref(false)
const isTemp = ref(false)
const renewLoading = ref(false)
const exportLoading = ref(false)

async function handleExport() {
  exportLoading.value = true
  try {
    const { data } = await api.get('/mailbox/export', {
      headers: { Authorization: `Bearer ${token.value}` },
    })
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${email.value}_export.json`
    link.click()
    URL.revokeObjectURL(url)
    message.success(locale.value === 'zh' ? '导出成功' : 'Export successful')
  } catch {
    message.error(locale.value === 'zh' ? '导出失败' : 'Export failed')
  } finally { exportLoading.value = false }
}

const sanitizedHtml = computed(() => {
  if (!currentEmail.value?.body_html) return ''
  return currentEmail.value.body_html.replace(/http:\/\//g, 'https://')
})

function extractEmail(str) {
  if (!str) return ''
  const m = str.match(/<([^>]+@[^>]+)>/)
  return m?.[1] || str
}

if (!token.value) {
  router.push('/login')
}

const mailboxApi = api.create({
  baseURL: '',
  timeout: 15000,
  headers: { Authorization: `Bearer ${token.value}` }
})

const columns = [
  { title: () => t('inbox_from'), key: 'sender', width: 200, ellipsis: { tooltip: true },
    render: row => h('span', {
      style: 'cursor: pointer; color: var(--text-secondary)',
      onClick: (e) => {
        e.stopPropagation()
        senderFilter.value = extractEmail(row.sender)
        fetchEmails()
      }
    }, row.sender)
  },
  {
    title: () => t('email_subject'), key: 'subject', ellipsis: { tooltip: true },
    render: row => h('span', { style: row.is_read ? {} : { fontWeight: 600, color: '#00f0ff' } }, row.subject || t('email_no_subject'))
  },
  {
    title: () => t('inbox_code'), key: 'code', width: 100,
    render: row => row.code ? h(NTag, { type: 'success', bordered: false, size: 'small' }, () => row.code) : '-'
  },
  {
    title: () => t('inbox_time'), key: 'received_at', width: 160,
    render: row => new Date(row.received_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  },
]

function rowProps(row) {
  return { style: 'cursor: pointer', onClick: () => viewEmail(row.id) }
}

async function fetchEmails() {
  loading.value = true
  try {
    const { data } = await mailboxApi.get('/mailbox/emails', { params: { page: page.value, size: 20 } })
    emails.value = data.data
    total.value = data.total
  } catch (e) {
    if (e.response?.status === 401) {
      handleLogout()
    }
  } finally { loading.value = false }
}

async function viewEmail(id) {
  try {
    const { data } = await mailboxApi.get(`/mailbox/emails/${id}`)
    currentEmail.value = data
    showDetail.value = true
  } catch {}
}

function handleLogout() {
  localStorage.removeItem('mailbox_token')
  localStorage.removeItem('mailbox_email')
  router.push('/login')
}

onMounted(() => {
  fetchEmails()
  fetchMailboxInfo()
})

async function fetchMailboxInfo() {
  try {
    const { data } = await mailboxApi.get('/mailbox/me')
    if (data.expires_at) {
      const d = new Date(data.expires_at)
      expiresAt.value = d.toLocaleDateString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
      const daysLeft = (d - new Date()) / 86400000
      isExpiringSoon.value = daysLeft < 7
    }
    isTemp.value = data.is_temp || false
  } catch {}
}

async function handleRenew() {
  renewLoading.value = true
  try {
    const { data } = await mailboxApi.post('/mailbox/renew')
    message.success(locale.value === 'zh' ? `✅ 已续期至 ${data.expires_at}` : `✅ Renewed until ${data.expires_at}`)
    fetchMailboxInfo()
  } catch (e) {
    const msg = e.response?.data?.error || (locale.value === 'zh' ? '续期失败' : 'Renew failed')
    message.error(msg)
  } finally { renewLoading.value = false }
}
</script>

<style scoped>
.inbox-page {
  max-width: 95%;
  margin: 0 auto;
  padding: 24px;
}

.inbox-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px 20px;
  background: var(--bg-card, #0d1520);
  border: 1px solid var(--border-color, #1a2940);
  border-radius: 10px;
}

.inbox-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.inbox-icon {
  font-size: 24px;
}

.inbox-title h2 {
  font-family: 'JetBrains Mono', monospace;
  font-size: 15px;
  font-weight: 500;
  color: #c8d6e5;
}

.email-text {
  background: #0d1520;
  color: #e8e8f0;
  padding: 16px;
  border-radius: 8px;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  max-height: 70vh;
  min-height: 300px;
  overflow-y: auto;
}

.email-html {
  background: #fff;
  color: #333;
  padding: 16px;
  border-radius: 8px;
  max-height: 70vh;
  min-height: 300px;
  overflow-y: auto;
}
</style>

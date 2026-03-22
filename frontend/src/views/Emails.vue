<template>
  <div>
    <!-- 筛选栏 -->
    <div class="filter-bar">
      <n-select v-model:value="filters.domain_id" :options="domainOptions" :placeholder="t('email_filter_domain')"
        clearable style="width: 150px" size="small" @update:value="handleSearch" />
      <n-input v-model:value="filters.to" :placeholder="t('email_filter_to')" clearable style="width: 180px"
        size="small" @keyup.enter="handleSearch" />
      <n-input v-model:value="filters.from" :placeholder="t('email_filter_from')" clearable style="width: 150px"
        size="small" @keyup.enter="handleSearch" />
      <n-button type="primary" size="small" @click="handleSearch">{{ t('email_search') }}</n-button>
      <n-button size="small" quaternary @click="resetFilters">{{ t('email_reset') }}</n-button>
      <n-checkbox v-model:checked="filters.has_code" size="small" @update:checked="handleSearch">
        {{ locale === 'zh' ? '有验证码' : 'Has Code' }}
      </n-checkbox>
      <span class="email-count">{{ total >= 0 ? total : '...' }} {{ locale === 'zh' ? '封' : 'emails' }}</span>
    </div>

    <!-- 黑名单域名 -->
    <div class="blacklist-bar" v-if="excludeDomains.length > 0 || showBlacklistInput">
      <span class="blacklist-label">{{ locale === 'zh' ? '🚫 屏蔽域名:' : '🚫 Blocked:' }}</span>
      <n-tag v-for="(d, i) in excludeDomains" :key="d" closable size="small" type="error"
        @close="removeExcludeDomain(i)" style="margin-right: 4px">{{ d }}</n-tag>
      <n-input v-if="showBlacklistInput" v-model:value="blacklistInput" size="tiny"
        :placeholder="locale === 'zh' ? '输入域名回车添加' : 'domain, Enter to add'"
        style="width: 160px" @keyup.enter="addExcludeDomain" @blur="showBlacklistInput = false" ref="blacklistInputRef" />
      <n-button v-else size="tiny" quaternary @click="showBlacklistInput = true">+</n-button>
    </div>

    <n-data-table :columns="columns" :data="emails" :loading="loading" :bordered="false"
      :row-class-name="rowClassName" :row-props="rowProps" size="small" :scroll-x="1000" />

    <div style="display: flex; justify-content: center; margin-top: 16px" v-if="total > size">
      <n-pagination v-model:page="page" :page-count="Math.ceil(total / size)"
        :page-size="size" show-quick-jumper @update:page="fetchData" />
    </div>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NTag, NCheckbox, useMessage } from 'naive-ui'
import { listEmails, listDomains, toggleEmailStar } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const emails = ref([])
const loading = ref(false)
const page = ref(1)
const size = ref(20)
const total = ref(0)
const domainOptions = ref([])

const filters = ref({ domain_id: null, to: '', from: '', has_code: false })
const excludeDomains = ref(JSON.parse(localStorage.getItem('email_exclude_domains') || '[]'))
const showBlacklistInput = ref(false)
const blacklistInput = ref('')
const RETENTION_DAYS = 7

function addExcludeDomain() {
  const d = blacklistInput.value.trim().toLowerCase()
  if (d && !excludeDomains.value.includes(d)) {
    excludeDomains.value.push(d)
    localStorage.setItem('email_exclude_domains', JSON.stringify(excludeDomains.value))
    page.value = 1
    fetchData()
  }
  blacklistInput.value = ''
  showBlacklistInput.value = false
}

function removeExcludeDomain(index) {
  excludeDomains.value.splice(index, 1)
  localStorage.setItem('email_exclude_domains', JSON.stringify(excludeDomains.value))
  page.value = 1
  fetchData()
}

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  {
    title: '⭐', key: 'star', width: 45,
    render: row => h(NButton, {
      text: true, size: 'small',
      style: { fontSize: '16px' },
      onClick: (e) => { e.stopPropagation(); handleToggleStar(row) }
    }, () => row.is_starred ? '⭐' : '☆')
  },
  {
    title: () => t('email_recipient'), key: 'recipient', width: 220, ellipsis: { tooltip: true },
    render: row => h('span', {
      style: 'color: #4dd0e1; cursor: pointer; font-size: 13px',
      onClick: (e) => {
        e.stopPropagation()
        filterByMailbox(row.recipient)
      }
    }, row.recipient)
  },
  { title: () => t('email_sender'), key: 'sender', width: 200, ellipsis: { tooltip: true },
    render: row => {
      const addr = extractEmailAddress(row.sender)
      const domain = addr.split('@')[1] || ''
      return h('span', {
        style: 'font-size: 13px; color: var(--text-secondary); cursor: pointer',
        onClick: (e) => {
          e.stopPropagation()
          filterBySender(row.sender)
        },
        onContextmenu: (e) => {
          e.preventDefault()
          e.stopPropagation()
          if (domain && !excludeDomains.value.includes(domain)) {
            excludeDomains.value.push(domain)
            localStorage.setItem('email_exclude_domains', JSON.stringify(excludeDomains.value))
            page.value = 1
            fetchData()
          }
        }
      }, row.sender)
    }
  },
  {
    title: () => t('email_subject'), key: 'subject', ellipsis: { tooltip: true },
    render: row => h('span', { style: row.is_read ? { fontSize: '13px' } : { fontWeight: 600, fontSize: '13px' } }, row.subject || t('email_no_subject'))
  },
  {
    title: () => t('email_code'), key: 'code', width: 100,
    render: row => row.code ? h(NTag, { type: 'success', bordered: false, size: 'small' }, () => row.code) : h('span', { style: 'color: var(--text-secondary); font-size: 12px' }, '-')
  },
  {
    title: () => t('email_time'), key: 'received_at', width: 145,
    render: row => h('span', { style: 'font-size: 12px; color: var(--text-secondary)' }, new Date(row.received_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US'))
  },
  {
    title: () => t('email_expires'), key: 'expires', width: 100,
    render: row => {
      if (row.is_starred) return h('span', { style: 'font-size: 12px; color: #00c853' }, t('email_never'))
      const received = new Date(row.received_at)
      const expires = new Date(received.getTime() + RETENTION_DAYS * 86400000)
      const now = new Date()
      const hoursLeft = (expires - now) / 3600000
      const color = hoursLeft < 0 ? '#e74c3c' : hoursLeft < 24 ? '#f39c12' : 'var(--text-secondary)'
      const text = hoursLeft < 0 ? t('email_expired') : `${Math.ceil(hoursLeft / 24)}${t('email_days')}`
      return h('span', { style: `font-size: 12px; color: ${color}` }, text)
    }
  },
]

function rowClassName(row) {
  return row.is_read ? 'table-row read' : 'table-row unread'
}

function rowProps(row) {
  return {
    style: 'cursor: pointer',
    onClick: () => {
      router.push({ name: 'EmailDetail', params: { id: row.id } })
    }
  }
}

function extractEmailAddress(recipient) {
  if (!recipient) return ''
  const bracketMatch = recipient.match(/<([^>]+@[^>]+)>/)
  if (bracketMatch?.[1]) return bracketMatch[1]

  const plainMatch = recipient.match(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/i)
  return plainMatch?.[0] || recipient
}

function filterByMailbox(recipient) {
  filters.value.to = extractEmailAddress(recipient)
  page.value = 1
  fetchData()
}

function filterBySender(sender) {
  filters.value.from = extractEmailAddress(sender)
  page.value = 1
  fetchData()
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function resetFilters() {
  filters.value = { domain_id: null, to: '', from: '', has_code: false }
  page.value = 1
  fetchData()
}

function initFiltersFromRoute() {
  const q = route.query

  if (q.domain_id !== undefined) {
    const parsed = Number(q.domain_id)
    filters.value.domain_id = Number.isNaN(parsed) ? null : parsed
  }
  if (typeof q.to === 'string') {
    filters.value.to = q.to
  }
  if (typeof q.from === 'string') {
    filters.value.from = q.from
  }
  if (q.page !== undefined) {
    const parsed = Number(q.page)
    if (!Number.isNaN(parsed) && parsed > 0) page.value = parsed
  }
}

async function fetchDomains() {
  try {
    const { data } = await listDomains()
    domainOptions.value = data.data.map(d => ({ label: d.name, value: d.id }))
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (filters.value.domain_id) params.domain_id = filters.value.domain_id
    if (filters.value.to) params.to = filters.value.to
    if (filters.value.from) params.from = filters.value.from
    if (filters.value.has_code) params.has_code = '1'
    if (excludeDomains.value.length > 0) params.exclude_domains = excludeDomains.value.join(',')

    // Phase 1: 先加载数据（跳过 COUNT），立即展示
    const { data } = await listEmails({ ...params, skip_count: '1' })
    emails.value = data.data
    loading.value = false

    // Phase 2: 异步获取总数（用于分页）
    listEmails(params).then(res => {
      total.value = res.data.total
    }).catch(() => {})
  } catch {} finally { loading.value = false }
}

onMounted(() => {
  initFiltersFromRoute()
  fetchDomains()
  fetchData()
})

async function handleToggleStar(row) {
  try {
    const { data } = await toggleEmailStar(row.id)
    row.is_starred = data.is_starred
  } catch {}
}
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
  padding: 10px 16px;
  background: var(--bg-card);
  border-radius: 10px;
  border: 1px solid var(--border-color);
  flex-wrap: wrap;
}

.blacklist-bar {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 16px;
  padding: 6px 16px;
  background: var(--bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-color);
  flex-wrap: wrap;
  opacity: 0.85;
}

.blacklist-label {
  font-size: 12px;
  color: var(--text-secondary);
  margin-right: 4px;
}

.email-count {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-secondary);
  font-family: 'JetBrains Mono', monospace;
}

:deep(.table-row) { background: var(--bg-card); }
:deep(.table-row:hover td) { background: var(--bg-hover) !important; }
:deep(.table-row.unread td:first-child) { border-left: 3px solid #00f0ff; }
</style>

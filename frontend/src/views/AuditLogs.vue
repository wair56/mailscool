<template>
  <div>
    <div class="filter-bar">
      <n-input v-model:value="filters.username" :placeholder="t('audit_filter_user')" clearable
        style="width: 160px" size="small" @keyup.enter="handleSearch" />
      <n-input v-model:value="filters.action" :placeholder="t('audit_filter_action')" clearable
        style="width: 160px" size="small" @keyup.enter="handleSearch" />
      <n-button type="primary" size="small" @click="handleSearch">{{ t('email_search') }}</n-button>
      <n-button size="small" quaternary @click="resetFilters">{{ t('email_reset') }}</n-button>
      <span class="list-count">{{ total }} {{ t('audit_count_label') }}</span>
    </div>

    <n-data-table :columns="columns" :data="logs" :loading="loading" :bordered="false"
      :row-class-name="() => 'table-row'" />

    <div style="display: flex; justify-content: center; margin-top: 16px" v-if="total > size">
      <n-pagination v-model:page="page" :page-count="Math.ceil(total / size)"
        :page-size="size" show-quick-jumper @update:page="fetchData" />
    </div>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NTag } from 'naive-ui'
import { listAuditLogs } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()
const logs = ref([])
const loading = ref(false)
const page = ref(1)
const size = ref(50)
const total = ref(0)
const filters = ref({ username: '', action: '' })

function handleSearch() { page.value = 1; fetchData() }
function resetFilters() { filters.value = { username: '', action: '' }; page.value = 1; fetchData() }

const actionColors = {
  login: 'info',
  create_domain: 'success',
  delete_domain: 'error',
  toggle_domain: 'warning',
  create_api_key: 'success',
  delete_api_key: 'error',
  toggle_api_key: 'warning',
  create_admin: 'success',
  delete_admin: 'error',
  change_password: 'info',
}

const columns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: () => t('audit_operator'), key: 'username', width: 100 },
  { title: () => t('audit_action'), key: 'action', width: 140,
    render: row => h(NTag, {
      type: actionColors[row.action] || 'default',
      size: 'small',
      bordered: false
    }, () => row.action)
  },
  { title: () => t('audit_target'), key: 'target', ellipsis: { tooltip: true } },
  { title: () => t('audit_detail'), key: 'detail', ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip', width: 140 },
  { title: () => t('audit_time'), key: 'created_at', width: 170,
    render: row => new Date(row.created_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  },
]

async function fetchData() {
  loading.value = true
  try {
    const { data } = await listAuditLogs({ page: page.value, size: size.value })
    logs.value = data.data
    total.value = data.total
  } catch {} finally { loading.value = false }
}

onMounted(fetchData)
</script>

<style scoped>
.filter-bar {
  display: flex; gap: 10px; align-items: center; margin-bottom: 16px;
  padding: 10px 16px; background: var(--bg-card); border-radius: 10px;
  border: 1px solid var(--border-color); flex-wrap: wrap;
}
.list-count { margin-left: auto; font-size: 12px; color: var(--text-secondary); font-family: 'JetBrains Mono', monospace; }
:deep(.table-row) { background: #1a1a2e; }
:deep(.table-row:hover td) { background: #252540 !important; }
</style>

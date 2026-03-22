<template>
  <div>
    <div class="filter-bar">
      <n-input v-model:value="filterText" :placeholder="t('apikey_filter_ph')" clearable
        style="width: 240px" size="small" />
      <n-button size="small" quaternary @click="filterText = ''">{{ t('email_reset') }}</n-button>
      <span class="list-count">{{ filteredKeys.length }} {{ t('apikey_count_label') }}</span>
      <n-button type="primary" size="small" @click="showCreateModal = true">{{ t('apikey_create') }}</n-button>
    </div>

    <n-data-table :columns="columns" :data="filteredKeys" :loading="loading" :bordered="false" :scroll-x="1000"
      :row-class-name="() => 'table-row'" />

    <!-- 创建 API Key 对话框 -->
    <n-modal v-model:show="showCreateModal" preset="dialog" :title="t('apikey_create_title')"
      :positive-text="t('common_create')" :negative-text="t('common_cancel')" @positive-click="handleCreate" style="width: 560px">
      <n-form :model="createForm" label-placement="left" label-width="100">
        <n-form-item :label="t('apikey_name')">
          <n-input v-model:value="createForm.name" :placeholder="t('apikey_name_ph')" />
        </n-form-item>
        <n-form-item :label="t('apikey_domains')">
          <n-select v-model:value="createForm.domain_ids" :options="domainOptions"
            multiple :placeholder="t('apikey_domains_ph')" />
        </n-form-item>
        <n-form-item :label="t('apikey_rate')">
          <n-input-number v-model:value="createForm.rate_limit" :min="1" :max="10000"
            :placeholder="t('apikey_rate_ph')" style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('apikey_expires')">
          <n-date-picker v-model:value="createForm.expires_at_ts" type="date"
            clearable :placeholder="t('apikey_expires_ph')" style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('apikey_ip')">
          <n-input v-model:value="createForm.ip_whitelist"
            :placeholder="t('apikey_ip_ph')" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 编辑 API Key 对话框 -->
    <n-modal v-model:show="showEditModal" preset="dialog" :title="t('apikey_edit_title')"
      :positive-text="t('common_save')" :negative-text="t('common_cancel')" @positive-click="handleUpdate" style="width: 560px">
      <n-form :model="editForm" label-placement="left" label-width="100">
        <n-form-item :label="t('apikey_name')">
          <n-input v-model:value="editForm.name" />
        </n-form-item>
        <n-form-item :label="t('apikey_domains')">
          <n-select v-model:value="editForm.domain_ids" :options="domainOptions"
            multiple :placeholder="t('apikey_domains_ph')" />
        </n-form-item>
        <n-form-item :label="t('apikey_rate')">
          <n-input-number v-model:value="editForm.rate_limit" :min="1" :max="10000"
            style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('apikey_expires')">
          <n-date-picker v-model:value="editForm.expires_at_ts" type="date"
            clearable :placeholder="t('apikey_expires_ph')" style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('apikey_ip')">
          <n-input v-model:value="editForm.ip_whitelist"
            :placeholder="t('apikey_ip_ph')" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 新建 Key 结果 -->
    <n-modal v-model:show="showKeyResultModal" preset="dialog" :title="t('apikey_created_title')" type="success">
      <n-alert type="warning" :title="t('apikey_created_warning')">
        {{ t('apikey_created_notice') }}
      </n-alert>
      <n-input :value="newKey" readonly style="margin-top: 12px; font-family: monospace" />
      <n-button style="margin-top: 8px" @click="copyKey" block>{{ t('apikey_copy') }}</n-button>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { NButton, NSwitch, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listApiKeys, createApiKey, updateApiKey, toggleApiKey, deleteApiKey, listDomains } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()

const message = useMessage()
const dialog = useDialog()
const apiKeys = ref([])
const loading = ref(false)
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showKeyResultModal = ref(false)
const newKey = ref('')
const domainOptions = ref([])
const filterText = ref('')

const filteredKeys = computed(() => {
  if (!filterText.value) return apiKeys.value
  const kw = filterText.value.toLowerCase()
  return apiKeys.value.filter(k =>
    (k.name || '').toLowerCase().includes(kw) ||
    (k.key_prefix || '').toLowerCase().includes(kw) ||
    (k.domains || []).some(d => d.name.toLowerCase().includes(kw))
  )
})

const createForm = ref({
  name: '', domain_ids: [], rate_limit: 100, ip_whitelist: '', expires_at_ts: null
})

const editForm = ref({
  id: null, name: '', domain_ids: [], rate_limit: 100, ip_whitelist: '', expires_at_ts: null
})

const columns = [
  { title: 'ID', key: 'id', width: 45 },
  { title: () => t('apikey_col_name'), key: 'name', width: 120, ellipsis: { tooltip: true },
    render: row => row.is_system
      ? h('span', { style: 'display:flex;align-items:center;gap:4px' }, [
          h(NTag, { size: 'tiny', type: 'warning', bordered: false }, () => '🔒'),
          row.name
        ])
      : row.name
  },
  { title: () => t('apikey_col_domains'), key: 'domains', width: 120,
    render: row => h(NSpace, { size: 4, wrap: true }, () =>
      (row.domains || []).map(d => h(NTag, { size: 'small', bordered: false }, () => d.name))
    )
  },
  { title: 'KEY', key: 'key_prefix', width: 150,
    render: row => {
      const full = row.key_plain || row.key_prefix || ''
      const short = full.length > 16 ? full.slice(0, 16) + '…' : full
      return h('span', {
        style: 'display: inline-flex; align-items: center; gap: 4px; cursor: pointer',
        title: 'Copy',
        onClick: () => { navigator.clipboard.writeText(full); message.success(t('apikey_copied')) }
      }, [
        h('code', { style: 'color: #a29bfe; font-size: 11px' }, short),
        h('span', { style: 'font-size: 10px; opacity: 0.4' }, '📋')
      ])
    }
  },
  { title: () => t('apikey_col_emails'), key: 'total_emails', width: 55,
    render: row => h(NTag, { type: row.total_emails > 0 ? 'success' : 'default', bordered: false, size: 'small' }, () => row.total_emails ?? 0)
  },
  { title: () => t('apikey_col_mailboxes'), key: 'total_mailboxes', width: 55,
    render: row => h(NTag, { type: row.total_mailboxes > 0 ? 'info' : 'default', bordered: false, size: 'small' }, () => row.total_mailboxes ?? 0)
  },
  { title: () => t('apikey_col_expires'), key: 'expires_at', width: 90,
    render: row => row.expires_at
      ? h('span', { style: `font-size: 12px; color: ${new Date(row.expires_at) < new Date() ? '#e74c3c' : '#8e9aab'}` },
          new Date(row.expires_at).toLocaleDateString())
      : h('span', { style: 'font-size: 12px; color: #5e7290' }, t('apikey_permanent'))
  },
  { title: () => t('apikey_col_rate'), key: 'rate_limit', width: 70,
    render: row => h('span', { style: 'font-size: 12px' }, `${row.rate_limit}${t('apikey_rate_suffix')}`)
  },
  { title: () => t('apikey_col_status'), key: 'is_active', width: 60,
    render: row => h(NSwitch, {
      value: row.is_active, size: 'small',
      onUpdateValue: () => handleToggle(row)
    })
  },
  { title: () => t('apikey_col_creator'), key: 'created_by_name', width: 80,
    render: row => row.created_by_name
      ? h('span', { style: 'font-size: 12px; color: #8e9aab' }, row.created_by_name)
      : h('span', { style: 'font-size: 12px; color: #5e7290' }, '-')
  },
  { title: () => t('apikey_col_actions'), key: 'actions', width: 100,
    render: row => h('div', { style: 'display: flex; gap: 2px' }, [
      h(NButton, {
        size: 'small', quaternary: true, type: 'info',
        onClick: () => openEdit(row)
      }, () => t('apikey_edit')),
      h(NButton, {
        size: 'small', quaternary: true, type: 'error',
        onClick: () => handleDelete(row)
      }, () => t('apikey_delete')),
    ])
  },
]

function openEdit(row) {
  editForm.value = {
    id: row.id,
    name: row.name || '',
    domain_ids: (row.domains || []).map(d => d.id),
    rate_limit: row.rate_limit || 100,
    ip_whitelist: row.ip_whitelist || '',
    expires_at_ts: row.expires_at ? new Date(row.expires_at).getTime() : null
  }
  showEditModal.value = true
}

async function handleUpdate() {
  const payload = {
    name: editForm.value.name,
    rate_limit: editForm.value.rate_limit,
    ip_whitelist: editForm.value.ip_whitelist,
    domain_ids: editForm.value.domain_ids,
    expires_at: editForm.value.expires_at_ts
      ? new Date(editForm.value.expires_at_ts).toISOString().split('T')[0]
      : ''
  }
  try {
    await updateApiKey(editForm.value.id, payload)
    message.success(t('apikey_updated'))
    showEditModal.value = false
    fetchData()
  } catch (e) {
    message.error(e.response?.data?.error || t('apikey_update_fail'))
    return false
  }
}

async function fetchData() {
  loading.value = true
  try {
    const { data } = await listApiKeys()
    apiKeys.value = data.data
  } catch {} finally { loading.value = false }
}

async function fetchDomains() {
  try {
    const { data } = await listDomains()
    domainOptions.value = data.data.map(d => ({ label: d.name, value: d.id }))
  } catch {}
}

async function handleCreate() {
  if (!createForm.value.name || !createForm.value.domain_ids.length) {
    message.warning(t('apikey_require_fields'))
    return false
  }
  try {
    const payload = { ...createForm.value }
    if (payload.expires_at_ts) {
      payload.expires_at = new Date(payload.expires_at_ts).toISOString()
    }
    delete payload.expires_at_ts
    const { data } = await createApiKey(payload)
    newKey.value = data.key
    showKeyResultModal.value = true
    createForm.value = { name: '', domain_ids: [], rate_limit: 100, ip_whitelist: '', expires_at_ts: null }
    fetchData()
  } catch (e) { message.error(e.response?.data?.error || t('apikey_create_fail')); return false }
}

async function handleToggle(row) {
  try {
    const { data } = await toggleApiKey(row.id)
    row.is_active = data.is_active
  } catch {}
}

function handleDelete(row) {
  const isSystem = row.is_system
  dialog.warning({
    title: t('apikey_confirm_delete'),
    content: isSystem
      ? (locale.value === 'zh'
        ? `⚠️ "${row.name}" 是系统自动创建的 Key，删除后对应的 CF Worker 将无法收件！确认删除后请重新运行「一键配置」。`
        : `⚠️ "${row.name}" is a system key. Deleting it will break the CF Worker! Re-run auto setup after deletion.`)
      : `${t('apikey_delete')} "${row.name}"?`,
    positiveText: t('common_delete'),
    negativeText: t('common_cancel'),
    onPositiveClick: async () => {
      try {
        await deleteApiKey(row.id)
        message.success(t('apikey_deleted'))
        fetchData()
      } catch {}
    }
  })
}

function copyKey() {
  const text = newKey.value
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(() => message.success(t('apikey_copied_clipboard')))
    return;
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.cssText = 'position:fixed;left:-9999px;top:-9999px;opacity:0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    ta.setSelectionRange(0, text.length)
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    if (ok) { message.success(t('apikey_copied_clipboard')); return }
  } catch (e) { /* fallthrough */ }
  window.prompt(t('apikey_copy_manual'), text)
}

onMounted(() => { fetchDomains(); fetchData() })
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

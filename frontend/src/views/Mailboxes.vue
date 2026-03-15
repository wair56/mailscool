<template>
  <div>
    <div class="filter-bar">
      <n-input v-model:value="filterText" :placeholder="t('mailbox_filter_ph')" clearable
        style="width: 220px" size="small" />
      <n-button size="small" quaternary @click="filterText = ''">{{ t('email_reset') }}</n-button>
      <span class="list-count">{{ filteredMailboxes.length }} {{ t('mailbox_count_label') }}</span>
      <n-button type="primary" size="small" @click="showCreate = true">{{ t('mailbox_create') }}</n-button>
    </div>

    <n-data-table :columns="columns" :data="filteredMailboxes" :loading="loading" :bordered="false" />

    <!-- 创建邮箱 -->
    <n-modal v-model:show="showCreate" preset="dialog" :title="t('mailbox_create_title')" :positive-text="t('mailbox_create_btn')"
      @positive-click="handleCreate" style="width: 440px">
      <n-form :model="createForm" label-placement="left" label-width="80">
        <n-form-item :label="t('mailbox_create_prefix')">
          <n-input-group>
            <n-input v-model:value="createForm.prefix" :placeholder="t('mailbox_create_prefix_ph')" style="width: 40%" />
            <n-input-group-label>@</n-input-group-label>
            <n-select v-model:value="createForm.domain" :options="domainOptions" :placeholder="t('mailbox_create_domain')" style="width: 60%" />
          </n-input-group>
        </n-form-item>
        <n-form-item :label="t('mailbox_password')">
          <n-input v-model:value="createForm.password" :placeholder="t('mailbox_password')" />
        </n-form-item>
        <n-form-item :label="t('mailbox_create_webhook')">
          <n-input v-model:value="createForm.webhook_url" :placeholder="t('mailbox_create_webhook_ph')" clearable />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 编辑邮箱类型/有效期/Webhook -->
    <n-modal v-model:show="showEdit" preset="dialog" :title="t('mailbox_edit_title')" :positive-text="t('mailbox_save')"
      @positive-click="handleUpdate" style="width: 480px">
      <n-form :model="editForm" label-placement="left" label-width="100">
        <n-form-item :label="t('mailbox_email')">
          <n-input :value="editForm.email" readonly />
        </n-form-item>
        <n-form-item :label="t('mailbox_type')">
          <n-select v-model:value="editForm.is_temp" :options="typeOptions" />
        </n-form-item>
        <n-form-item :label="t('mailbox_expires')">
          <n-date-picker v-model:value="editForm.expires_at_ts" type="date" clearable
            style="width: 100%" />
        </n-form-item>
        <n-form-item label="Webhook URL">
          <n-input v-model:value="editForm.webhook_url" placeholder="https://your.hook/endpoint" clearable />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NTag, NTooltip, useMessage, useDialog } from 'naive-ui'
import { listMailboxes, createMailbox, updateMailbox, deleteMailbox, listDomains } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()

const router = useRouter()
const route = useRoute()
const message = useMessage()
const dialog = useDialog()
const mailboxes = ref([])
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const createForm = ref({ prefix: '', domain: '', password: '', webhook_url: '' })
const editForm = ref({ id: null, email: '', is_temp: false, expires_at_ts: null, webhook_url: '' })
const filterText = ref('')
const domainOptions = ref([])

const filteredMailboxes = computed(() => {
  if (!filterText.value) return mailboxes.value
  const kw = filterText.value.toLowerCase()
  return mailboxes.value.filter(m =>
    (m.email || '').toLowerCase().includes(kw) ||
    (m.domain_name || '').toLowerCase().includes(kw)
  )
})

const typeOptions = [
  { label: () => t('mailbox_type_long'), value: false },
  { label: () => t('mailbox_type_temp'), value: true }
]

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: () => t('mailbox_email'), key: 'email', width: 240 },
  {
    title: () => t('mailbox_password'), key: 'password_plain', width: 160,
    render: row => {
      if (!row.password_plain) return '-'
      return h('span', {
        style: 'cursor: pointer; color: #5e7290; font-family: JetBrains Mono, monospace; font-size: 12px',
        class: 'pwd-mask',
        onMouseenter: (e) => { e.target.textContent = row.password_plain },
        onMouseleave: (e) => { e.target.textContent = '••••••••' }
      }, '••••••••')
    }
  },
  { title: () => t('mailbox_domain'), key: 'domain_name', width: 140 },
  {
    title: () => t('mailbox_type'), key: 'is_temp', width: 80,
    render: row => row.is_temp
      ? h(NTag, { type: 'warning', bordered: false, size: 'small' }, () => t('mailbox_type_temp'))
      : h(NTag, { type: 'info', bordered: false, size: 'small' }, () => t('mailbox_type_long'))
  },
  {
    title: () => t('mailbox_expires'), key: 'expires_at', width: 170,
    render: row => row.expires_at ? new Date(row.expires_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US') : '-'
  },
  {
    title: () => t('mailbox_email_count'), key: 'total_emails', width: 80,
    render: row => h(NTag, {
      type: row.total_emails > 0 ? 'success' : 'default',
      bordered: false,
      size: 'small',
      style: row.total_emails > 0 ? 'cursor: pointer' : '',
      onClick: () => {
        if (row.total_emails < 1) return
        router.push({
          path: '/emails',
          query: {
            domain_id: row.domain_id,
            to: row.email
          }
        })
      }
    }, () => row.total_emails)
  },
  {
    title: () => t('mailbox_created_at'), key: 'created_at', width: 170,
    render: row => new Date(row.created_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  },
  {
    title: () => t('mailbox_source'), key: 'created_ip', width: 140,
    render: row => {
      const ip = row.created_ip || '-'
      const ua = row.created_ua || ''
      if (!ua) return h('span', { style: 'font-size: 12px; color: #8c8c8c' }, ip)
      return h(NTooltip, { trigger: 'hover' }, {
        trigger: () => h('span', { style: 'font-size: 12px; color: #5e7290; cursor: help; text-decoration: underline dotted' }, ip),
        default: () => h('div', { style: 'max-width: 360px; word-break: break-all; font-size: 12px' }, ua)
      })
    }
  },
  {
    title: 'Webhook', key: 'webhook_url', width: 180, ellipsis: { tooltip: true },
    render: row => {
      if (!row.webhook_url) return h('span', { style: 'color: #555; font-size: 12px' }, '-')
      return h(NTooltip, { trigger: 'hover' }, {
        trigger: () => h('span', { style: 'font-size: 11px; color: #0aff9d; cursor: help; font-family: JetBrains Mono, monospace' }, '🔗 ' + (row.webhook_url.length > 20 ? row.webhook_url.slice(0, 20) + '…' : row.webhook_url)),
        default: () => h('div', { style: 'max-width: 400px; word-break: break-all; font-size: 12px' }, row.webhook_url)
      })
    }
  },
  {
    title: () => t('mailbox_actions'), key: 'actions', width: 130,
    render: row => h('div', { style: 'display: flex; gap: 4px' }, [
      h(NButton, {
        size: 'small', quaternary: true, type: 'info',
        onClick: () => openEdit(row)
      }, () => t('mailbox_edit')),
      h(NButton, {
        size: 'small', quaternary: true, type: 'error',
        onClick: () => confirmDelete(row)
      }, () => t('mailbox_delete'))
    ])
  }
]

function openEdit(row) {
  editForm.value = {
    id: row.id,
    email: row.email,
    is_temp: row.is_temp,
    expires_at_ts: row.expires_at ? new Date(row.expires_at).getTime() : null,
    webhook_url: row.webhook_url || ''
  }
  showEdit.value = true
}

async function handleUpdate() {
  const data = {
    is_temp: editForm.value.is_temp,
    expires_at: editForm.value.expires_at_ts
      ? new Date(editForm.value.expires_at_ts).toISOString().split('T')[0]
      : '',
    webhook_url: editForm.value.webhook_url
  }
  try {
    await updateMailbox(editForm.value.id, data)
    message.success(t('mailbox_updated'))
    showEdit.value = false
    fetchData()
  } catch (e) {
    message.error(e.response?.data?.error || t('mailbox_update_fail'))
    return false
  }
}

async function fetchData() {
  loading.value = true
  try {
    const { data } = await listMailboxes()
    mailboxes.value = data.data
  } catch {} finally { loading.value = false }
}

async function handleCreate() {
  if (!createForm.value.prefix || !createForm.value.domain || !createForm.value.password) {
    message.warning(t('mailbox_require_fields'))
    return false
  }
  const payload = {
    email: `${createForm.value.prefix}@${createForm.value.domain}`,
    password: createForm.value.password,
    webhook_url: createForm.value.webhook_url
  }
  try {
    await createMailbox(payload)
    message.success(t('mailbox_created'))
    createForm.value = { prefix: '', domain: domainOptions.value.length ? domainOptions.value[0].value : '', password: '', webhook_url: '' }
    showCreate.value = false
    fetchData()
  } catch (e) {
    message.error(e.response?.data?.error || t('mailbox_create_fail'))
    return false
  }
}

function confirmDelete(row) {
  const emailCount = row.total_emails || 0
  dialog.warning({
    title: t('mailbox_delete_title'),
    content: `${t('mailbox_delete')} "${row.email}"? ${emailCount > 0 ? `(${emailCount} emails)` : ''}`,
    positiveText: emailCount > 0 ? t('mailbox_delete_with_emails') : t('mailbox_confirm_delete'),
    negativeText: t('common_cancel'),
    ...(emailCount > 0 ? { action: () => [
      h(NButton, { size: 'small', onClick: () => { handleDelete(row.id, false); dialog.destroyAll() } }, () => t('mailbox_only_mailbox')),
      h(NButton, { size: 'small', type: 'error', onClick: () => { handleDelete(row.id, true); dialog.destroyAll() } }, () => t('mailbox_delete_all')),
    ] } : {}),
    onPositiveClick: () => handleDelete(row.id, emailCount > 0)
  })
}

async function handleDelete(id, deleteEmails = false) {
  try {
    const { data } = await deleteMailbox(id, deleteEmails)
    const msg = deleteEmails && data.deleted_emails > 0
      ? `${t('mailbox_deleted')} + ${data.deleted_emails} emails`
      : t('mailbox_deleted')
    message.success(msg)
    fetchData()
  } catch {}
}

async function fetchDomains() {
  try {
    const { data } = await listDomains()
    domainOptions.value = data.data.map(d => ({ label: d.name, value: d.name }))
    if (domainOptions.value.length > 0 && !createForm.value.domain) {
      createForm.value.domain = domainOptions.value[0].value
    }
  } catch (e) {
    console.error('Failed to load domains', e)
  }
}

onMounted(() => {
  // 从 URL query 初始化筛选
  if (route.query.domain) filterText.value = route.query.domain
  fetchDomains()
  fetchData()
})
</script>

<style scoped>
.filter-bar {
  display: flex; gap: 10px; align-items: center; margin-bottom: 16px;
  padding: 10px 16px; background: var(--bg-card); border-radius: 10px;
  border: 1px solid var(--border-color); flex-wrap: wrap;
}
.list-count { margin-left: auto; font-size: 12px; color: var(--text-secondary); font-family: 'JetBrains Mono', monospace; }
</style>

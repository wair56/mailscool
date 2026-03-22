<template>
  <div>
    <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
      <h3 style="margin: 0">{{ t('admin_title') }}</h3>
      <n-button type="primary" @click="showCreateModal = true" v-if="isSuperAdmin">
        {{ t('admin_create') }}
      </n-button>
    </div>

    <n-data-table :columns="columns" :data="admins" :loading="loading" :bordered="false" :scroll-x="800"
      :row-class-name="() => 'table-row'" />

    <!-- 创建管理员对话框 -->
    <n-modal v-model:show="showCreateModal" preset="dialog" :title="t('admin_create_title')"
      :positive-text="t('common_create')" :negative-text="t('common_cancel')" @positive-click="handleCreate" style="width: 480px">
      <n-form :model="createForm" label-placement="left" label-width="80">
        <n-form-item :label="t('admin_username')">
          <n-input v-model:value="createForm.username" :placeholder="t('admin_username_ph')" />
        </n-form-item>
        <n-form-item :label="t('admin_password')">
          <n-input v-model:value="createForm.password" type="password" :placeholder="t('admin_password_ph')"
            show-password-on="click" />
        </n-form-item>
        <n-form-item :label="t('admin_role')">
          <n-select v-model:value="createForm.role" :options="roleOptions" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 修改密码对话框 -->
    <n-modal v-model:show="showPasswordModal" preset="dialog" :title="t('admin_change_password_title')"
      :positive-text="t('common_confirm')" :negative-text="t('common_cancel')" @positive-click="handleChangePassword" style="width: 420px">
      <n-form :model="passwordForm" label-placement="left" label-width="80">
        <n-form-item :label="t('admin_old_password')">
          <n-input v-model:value="passwordForm.old_password" type="password" :placeholder="t('admin_old_password_ph')"
            show-password-on="click" />
        </n-form-item>
        <n-form-item :label="t('admin_new_password')">
          <n-input v-model:value="passwordForm.new_password" type="password" :placeholder="t('admin_new_password_ph')"
            show-password-on="click" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 超管重置密码对话框 -->
    <n-modal v-model:show="showResetModal" preset="dialog" :title="'重置密码 - ' + (resetTarget?.username || '')"
      :positive-text="t('common_confirm')" :negative-text="t('common_cancel')" @positive-click="handleResetPassword" style="width: 420px">
      <n-form :model="resetForm" label-placement="left" label-width="80">
        <n-form-item label="新密码">
          <n-input v-model:value="resetForm.new_password" type="password" placeholder="请输入新密码"
            show-password-on="click" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 域名分配对话框 -->
    <n-modal v-model:show="showDomainModal" preset="dialog" :title="t('admin_assign_title') + ' - ' + domainTarget?.username"
      :positive-text="t('common_save')" :negative-text="t('common_cancel')" @positive-click="handleSaveDomains" style="width: 520px">
      <n-spin :show="domainLoading">
        <div v-if="allDomains.length === 0" style="color: #8888aa; text-align: center; padding: 24px">
          {{ t('admin_no_domains') }}
        </div>
        <n-checkbox-group v-else v-model:value="selectedDomainIds">
          <div class="domain-grid">
            <div v-for="d in allDomains" :key="d.id" class="domain-check-item">
              <n-checkbox :value="d.id" :label="d.name" />
            </div>
          </div>
        </n-checkbox-group>
      </n-spin>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listAdmins, createAdmin, deleteAdmin, changePassword, resetAdminPassword, listDomains, getAdminDomains, updateAdminDomains } from '../api'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()

const auth = useAuthStore()
const message = useMessage()
const dialog = useDialog()
const admins = ref([])
const loading = ref(false)
const showCreateModal = ref(false)
const showPasswordModal = ref(false)
const showResetModal = ref(false)
const resetTarget = ref(null)
const showDomainModal = ref(false)
const domainLoading = ref(false)
const domainTarget = ref(null)
const allDomains = ref([])
const selectedDomainIds = ref([])

const isSuperAdmin = computed(() => auth.admin?.role === 'super_admin')

const createForm = ref({ username: '', password: '', role: 'admin' })
const passwordForm = ref({ old_password: '', new_password: '' })
const resetForm = ref({ new_password: '' })

const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Super Admin', value: 'super_admin' },
]

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: () => t('admin_username'), key: 'username', width: 140 },
  { title: () => t('admin_role'), key: 'role', width: 120,
    render: row => h(NTag, {
      type: row.role === 'super_admin' ? 'warning' : 'info',
      bordered: false, size: 'small'
    }, () => row.role === 'super_admin' ? 'Super Admin' : 'Admin')
  },
  { title: () => t('admin_created_at'), key: 'created_at', width: 170,
    render: row => new Date(row.created_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  },
  { title: () => t('admin_domain_count'), key: 'domain_count', width: 100,
    render: row => row.role === 'super_admin' ? h(NTag, { type: 'warning', bordered: false, size: 'small' }, () => locale.value === 'zh' ? '全部' : 'All') : h(NTag, { type: 'info', bordered: false, size: 'small' }, () => `${row.domain_count || 0}`)
  },
  { title: () => t('admin_actions'), key: 'actions', width: 260,
    render: row => h(NSpace, { size: 4 }, () => {
      const buttons = []
      if (row.id === auth.admin?.id) {
        buttons.push(h(NButton, {
          size: 'small', quaternary: true, type: 'info',
          onClick: () => { showPasswordModal.value = true }
        }, () => t('admin_change_password')))
      }
      if (isSuperAdmin.value && row.role !== 'super_admin') {
        buttons.push(h(NButton, {
          size: 'small', quaternary: true, type: 'warning',
          onClick: () => { resetTarget.value = row; resetForm.value.new_password = ''; showResetModal.value = true }
        }, () => '重置密码'))
        buttons.push(h(NButton, {
          size: 'small', quaternary: true, type: 'success',
          onClick: () => handleAssignDomains(row)
        }, () => t('admin_assign_domains')))
      }
      if (isSuperAdmin.value && row.id !== auth.admin?.id) {
        buttons.push(h(NButton, {
          size: 'small', quaternary: true, type: 'error',
          onClick: () => handleDelete(row)
        }, () => t('admin_delete')))
      }
      return buttons
    })
  },
]

async function fetchData() {
  loading.value = true
  try {
    const { data } = await listAdmins()
    admins.value = data.data
  } catch {} finally { loading.value = false }
}

async function handleCreate() {
  if (!createForm.value.username || !createForm.value.password) {
    message.warning(t('mailbox_require_fields'))
    return false
  }
  try {
    await createAdmin(createForm.value)
    message.success(t('admin_created'))
    createForm.value = { username: '', password: '', role: 'admin' }
    fetchData()
  } catch (e) { message.error(e.response?.data?.error || t('admin_create_fail')); return false }
}

function handleDelete(row) {
  dialog.warning({
    title: t('admin_confirm_delete'),
    content: `${t('admin_delete')} "${row.username}"?`,
    positiveText: t('common_delete'),
    negativeText: t('common_cancel'),
    onPositiveClick: async () => {
      try {
        await deleteAdmin(row.id)
        message.success(t('admin_deleted'))
        fetchData()
      } catch (e) { message.error(e.response?.data?.error || t('common_delete')) }
    }
  })
}

async function handleAssignDomains(row) {
  domainTarget.value = row
  domainLoading.value = true
  showDomainModal.value = true

  try {
    const [domainsRes, assignedRes] = await Promise.all([
      listDomains(),
      getAdminDomains(row.id)
    ])
    allDomains.value = domainsRes.data.data || []
    selectedDomainIds.value = assignedRes.data.domain_ids || []
  } catch (e) {
    message.error(t('admin_domains_fail'))
  } finally { domainLoading.value = false }
}

async function handleSaveDomains() {
  try {
    await updateAdminDomains(domainTarget.value.id, { domain_ids: selectedDomainIds.value || [] })
    message.success(t('admin_domains_saved'))
    fetchData()
  } catch (e) {
    message.error(e.response?.data?.error || t('admin_domains_fail'))
    return false
  }
}

async function handleChangePassword() {
  if (!passwordForm.value.old_password || !passwordForm.value.new_password) {
    message.warning(t('mailbox_require_fields'))
    return false
  }
  try {
    await changePassword(passwordForm.value)
    message.success(t('admin_password_changed'))
    passwordForm.value = { old_password: '', new_password: '' }
    auth.logout()
  } catch (e) { message.error(e.response?.data?.error || t('admin_password_fail')); return false }
}

async function handleResetPassword() {
  if (!resetForm.value.new_password) {
    message.warning('请输入新密码')
    return false
  }
  try {
    await resetAdminPassword(resetTarget.value.id, resetForm.value)
    message.success(`已重置 ${resetTarget.value.username} 的密码`)
    resetForm.value = { new_password: '' }
  } catch (e) { message.error(e.response?.data?.error || '重置密码失败'); return false }
}

onMounted(fetchData)
</script>

<style scoped>
:deep(.table-row) { background: #1a1a2e; }
:deep(.table-row:hover td) { background: #252540 !important; }

.domain-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  padding: 8px 0;
}

.domain-check-item {
  padding: 8px 12px;
  background: #12121a;
  border: 1px solid #2a2a45;
  border-radius: 6px;
}
</style>

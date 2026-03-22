<template>
  <div>
    <div class="filter-bar">
      <n-input v-model:value="filterText" :placeholder="t('domain_filter_ph')" clearable
        style="width: 200px" size="small" />
      <span class="list-count">{{ filteredDomains.length }} {{ t('domain_count_label') }}</span>
      <n-button type="primary" size="small" @click="showAddModal = true">{{ t('domain_add') }}</n-button>
    </div>

    <n-data-table :columns="columns" :data="filteredDomains" :loading="loading" :bordered="false" :scroll-x="800"
      :row-class-name="() => 'table-row'" />

    <!-- 添加域名对话框 -->
    <n-modal v-model:show="showAddModal" preset="dialog" :title="t('domain_add_title')" :positive-text="t('domain_confirm')"
      :negative-text="t('domain_cancel')" @positive-click="handleAdd" style="width: 480px">
      <n-form :model="addForm" label-placement="left" label-width="60">
        <n-form-item :label="t('domain_name')">
          <n-input v-model:value="addForm.name" placeholder="example.com" />
        </n-form-item>
        <n-form-item :label="t('domain_note')">
          <n-input v-model:value="addForm.note" :placeholder="t('domain_optional')" />
        </n-form-item>
      </n-form>
      <n-alert type="info" :title="t('domain_cf_title')" style="margin-top: 12px">
        <ol style="padding-left: 16px; margin: 4px 0; line-height: 1.8">
          <li>{{ t('domain_cf_step1') }}</li>
          <li>{{ t('domain_cf_step2') }}</li>
          <li>{{ t('domain_cf_step3') }}</li>
        </ol>
      </n-alert>
    </n-modal>

    <!-- 编辑备注对话框 -->
    <n-modal v-model:show="showEditModal" preset="dialog" :title="t('domain_edit_note')" :positive-text="t('domain_save')"
      :negative-text="t('domain_cancel')" @positive-click="handleSaveNote" style="width: 420px">
      <n-input v-model:value="editNote" type="textarea" :placeholder="t('domain_input_note')"
        :autosize="{ minRows: 2, maxRows: 5 }" />
    </n-modal>

    <!-- DNS 校验结果 -->
    <n-modal v-model:show="showDNSModal" preset="dialog" :title="'DNS 校验 - ' + dnsResult.domain"
      style="width: 560px">
      <!-- 总体状态 -->
      <n-result v-if="dnsResult.status === 'pass'" status="success" :title="t('domain_config_ok')"
        :description="t('domain_config_ok_desc')" style="padding: 12px 0" />
      <n-result v-else status="warning" :title="t('domain_config_incomplete')"
        :description="t('domain_config_incomplete_desc')" style="padding: 12px 0" />

      <!-- 校验清单 -->
      <div class="check-list">
        <div v-for="check in (dnsResult.checks || [])" :key="check.name" class="check-item">
          <div class="check-header">
            <span class="check-icon">
              {{ check.status === 'pass' ? '✅' : check.status === 'fail' ? '❌' : '⚠️' }}
            </span>
            <span class="check-name">{{ check.name }}</span>
            <n-tag :type="check.status === 'pass' ? 'success' : check.status === 'fail' ? 'error' : 'warning'"
              size="small" :bordered="false">
              {{ check.status === 'pass' ? t('domain_dns_pass') : check.status === 'fail' ? t('domain_dns_fail') : t('domain_dns_optional') }}
            </n-tag>
          </div>
          <div class="check-detail">{{ check.detail }}</div>
          <div v-if="check.records?.length" class="check-records">
            <n-tag v-for="r in check.records" :key="r" size="small"
              style="margin: 2px; font-family: 'JetBrains Mono', monospace; max-width: 100%; white-space: normal; word-break: break-all; height: auto; line-height: 1.4; padding: 4px 8px" :bordered="false" type="info">
              {{ r }}
            </n-tag>
          </div>
        </div>
      </div>
    </n-modal>

    <!-- Email Config Guide -->
    <n-collapse style="margin-top: 20px">
      <n-collapse-item :title="t('domain_guide_title')" name="email-config">
        <n-alert type="info" :bordered="false" style="margin-bottom: 12px">
          <span v-html="t('domain_guide_intro')"></span>
            <br/><span style="margin-top: 4px; display: inline-block" v-html="t('domain_guide_catchall')"></span>
        </n-alert>
        <n-alert type="success" :bordered="false" style="margin-bottom: 12px">
          <span v-html="t('domain_guide_auto_title')"></span>
            <br/><span style="margin-top:4px;display:inline-block;opacity:0.8">{{ t('domain_guide_auto_hint') }}</span>
        </n-alert>
        <div class="cf-steps">
          <div class="cf-step">
            <span class="step-num">1</span>
            <div><strong>{{ t('domain_step1_title') }}</strong>
              <p>{{ t('domain_step1_desc') }} <a href="https://dash.cloudflare.com" target="_blank">Cloudflare Dashboard ↗</a> {{ t('domain_step1_desc2') }}</p>
            </div>
          </div>
          <div class="cf-step">
            <span class="step-num">2</span>
            <div><strong>{{ t('domain_step2_title') }}</strong>
              <p>{{ t('domain_step2_desc') }} <a href="https://dash.cloudflare.com/?to=/:account/:zone/email/routing/overview" target="_blank">Email Routing ↗</a> {{ t('domain_step2_desc2') }}</p>
            </div>
          </div>
          <div class="cf-step">
            <span class="step-num">3</span>
            <div><strong>{{ t('domain_step3_title') }}</strong>
              <p>{{ t('domain_step3_desc') }} <a href="https://dash.cloudflare.com/?to=/:account/workers-and-pages" target="_blank">Workers ↗</a> {{ t('domain_step3_desc2') }}</p>
              <pre class="code-block">export default {
  async email(message, env) {
    const to = message.to;
    const from = message.from;
    console.log("--- Email received / 收到邮件 ---");
    console.log("From / 发件人: " + from + " → To / 收件人: " + to);
    const raw = new Response(message.raw);
    const body = await raw.arrayBuffer();
    const resp = await fetch("https://mailer-api.mails.cool/api/receive", {
      method: "POST",
      headers: {
        "Authorization": "Bearer sk_your_api_key",
        "Content-Type": "application/octet-stream"
      },
      body: body
    });
    const text = await resp.text();
    if (resp.ok) {
      console.log("✅ Forwarded / 转发成功 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
    } else {
      console.log("⚠ Forward failed / 转发失败 | Status: " + resp.status);
      console.log("Response / 响应: " + text.substring(0, 200));
    }
  }
}</pre>
              <n-alert type="warning" :bordered="false" style="margin-top: 8px" size="small">
                ⚠️ <span v-html="t('domain_step3_warning')"></span>
              </n-alert>
            </div>
          </div>
          <div class="cf-step">
            <span class="step-num">4</span>
            <div><strong>{{ t('domain_step4_title') }}</strong>
              <p>{{ t('domain_step4_desc') }} <a href="https://dash.cloudflare.com/?to=/:account/:zone/email/routing/routes" target="_blank">Email Routing → Routes ↗</a> {{ t('domain_step4_desc2') }}</p>
            </div>
          </div>
          <div class="cf-step">
            <span class="step-num">5</span>
            <div><strong>{{ t('domain_step5_title') }}</strong>
              <p>{{ t('domain_step5_desc') }}</p>
            </div>
          </div>
        </div>
      </n-collapse-item>
    </n-collapse>

    <!-- Cloudflare 一键配置弹窗 -->
    <n-modal v-model:show="showCFModal" preset="card" :title="t('domain_cf_auto_title')" style="width: 560px">
      <n-alert type="info" :bordered="false" style="margin-bottom: 16px">
        {{ t('domain_cf_auto_desc') }}
      </n-alert>
      <n-collapse style="margin-bottom: 16px">
        <n-collapse-item :title="t('domain_cf_token_guide')">
          <div class="cf-token-guide" style="display:flex;gap:16px;align-items:flex-start">
            <div style="flex:1;min-width:0">
                <p><strong>1.</strong> {{ t('domain_cf_token_step1') }} <a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank">Cloudflare API Tokens ↗</a></p>
                <p><strong>2.</strong> <span v-html="t('domain_cf_token_step2')"></span></p>
                <p><strong>3.</strong> {{ t('domain_cf_token_step3') }}</p>
                <div class="cf-perm-list">
                  <n-tag type="success" size="small" :bordered="false">Zone → Zone → Read</n-tag>
                  <n-tag type="success" size="small" :bordered="false">Zone → Zone Settings → Edit</n-tag>
                  <n-tag type="success" size="small" :bordered="false">Zone → DNS → Edit</n-tag>
                  <n-tag type="success" size="small" :bordered="false">Zone → Email Routing Rules → Edit</n-tag>
                  <n-tag type="warning" size="small" :bordered="false">Account → Worker Scripts → Edit</n-tag>
                  <n-tag type="warning" size="small" :bordered="false">Account → Email Routing Addresses → Edit</n-tag>
                </div>
                <p><strong>4.</strong> <span v-html="t('domain_cf_token_step4')"></span></p>
                <p><strong>5.</strong> <span v-html="t('domain_cf_token_step5')"></span></p>
            </div>
            <div style="flex:0 0 280px">
              <img src="/cf-token-guide.png?v=2" :alt="t('domain_cf_token_ref')" style="width:100%;border-radius:8px;border:1px solid rgba(255,255,255,0.1)" />
            </div>
          </div>
        </n-collapse-item>
      </n-collapse>
      <n-form label-placement="left" label-width="auto">
        <n-form-item :label="t('domain_cf_domain_label')">
          <n-tag type="info" :bordered="false">{{ cfSetupDomainName }}</n-tag>
        </n-form-item>
        <n-form-item label="CF API Token">
          <n-input v-model:value="cfForm.cf_token" type="password" show-password-on="click" placeholder="Bearer token from Cloudflare" />
        </n-form-item>
        <n-form-item :label="t('domain_cf_receive_url')">
          <n-input v-model:value="cfForm.receive_url" :placeholder="receiveUrlPlaceholder" />
        </n-form-item>
      </n-form>

      <!-- Steps progress -->
      <div v-if="cfSteps.length" class="cf-setup-progress">
        <div v-for="s in cfSteps" :key="s.step" class="cf-setup-row">
          <span :class="'cf-dot ' + s.status"></span>
          <span class="cf-step-label">{{ s.step }}</span>
          <span v-if="s.error" class="cf-step-error">{{ s.error }}</span>
        </div>
      </div>

      <template #action>
        <n-button type="primary" :loading="cfLoading" :disabled="!cfForm.cf_token" @click="handleCFSetup">
          {{ t('domain_cf_start_btn') }}
        </n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, computed, onMounted } from 'vue'
import { NButton, NSwitch, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import { listDomains, createDomain, updateDomain, deleteDomain, toggleDomain, checkDomainDNS, cfSetupDomain as cfSetupDomainAPI } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()
const router = useRouter()

const receiveUrlPlaceholder = computed(() => {
  try {
    const host = window.location.host.replace(/^[^.]+\./, '')
    return `https://mailer-api.${host}/api/receive`
  } catch { return 'https://your-server.com/api/receive' }
})

const message = useMessage()
const dialog = useDialog()
const domains = ref([])
const loading = ref(false)
const showAddModal = ref(false)
const showEditModal = ref(false)
const showDNSModal = ref(false)
const showCFModal = ref(false)
const dnsResult = ref({})
const addForm = ref({ name: '', note: '' })
const editNote = ref('')
const editingDomainId = ref(null)
const cfSetupDomainId = ref(null)
const cfSetupDomainName = ref('')
const cfForm = ref({ cf_token: '', receive_url: '' })
const cfSteps = ref([])
const cfLoading = ref(false)
const dnsStatusMap = ref({}) // domainId -> { status, checks }
const filterText = ref('')

const filteredDomains = computed(() => {
  if (!filterText.value) return domains.value
  const kw = filterText.value.toLowerCase()
  return domains.value.filter(d => (d.name || '').toLowerCase().includes(kw) || (d.note || '').toLowerCase().includes(kw))
})

function dnsIndicator(status) {
  const color = status === 'pass' ? '#0aff9d' : status === 'fail' ? '#ff3d71' : status === 'warn' ? '#ffaa00' : '#2a3a50'
  const glow = status === 'pass' ? 'rgba(10,255,157,0.4)' : status === 'fail' ? 'rgba(255,61,113,0.4)' : status === 'warn' ? 'rgba(255,170,0,0.4)' : 'none'
  return { background: color, width: '8px', height: '8px', borderRadius: '50%', display: 'inline-block', boxShadow: `0 0 6px ${glow}` }
}

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: () => t('domain_name'), key: 'name', render: row => h(NTag, { bordered: false, type: 'info', style: 'cursor: pointer', onClick: () => router.push({ path: '/mailboxes', query: { domain: row.name } }) }, () => row.name) },
  { title: () => t('domain_status'), key: 'is_active', width: 90,
    render: row => {
      const ds = dnsStatusMap.value[row.id]
      const dnsOk = ds && ds.status === 'pass'
      const canToggle = dnsOk || row.is_active // 可以关闭，但开启需 DNS 通过
      return h('div', { style: 'display: flex; align-items: center; gap: 6px' }, [
        h(NSwitch, {
          value: row.is_active,
          disabled: !canToggle,
          onUpdateValue: () => handleToggle(row),
          checkedChildren: '启用',
          uncheckedChildren: '禁用'
        }),
        !dnsOk && !row.is_active ? h('span', {
          style: 'font-size: 10px; color: var(--text-secondary)',
          title: 'DNS 未通过，请先完成域名配置'
        }, '⚠️') : null,
      ].filter(Boolean))
    }
  },
  { title: 'DNS', key: 'dns_status', width: 100,
    render: row => {
      const ds = dnsStatusMap.value[row.id]
      if (!ds) return h('span', { style: 'color: var(--text-secondary); font-size: 11px' }, '...')
      const checks = ds.checks || []
      const labels = { 'MX 记录': 'MX', 'Cloudflare Email Routing': 'CF', 'SPF 记录': 'SPF', 'DMARC 记录': 'DMARC', 'SMTP 端口': 'SMTP' }
      return h('div', {
        style: 'display: flex; gap: 8px; align-items: center; cursor: pointer',
        title: checks.map(c => `${c.name}: ${c.status === "pass" ? "✓ 通过" : c.status === "fail" ? "✗ 未通过" : "⚠ 可选"}`).join('\n'),
        onClick: () => { dnsResult.value = ds; showDNSModal.value = true }
      }, checks.map(c =>
        h('span', {
          style: { ...dnsIndicator(c.status), width: '10px', height: '10px' },
          title: `${c.name}: ${c.detail}`
        })
      ))
    }
  },
  { title: 'API', key: 'total_api_keys', width: 60,
    render: row => h(NTag, { type: row.total_api_keys > 0 ? 'info' : 'default', bordered: false, size: 'small' }, () => row.total_api_keys ?? 0)
  },
  { title: () => t('mailbox_email_count'), key: 'total_mailboxes', width: 60,
    render: row => h(NTag, { type: row.total_mailboxes > 0 ? 'success' : 'default', bordered: false, size: 'small' }, () => row.total_mailboxes ?? 0)
  },
  { title: () => t('domain_email_count'), key: 'total_emails', width: 60,
    render: row => h(NTag, { type: row.total_emails > 0 ? 'success' : 'default', bordered: false, size: 'small' }, () => row.total_emails ?? 0)
  },
  { title: () => t('domain_note'), key: 'note', ellipsis: { tooltip: true },
    render: row => h('div', {
      style: 'display: flex; align-items: center; gap: 6px; cursor: pointer; min-height: 24px',
      onClick: () => openEditNote(row)
    }, [
      h('span', { style: 'flex: 1; color: ' + (row.note ? 'var(--text-primary)' : 'var(--text-secondary)') },
        row.note || '—'),
      h(NButton, { size: 'tiny', quaternary: true, type: 'info', onClick: (e) => { e.stopPropagation(); openEditNote(row) } }, () => '✏️')
    ])
  },
  { title: () => t('domain_created_at'), key: 'created_at', width: 180,
    render: row => new Date(row.created_at).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US')
  },
  { title: () => t('domain_actions'), key: 'actions', width: 160,
    render: row => h(NSpace, { size: 4 }, () => [
      h(NButton, { size: 'small', quaternary: true, type: 'info', onClick: () => openCFSetup(row) }, () => '☁️ CF'),
      h(NButton, { size: 'small', quaternary: true, type: 'error', onClick: () => handleDelete(row) }, () => t('domain_delete')),
    ])
  },
]

async function fetchData() {
  loading.value = true
  try {
    const { data } = await listDomains()
    domains.value = data.data
    // Auto-check DNS for all domains
    for (const d of data.data) {
      checkDNSStatus(d.id)
    }
  } catch {} finally { loading.value = false }
}

async function checkDNSStatus(domainId) {
  try {
    const { data } = await checkDomainDNS(domainId)
    dnsStatusMap.value = { ...dnsStatusMap.value, [domainId]: data }
  } catch {
    dnsStatusMap.value = { ...dnsStatusMap.value, [domainId]: { status: 'fail', checks: [] } }
  }
}

async function handleAdd() {
  if (!addForm.value.name) { message.warning(t('domain_name')); return false }
  try {
    await createDomain(addForm.value)
    message.success(t('domain_added'))
    addForm.value = { name: '', note: '' }
    fetchData()
  } catch (e) { message.error(e.response?.data?.error || t('domain_add_fail')); return false }
}

function openEditNote(row) {
  editingDomainId.value = row.id
  editNote.value = row.note || ''
  showEditModal.value = true
}

async function handleSaveNote() {
  try {
    await updateDomain(editingDomainId.value, { note: editNote.value })
    message.success(t('domain_saved'))
    fetchData()
  } catch (e) { message.error(t('settings_save_fail')); return false }
}

async function handleToggle(row) {
  try {
    const { data } = await toggleDomain(row.id)
    row.is_active = data.is_active
    message.success(data.is_active ? '已启用' : '已禁用')
  } catch {}
}

function handleDelete(row) {
  dialog.warning({
    title: t('domain_confirm_delete'),
    content: `${t('domain_delete')} "${row.name}"?`,
    positiveText: t('domain_delete'),
    negativeText: t('domain_cancel'),
    onPositiveClick: async () => {
      try {
        await deleteDomain(row.id)
        message.success(t('domain_deleted'))
        fetchData()
      } catch {}
    }
  })
}

function openCFSetup(row) {
  cfSetupDomainId.value = row.id
  cfSetupDomainName.value = row.name
  cfForm.value = { cf_token: '', receive_url: '' }
  cfSteps.value = []
  showCFModal.value = true
}

async function handleCFSetup() {
  cfLoading.value = true
  cfSteps.value = []
  try {
    const { data } = await cfSetupDomainAPI(cfSetupDomainId.value, cfForm.value)
    cfSteps.value = data.steps || []
    const apiKeyInfo = data.api_key || ''
    const isReused = (data.steps || []).find(s => s.step === 'create_api_key')?.error?.includes('reused')
    dialog.success({
      title: t('domain_cf_done_title'),
      content: t('domain_cf_done_desc', { domain: cfSetupDomainName.value }) + '\n\n' +
          (apiKeyInfo ? `API Key: ${apiKeyInfo}\n` : '') +
          (isReused ? t('domain_cf_done_key_reused') : t('domain_cf_done_key_new')) +
          '\n\n' + t('domain_cf_done_dns_hint'),
      positiveText: t('domain_cf_done_ok'),
    })
    fetchData()
  } catch (e) {
    const resp = e.response?.data
    cfSteps.value = resp?.steps || []
    message.error(resp?.error || 'Setup failed')
  } finally {
    cfLoading.value = false
  }
}

async function handleCheckDNS(row) {
  try {
    const { data } = await checkDomainDNS(row.id)
    dnsResult.value = data
    showDNSModal.value = true
  } catch (e) { message.error(t('domain_dns_query_fail')) }
}

onMounted(fetchData)
</script>

<style scoped>
:deep(.table-row) {
  background: var(--bg-card);
}
:deep(.table-row:hover td) {
  background: var(--bg-hover) !important;
}

.check-list {
  margin-top: 8px;
}

.check-item {
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  margin-bottom: 8px;
  background: var(--bg-primary);
  transition: all 0.3s;
}
.check-item:hover {
  border-color: rgba(0, 240, 255, 0.15);
}

.check-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.check-icon {
  font-size: 16px;
}

.check-name {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  flex: 1;
}

.check-detail {
  color: var(--text-secondary);
  font-size: 13px;
  margin-top: 4px;
  padding-left: 24px;
}

.check-records {
  margin-top: 6px;
  padding-left: 24px;
}

.cf-steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cf-step {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px 14px;
  background: rgba(0, 240, 255, 0.02);
  border-radius: 8px;
  border: 1px solid rgba(0, 240, 255, 0.06);
}

.step-num {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(0, 240, 255, 0.12);
  border: 1px solid rgba(0, 240, 255, 0.3);
  color: #00f0ff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
}

.cf-step strong { display: block; margin-bottom: 2px; font-size: 13px; }
.cf-step p { color: var(--text-secondary); font-size: 12px; margin: 0; line-height: 1.5; }
.cf-step a { color: #00f0ff; }

.code-block {
  background: #080c14;
  color: #00f0ff;
  padding: 14px;
  border-radius: 8px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 8px 0;
  border: 1px solid var(--border-color);
}

.filter-bar {
  display: flex; gap: 10px; align-items: center; margin-bottom: 16px;
  padding: 10px 16px; background: var(--bg-card); border-radius: 10px;
  border: 1px solid var(--border-color); flex-wrap: wrap;
}
.list-count { margin-left: auto; font-size: 12px; color: var(--text-secondary); font-family: 'JetBrains Mono', monospace; }

/* CF Setup Progress */
.cf-setup-progress {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cf-setup-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(0, 240, 255, 0.03);
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
}
.cf-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.cf-dot.ok { background: #0aff9d; box-shadow: 0 0 6px rgba(10,255,157,0.4); }
.cf-dot.failed { background: #ff3d71; box-shadow: 0 0 6px rgba(255,61,113,0.4); }
.cf-dot.warning { background: #ffaa00; box-shadow: 0 0 6px rgba(255,170,0,0.4); }
.cf-step-label { color: var(--text-primary); }
.cf-step-error { color: #ff3d71; font-size: 11px; margin-left: auto; max-width: 50%; text-align: right; }
.cf-token-guide p { margin: 4px 0; font-size: 13px; line-height: 1.6; }
.cf-token-guide a { color: #4dd0e1; }
.cf-perm-list { display: flex; flex-wrap: wrap; gap: 6px; margin: 8px 0; }
</style>

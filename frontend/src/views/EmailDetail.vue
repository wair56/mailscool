<template>
  <div>
    <n-page-header @back="$router.back()">
      <template #title>{{ email?.subject || t('email_loading') }}</template>
      <template #extra>
        <n-space>
          <n-button @click="handleToggleStar" :type="email?.is_starred ? 'warning' : 'default'" secondary>
            {{ email?.is_starred ? t('email_detail_starred') : t('email_detail_star') }}
          </n-button>
          <n-tag v-if="email?.code" type="success" size="large">
            {{ t('email_detail_code') }}: {{ email.code }}
          </n-tag>
        </n-space>
      </template>
    </n-page-header>

    <div v-if="email" style="margin-top: 20px">
      <n-descriptions bordered :column="2" label-placement="left"
        style="background: #1a1a2e; border-radius: 8px; margin-bottom: 16px">
        <n-descriptions-item :label="t('email_detail_from')">{{ email.sender }}</n-descriptions-item>
        <n-descriptions-item :label="t('email_detail_to')">{{ email.recipient }}</n-descriptions-item>
        <n-descriptions-item :label="t('email_detail_time')">{{ new Date(email.received_at).toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US') }}</n-descriptions-item>
        <n-descriptions-item :label="t('email_detail_size')">{{ formatSize(email.raw_size) }}</n-descriptions-item>
      </n-descriptions>

      <div v-if="parsedLinks.length" style="margin-bottom: 16px">
        <h4 style="margin-bottom: 8px; color: #8888aa">{{ t('email_detail_links') }}</h4>
        <n-space vertical>
          <n-tag v-for="link in parsedLinks" :key="link" type="info" :bordered="false" size="small">
            <a :href="link" target="_blank" style="color: inherit; text-decoration: none">{{ link }}</a>
          </n-tag>
        </n-space>
      </div>

      <n-tabs type="line" animated :default-value="email.body_html ? 'html' : 'text'">
        <n-tab-pane name="text" :tab="t('email_detail_text')">
          <pre class="email-body-text">{{ email.body_text || t('email_detail_no_text') }}</pre>
        </n-tab-pane>
        <n-tab-pane name="html" :tab="t('email_detail_html')" v-if="email.body_html">
          <div class="email-body-html" v-html="email.body_html" />
        </n-tab-pane>
      </n-tabs>
    </div>
    <n-spin v-else style="display: flex; justify-content: center; padding: 60px" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getEmail, toggleEmailStar } from '../api'
import { useI18n } from '../i18n'

const { t, locale } = useI18n()
const route = useRoute()
const message = useMessage()
const email = ref(null)

const parsedLinks = computed(() => {
  if (!email.value?.links) return []
  try { return JSON.parse(email.value.links) } catch { return [] }
})

function formatSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

onMounted(async () => {
  try {
    const { data } = await getEmail(route.params.id)
    email.value = data
  } catch {}
})

async function handleToggleStar() {
  try {
    const { data } = await toggleEmailStar(email.value.id)
    email.value.is_starred = data.is_starred
    message.success(data.is_starred ? t('email_detail_star_on') : t('email_detail_star_off'))
  } catch {}
}
</script>

<style scoped>
.email-body-html {
  background: #fff;
  color: #333;
  padding: 24px;
  border-radius: 8px;
  min-height: 400px;
  max-height: calc(100vh - 280px);
  overflow-y: auto;
}

.email-body-text {
  background: #1a1a2e;
  color: #e8e8f0;
  padding: 20px;
  border-radius: 8px;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  min-height: 400px;
  max-height: calc(100vh - 280px);
  overflow-y: auto;
}
</style>

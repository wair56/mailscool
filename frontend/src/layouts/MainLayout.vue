<template>
  <n-layout has-sider style="min-height: 100vh">
    <n-layout-sider
      bordered
      :width="240"
      :collapsed-width="64"
      collapse-mode="width"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
      :native-scrollbar="false"
      style="background: #0d1520; position: sticky; top: 0; height: 100vh; z-index: 20; overflow: hidden"
    >
      <div class="logo" :class="{ collapsed }">
        <span class="logo-mark">[<span class="logo-letter">M</span>]</span>
        <span v-if="!collapsed" class="logo-text">MAILS<span class="logo-accent">COOL</span></span>
      </div>
      <div style="display: flex; flex-direction: column; height: calc(100vh - 57px)">
        <n-menu
          :collapsed="collapsed"
          :collapsed-width="64"
          :collapsed-icon-size="22"
          :options="topMenuOptions"
          :value="activeKey"
          @update:value="handleMenuClick"
          :indent="24"
        />
        <div style="margin-top: auto">
          <n-menu
            :collapsed="collapsed"
            :collapsed-width="64"
            :collapsed-icon-size="22"
            :options="bottomMenuOptions"
            :value="activeKey"
            @update:value="handleMenuClick"
            :indent="24"
          />
        </div>
      </div>
    </n-layout-sider>
    <n-layout content-style="overflow-y: auto; height: 100vh">
      <n-layout-header bordered class="top-bar" style="position: sticky; top: 0; z-index: 10">
        <span class="page-title">{{ pageTitle }}</span>
        <n-space align="center">
          <n-dropdown :options="localeOptions" @select="setLocale" trigger="click" :value="locale">
            <n-button quaternary size="small" class="lang-btn">
              🌐 {{ currentLocaleName }}
            </n-button>
          </n-dropdown>
          <n-tag :bordered="false" size="small" :style="roleStyle">
            {{ auth.admin?.role === 'super_admin' ? 'SUPER ADMIN' : 'ADMIN' }}
          </n-tag>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary>
              {{ auth.admin?.username }}
              <template #icon><n-icon><person-outline /></n-icon></template>
            </n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <n-layout-content style="padding: 24px; background: #080c14; min-height: calc(100vh - 56px)">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup>
import { ref, computed, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NIcon } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { useI18n } from '../i18n'
import {
  HomeOutline,
  GlobeOutline,
  MailOutline,
  MailOpenOutline,
  KeyOutline,
  ListOutline,
  PersonOutline,
  PeopleOutline,
  SettingsOutline,
  CodeSlashOutline
} from '@vicons/ionicons5'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const collapsed = ref(false)
const { t, locale, setLocale, availableLocales } = useI18n()

const localeOptions = availableLocales.map(l => ({ label: l.label, key: l.code }))
const currentLocaleName = computed(() => availableLocales.find(l => l.code === locale.value)?.label || locale.value)

function renderIcon(icon) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const isSuperAdmin = computed(() => auth.admin?.role === 'super_admin')

const topMenuOptions = computed(() => [
  { label: t('nav_dashboard'), key: 'Dashboard', icon: renderIcon(HomeOutline) },
  { label: t('nav_domains'), key: 'Domains', icon: renderIcon(GlobeOutline) },
  { label: t('nav_mailboxes'), key: 'Mailboxes', icon: renderIcon(MailOpenOutline) },
  { label: t('nav_emails'), key: 'Emails', icon: renderIcon(MailOutline) },
])

const bottomMenuOptions = computed(() => {
  const items = [
    { label: t('nav_api_docs'), key: 'ApiDocs', icon: renderIcon(CodeSlashOutline) },
    { label: t('nav_api_keys'), key: 'ApiKeys', icon: renderIcon(KeyOutline) },
  ]
  if (isSuperAdmin.value) {
    items.push({ label: t('nav_admins'), key: 'Admins', icon: renderIcon(PeopleOutline) })
  }
  items.push({ label: t('nav_audit_logs'), key: 'AuditLogs', icon: renderIcon(ListOutline) })
  items.push({ label: t('nav_settings'), key: 'Settings', icon: renderIcon(SettingsOutline) })
  return items
})

const activeKey = computed(() => route.name)

const pageTitles = {
  Dashboard: 'page_dashboard',
  Domains: 'page_domains',
  Emails: 'page_emails',
  EmailDetail: 'page_email_detail',
  Mailboxes: 'page_mailboxes',
  ApiKeys: 'page_api_keys',
  Admins: 'page_admins',
  AuditLogs: 'page_audit_logs',
  ApiDocs: 'page_api_docs',
  Settings: 'page_settings',
}
const pageTitle = computed(() => t(pageTitles[route.name] || ''))

const roleStyle = computed(() => ({
  background: auth.admin?.role === 'super_admin' ? 'rgba(0, 240, 255, 0.08)' : 'rgba(10, 255, 157, 0.08)',
  color: auth.admin?.role === 'super_admin' ? '#00f0ff' : '#0aff9d',
  border: `1px solid ${auth.admin?.role === 'super_admin' ? 'rgba(0, 240, 255, 0.2)' : 'rgba(10, 255, 157, 0.2)'}`,
}))

const userMenuOptions = computed(() => [
  { label: t('user_change_password'), key: 'password' },
  { label: t('user_logout'), key: 'logout' },
])

function handleMenuClick(key) {
  if (key === 'ApiDocs') {
    const routeUrl = router.resolve({ name: 'ApiDocs' })
    window.open(routeUrl.href, '_blank')
    return
  }
  router.push({ name: key })
}

function handleUserMenu(key) {
  if (key === 'logout') {
    auth.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 10px;
  border-bottom: 1px solid #1a2940;
  transition: all 0.3s;
}

.logo.collapsed {
  padding: 0;
  justify-content: center;
}

.logo-mark {
  font-family: 'JetBrains Mono', monospace;
  font-size: 20px;
  font-weight: 700;
  color: #5e7290;
  flex-shrink: 0;
}

.logo-letter {
  color: #00f0ff;
  text-shadow: 0 0 8px rgba(0, 240, 255, 0.4);
  animation: glowPulse 3s ease-in-out infinite;
}

.logo-text {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  font-weight: 700;
  color: #c8d6e5;
  letter-spacing: 3px;
  white-space: nowrap;
}

.logo-accent {
  color: #00f0ff;
}

.top-bar {
  height: 56px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #0d1520;
  border-bottom: 1px solid #1a2940;
  box-shadow: 0 1px 0 0 rgba(0, 240, 255, 0.05);
}

.page-title {
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  font-weight: 500;
  letter-spacing: 1px;
}

@keyframes glowPulse {
  0%, 100% { opacity: 0.7; text-shadow: 0 0 6px rgba(0, 240, 255, 0.3); }
  50% { opacity: 1; text-shadow: 0 0 14px rgba(0, 240, 255, 0.6); }
}

/* 侧栏折叠按钮鼠标靠近才显示 */
:deep(.n-layout-sider__border) {
  transition: opacity 0.3s;
}
:deep(.n-layout-toggle-button) {
  opacity: 0;
  transition: opacity 0.3s;
}
:deep(.n-layout-sider:hover .n-layout-toggle-button) {
  opacity: 1;
}

.lang-btn {
  font-family: 'JetBrains Mono', monospace !important;
  font-size: 12px !important;
  color: #00f0ff !important;
  letter-spacing: 0.5px;
}
</style>

import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { public: true }
  },
  {
    path: '/inbox',
    name: 'MailboxInbox',
    component: () => import('../views/MailboxInbox.vue'),
    meta: { public: true }
  },
  {
    path: '/api-docs',
    name: 'ApiDocs',
    component: () => import('../views/ApiDocs.vue'),
    meta: { public: true }
  },
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'domains', name: 'Domains', component: () => import('../views/Domains.vue') },
      { path: 'emails', name: 'Emails', component: () => import('../views/Emails.vue') },
      { path: 'emails/:id', name: 'EmailDetail', component: () => import('../views/EmailDetail.vue') },
      { path: 'mailboxes', name: 'Mailboxes', component: () => import('../views/Mailboxes.vue') },
      { path: 'api-keys', name: 'ApiKeys', component: () => import('../views/ApiKeys.vue') },
      { path: 'admins', name: 'Admins', component: () => import('../views/Admins.vue') },
      { path: 'audit-logs', name: 'AuditLogs', component: () => import('../views/AuditLogs.vue') },
      { path: 'settings', name: 'Settings', component: () => import('../views/Settings.vue') },
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.token) {
    return { name: 'Login' }
  }
})

export default router

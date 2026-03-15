import axios from 'axios'
import { useAuthStore } from '../stores/auth'
import router from '../router'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '',
  timeout: 15000
})

// 请求拦截器：自动添加 JWT token
api.interceptors.request.use(config => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

// 响应拦截器：401 跳转登录
api.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      const auth = useAuthStore()
      auth.logout()
      router.push('/login')
    }
    return Promise.reject(error)
  }
)

export default api

// === Auth ===
export const login = (data) => api.post('/admin/login', data)
export const getMe = () => api.get('/admin/me')
export const changePassword = (data) => api.put('/admin/password', data)

// === Dashboard ===
export const getDashboard = () => api.get('/admin/dashboard')

// === Domains ===
export const listDomains = () => api.get('/admin/domains')
export const createDomain = (data) => api.post('/admin/domains', data)
export const updateDomain = (id, data) => api.put(`/admin/domains/${id}`, data)
export const deleteDomain = (id) => api.delete(`/admin/domains/${id}`)
export const toggleDomain = (id) => api.put(`/admin/domains/${id}/toggle`)
export const checkDomainDNS = (id) => api.post(`/admin/domains/${id}/check-dns`)
export const cfSetupDomain = (id, data) => api.post(`/admin/domains/${id}/cf-setup`, data, { timeout: 30000 })
export const getDomainStats = (id) => api.get(`/admin/domains/${id}/stats`)

// === Emails ===
export const listEmails = (params) => api.get('/admin/emails', { params })
export const getEmail = (id) => api.get(`/admin/emails/${id}`)

// === API Keys ===
export const listApiKeys = () => api.get('/admin/api-keys')
export const createApiKey = (data) => api.post('/admin/api-keys', data)
export const toggleApiKey = (id) => api.put(`/admin/api-keys/${id}/toggle`)
export const updateApiKey = (id, data) => api.put(`/admin/api-keys/${id}`, data)
export const deleteApiKey = (id) => api.delete(`/admin/api-keys/${id}`)

// === Admins ===
export const listAdmins = () => api.get('/admin/admins')
export const createAdmin = (data) => api.post('/admin/admins', data)
export const deleteAdmin = (id) => api.delete(`/admin/admins/${id}`)
export const getAdminDomains = (id) => api.get(`/admin/admins/${id}/domains`)
export const updateAdminDomains = (id, data) => api.put(`/admin/admins/${id}/domains`, data)

// === Audit Logs ===
export const listAuditLogs = (params) => api.get('/admin/audit-logs', { params })

// === System Settings ===
export const getSystemSettings = () => api.get('/admin/settings')
export const updateSystemSettings = (data) => api.put('/admin/settings', data)
export const getSystemStatus = () => api.get('/admin/system-status')
export const getSystemLogs = (lines = 100) => api.get('/admin/system-logs', { params: { lines } })

// === Emails Star ===
export const toggleEmailStar = (id) => api.put(`/admin/emails/${id}/star`)

// === Mailboxes ===
export const listMailboxes = () => api.get('/admin/mailboxes')
export const createMailbox = (data) => api.post('/admin/mailboxes', data)
export const updateMailbox = (id, data) => api.put(`/admin/mailboxes/${id}`, data)
export const deleteMailbox = (id, deleteEmails = false) => api.delete(`/admin/mailboxes/${id}${deleteEmails ? '?delete_emails=true' : ''}`)

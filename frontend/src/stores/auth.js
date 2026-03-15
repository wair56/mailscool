import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const admin = ref(JSON.parse(localStorage.getItem('admin') || 'null'))

  function setAuth(tokenVal, adminVal) {
    token.value = tokenVal
    admin.value = adminVal
    localStorage.setItem('token', tokenVal)
    localStorage.setItem('admin', JSON.stringify(adminVal))
  }

  function logout() {
    token.value = ''
    admin.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('admin')
  }

  return { token, admin, setAuth, logout }
})

import { ref, computed } from 'vue'
import zhMessages from './zh.json'
import enMessages from './en.json'
import jaMessages from './ja.json'
import koMessages from './ko.json'
import arMessages from './ar.json'
import frMessages from './fr.json'
import esMessages from './es.json'

const messages = { zh: zhMessages, en: enMessages, ja: jaMessages, ko: koMessages, ar: arMessages, fr: frMessages, es: esMessages }
const locale = ref(localStorage.getItem('locale') || 'zh')

export const availableLocales = [
  { code: 'zh', label: '中文' },
  { code: 'en', label: 'EN' },
  { code: 'ja', label: '日本語' },
  { code: 'ko', label: '한국어' },
  { code: 'ar', label: 'عربي' },
  { code: 'fr', label: 'FR' },
  { code: 'es', label: 'ES' }
]

export function useI18n() {
  const t = (key, params) => {
    let text = messages[locale.value]?.[key] || messages['en']?.[key] || key
    if (params) {
      Object.keys(params).forEach(k => {
        text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), params[k])
      })
    }
    return text
  }
  const setLocale = (l) => {
    locale.value = l
    localStorage.setItem('locale', l)
    // RTL 支持
    document.documentElement.dir = l === 'ar' ? 'rtl' : 'ltr'
    document.documentElement.lang = l
    // 刷新页面确保 NaiveUI locale 和 RTL 完全生效
    window.location.reload()
  }
  const isRTL = computed(() => locale.value === 'ar')
  return { t, locale: computed(() => locale.value), setLocale, isRTL, availableLocales }
}

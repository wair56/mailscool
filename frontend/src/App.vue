<template>
  <n-config-provider :theme="darkTheme" :locale="naiveLocale" :date-locale="naiveDateLocale">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import {
  darkTheme,
  zhCN, dateZhCN,
  enUS, dateEnUS,
  jaJP, dateJaJP,
  koKR, dateKoKR,
  arDZ, dateArDZ,
  frFR, dateFrFR,
  esAR, dateEsAR
} from 'naive-ui'
import { useI18n } from './i18n'

const { locale } = useI18n()

const localeMap = {
  zh: { locale: zhCN, dateLocale: dateZhCN },
  en: { locale: enUS, dateLocale: dateEnUS },
  ja: { locale: jaJP, dateLocale: dateJaJP },
  ko: { locale: koKR, dateLocale: dateKoKR },
  ar: { locale: arDZ, dateLocale: dateArDZ },
  fr: { locale: frFR, dateLocale: dateFrFR },
  es: { locale: esAR, dateLocale: dateEsAR },
}

const naiveLocale = computed(() => localeMap[locale.value]?.locale || enUS)
const naiveDateLocale = computed(() => localeMap[locale.value]?.dateLocale || dateEnUS)
</script>

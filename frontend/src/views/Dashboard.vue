<template>
  <div class="dashboard">
    <n-grid cols="2 s:2 m:3 l:4" responsive="screen" :x-gap="16" :y-gap="16">
      <n-gi>
        <div class="stat-card stagger-1">
          <div class="stat-glow cyan"></div>
          <div class="stat-label">TOTAL EMAILS</div>
          <div class="stat-value cyan">{{ animatedTotal }}</div>
          <div class="stat-sub">{{ t('stat_total_emails') }}</div>
        </div>
      </n-gi>
      <n-gi>
        <div class="stat-card stagger-2">
          <div class="stat-glow green"></div>
          <div class="stat-label">TODAY</div>
          <div class="stat-value green">{{ animatedToday }}</div>
          <div class="stat-sub">{{ t('stat_today') }}</div>
        </div>
      </n-gi>
      <n-gi>
        <div class="stat-card stagger-3">
          <div class="stat-glow yellow"></div>
          <div class="stat-label">DOMAINS</div>
          <div class="stat-value yellow">{{ stats.active_domains ?? 0 }}<span class="stat-total">/{{ stats.total_domains ?? 0 }}</span></div>
          <div class="stat-sub">{{ t('stat_active_domains') }}</div>
        </div>
      </n-gi>
      <n-gi>
        <div class="stat-card stagger-4">
          <div class="stat-glow red"></div>
          <div class="stat-label">API KEYS</div>
          <div class="stat-value red">{{ stats.total_api_keys ?? 0 }}</div>
          <div class="stat-sub">{{ t('stat_api_keys') }}</div>
        </div>
      </n-gi>
    </n-grid>

    <div class="chart-section stagger-5">
      <div class="chart-header">
        <span class="chart-dot"></span>
        <h3>INBOUND TRAFFIC — 7 DAYS</h3>
      </div>
      <div class="chart-container">
        <div v-for="(item, i) in chartData" :key="item.date" class="chart-col">
          <div class="chart-bar-track">
            <div class="chart-bar" :style="{ height: item.height + '%', animationDelay: (i * 80) + 'ms' }">
              <span class="chart-value">{{ item.count }}</span>
            </div>
          </div>
          <span class="chart-label">{{ item.label }}</span>
        </div>
      </div>
    </div>

    <!-- Recent panels -->
    <n-grid cols="1 m:2 l:3" responsive="screen" :x-gap="16" :y-gap="16" style="margin-top: 24px">
      <n-gi>
        <div class="panel-card">
          <div class="panel-header">
            <span class="chart-dot"></span>
            <h3>RECENT MAILBOXES</h3>
          </div>
          <div v-if="recentMailboxes.length === 0" class="panel-empty">-</div>
          <div v-else class="panel-list">
            <div v-for="mb in recentMailboxes" :key="mb.id" class="panel-item">
              <div class="panel-item-title">{{ mb.email }}</div>
              <div class="panel-item-sub">
                <span>{{ mb.domain_name }}</span>
                <span class="panel-item-time">{{ formatTime(mb.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </n-gi>
      <n-gi>
        <div class="panel-card">
          <div class="panel-header">
            <span class="chart-dot" style="background: #0aff9d; box-shadow: 0 0 8px rgba(10,255,157,0.5)"></span>
            <h3>LATEST EMAILS</h3>
          </div>
          <div v-if="latestEmails.length === 0" class="panel-empty">-</div>
          <div v-else class="panel-list">
            <div v-for="em in latestEmails" :key="em.id" class="panel-item">
              <div class="panel-item-title">{{ em.subject || t('email_no_subject') }}</div>
              <div class="panel-item-sub">
                <span>{{ em.recipient }}</span>
                <span class="panel-item-time">{{ formatTime(em.received_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </n-gi>
      <n-gi>
        <div class="panel-card">
          <div class="panel-header">
            <span class="chart-dot" style="background: #ff3d71; box-shadow: 0 0 8px rgba(255,61,113,0.5)"></span>
            <h3>RECENT API KEYS</h3>
          </div>
          <div v-if="recentApiKeys.length === 0" class="panel-empty">-</div>
          <div v-else class="panel-list">
            <div v-for="ak in recentApiKeys" :key="ak.id" class="panel-item">
              <div class="panel-item-title">{{ ak.name || ak.key_prefix }}</div>
              <div class="panel-item-sub">
                <span>限速: {{ ak.rate_limit }}/min</span>
                <n-tag :type="ak.is_active ? 'success' : 'error'" :bordered="false" size="small">
                  {{ ak.is_active ? 'ON' : 'OFF' }}
                </n-tag>
              </div>
            </div>
          </div>
        </div>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { NTag } from 'naive-ui'
import { getDashboard, listMailboxes, listEmails, listApiKeys } from '../api'
import { useI18n } from '../i18n'

const { t } = useI18n()

const stats = ref({})
const animatedTotal = ref(0)
const animatedToday = ref(0)
const recentMailboxes = ref([])
const latestEmails = ref([])
const recentApiKeys = ref([])

function animateNumber(target, setter, duration = 800) {
  const start = 0
  const startTime = performance.now()
  function tick(now) {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3) // easeOutCubic
    setter(Math.round(start + (target - start) * eased))
    if (progress < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

function formatTime(t) {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const chartData = computed(() => {
  const daily = stats.value.daily_stats || []
  const maxCount = Math.max(...daily.map(d => d.count), 1)
  return daily.map(d => ({
    date: d.date,
    count: d.count,
    height: Math.max((d.count / maxCount) * 100, 6),
    label: d.date?.slice(5) || ''
  }))
})

onMounted(async () => {
  try {
    const [dashRes, mbRes, emRes, akRes] = await Promise.all([
      getDashboard(),
      listMailboxes().catch(() => ({ data: { data: [] } })),
      listEmails({ page: 1, size: 5 }).catch(() => ({ data: { data: [] } })),
      listApiKeys().catch(() => ({ data: [] }))
    ])
    stats.value = dashRes.data
    animateNumber(dashRes.data.total_emails || 0, v => animatedTotal.value = v)
    animateNumber(dashRes.data.today_emails || 0, v => animatedToday.value = v, 600)

    recentMailboxes.value = (mbRes.data.data || []).slice(0, 5)
    latestEmails.value = (emRes.data.data || []).slice(0, 5)
    recentApiKeys.value = (Array.isArray(akRes.data) ? akRes.data : (akRes.data.data || [])).slice(0, 5)
  } catch {}
})
</script>

<style scoped>
.dashboard {
  max-width: 1200px;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
  position: relative;
  overflow: hidden;
  transition: all 0.3s ease;
}
.stat-card:hover {
  transform: translateY(-3px);
  border-color: rgba(0, 240, 255, 0.2);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.3);
}

/* Top glow bar */
.stat-glow {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  border-radius: 12px 12px 0 0;
}
.stat-glow.cyan { background: linear-gradient(90deg, transparent, #00f0ff, transparent); box-shadow: 0 0 12px rgba(0, 240, 255, 0.3); }
.stat-glow.green { background: linear-gradient(90deg, transparent, #0aff9d, transparent); box-shadow: 0 0 12px rgba(10, 255, 157, 0.3); }
.stat-glow.yellow { background: linear-gradient(90deg, transparent, #ffaa00, transparent); box-shadow: 0 0 12px rgba(255, 170, 0, 0.3); }
.stat-glow.red { background: linear-gradient(90deg, transparent, #ff3d71, transparent); box-shadow: 0 0 12px rgba(255, 61, 113, 0.3); }

.stat-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 2px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.stat-value {
  font-family: 'JetBrains Mono', monospace;
  font-size: 32px;
  font-weight: 700;
  animation: countUp 0.5s ease both;
}
.stat-value.cyan { color: #00f0ff; text-shadow: 0 0 10px rgba(0, 240, 255, 0.2); }
.stat-value.green { color: #0aff9d; text-shadow: 0 0 10px rgba(10, 255, 157, 0.2); }
.stat-value.yellow { color: #ffaa00; text-shadow: 0 0 10px rgba(255, 170, 0, 0.2); }
.stat-value.red { color: #ff3d71; text-shadow: 0 0 10px rgba(255, 61, 113, 0.2); }

.stat-total {
  font-size: 16px;
  color: var(--text-secondary);
  font-weight: 400;
}

.stat-sub {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}

/* Chart */
.chart-section {
  margin-top: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
}

.chart-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
}
.chart-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #00f0ff;
  box-shadow: 0 0 8px rgba(0, 240, 255, 0.5);
}
.chart-header h3 {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 1.5px;
  color: var(--text-primary);
}

.chart-container {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  height: 200px;
}
.chart-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}
.chart-bar-track {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

@keyframes barGrow {
  from { height: 0%; opacity: 0; }
  to { opacity: 1; }
}

.chart-bar {
  width: 100%;
  max-width: 56px;
  background: linear-gradient(180deg, #00f0ff, #0aff9d);
  border-radius: 6px 6px 2px 2px;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding-top: 6px;
  animation: barGrow 0.6s ease both;
  position: relative;
  box-shadow: 0 0 8px rgba(0, 240, 255, 0.15);
}

.chart-value {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: #080c14;
  font-weight: 700;
}

.chart-label {
  margin-top: 8px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-secondary);
}

/* Recent panels */
.panel-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px;
  min-height: 200px;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.panel-header h3 {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 1.5px;
  color: var(--text-primary);
  margin: 0;
}

.panel-empty {
  color: var(--text-secondary);
  font-size: 13px;
  text-align: center;
  padding: 32px 0;
}

.panel-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.panel-item {
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.2s ease;
}

.panel-item:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(0, 240, 255, 0.15);
}

.panel-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-item-sub {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: var(--text-secondary);
}

.panel-item-time {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.7;
}
</style>

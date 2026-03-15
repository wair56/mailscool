// MailsCool Chrome Extension - Popup Logic v1.0.0
// Flow: Create - Auto-fill email - Poll for code - Auto-fill code
// I18N: Uses chrome.i18n API with _locales/

const t = (key) => chrome.i18n.getMessage(key) || key

document.addEventListener('DOMContentLoaded', () => {
  // Apply i18n to all elements with data-i18n attribute
  document.querySelectorAll('[data-i18n]').forEach(el => {
    const msg = t(el.getAttribute('data-i18n'))
    if (msg) el.textContent = msg
  })

  ;(async () => {
    const $ = id => document.getElementById(id)
    const DEFAULT_SERVER = 'https://mails.cool'

    // ===== Global state ===== //
    let pollTimer = null
    let pollCount = 0

    // Load saved settings
    const stored = await chrome.storage.local.get(['apiUrl', 'apiKey', 'currentEmail', 'currentPass'])
    if (stored.apiUrl && stored.apiUrl !== DEFAULT_SERVER) {
      $('apiUrl').value = stored.apiUrl
      $('advancedPanel').classList.add('show')
      $('toggleArrow').classList.add('open')
    }
    if (stored.apiKey) {
      $('apiKey').value = stored.apiKey
      $('advancedPanel').classList.add('show')
      $('toggleArrow').classList.add('open')
    }

    // Restore previous mailbox
    if (stored.currentEmail) {
      showResult(stored.currentEmail, stored.currentPass || '')
      startPolling()
    }

    // Toggle advanced panel
    $('toggleAdvanced').addEventListener('click', () => {
      $('advancedPanel').classList.toggle('show')
      $('toggleArrow').classList.toggle('open')
    })

    // Save settings on change
    $('apiUrl').addEventListener('change', () => saveSettings())
    $('apiKey').addEventListener('change', () => saveSettings())

    function saveSettings() {
      chrome.storage.local.set({
        apiUrl: $('apiUrl').value.replace(/\/$/, ''),
        apiKey: $('apiKey').value
      })
    }

    function getApiUrl() {
      return ($('apiUrl').value.replace(/\/$/, '')) || DEFAULT_SERVER
    }

    // ===== Create temp mailbox =====
    $('createBtn').addEventListener('click', async () => {
      saveSettings()
      $('createBtn').disabled = true
      setStatus(t('creating'), 'polling')

      const apiUrl = getApiUrl()
      const apiKey = $('apiKey').value

      try {
        let resp
        if (apiKey) {
          resp = await fetch(apiUrl + '/api/mailboxes', {
            method: 'POST',
            headers: { 'Authorization': 'Bearer ' + apiKey, 'Content-Type': 'application/json' }
          })
        } else {
          resp = await fetch(apiUrl + '/mailbox/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
          })
        }
        if (!resp.ok) {
          const err = await resp.json().catch(() => ({}))
          throw new Error(err.error || 'HTTP ' + resp.status)
        }
        const data = await resp.json()

        // Save to storage
        await chrome.storage.local.set({
          currentEmail: data.email,
          currentPass: data.password,
          mailboxToken: data.token || ''
        })

        showResult(data.email, data.password)

        // AUTO-FILL: immediately fill email into the active page
        try {
          const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
          if (tab) {
            chrome.tabs.sendMessage(tab.id, { action: 'fill_email', email: data.email }).catch(() => {})
          }
        } catch (ignored) {}

        setStatus(t('autoFilled'), 'polling')
        startPolling()
      } catch (e) {
        setStatus('Error: ' + e.message, 'error')
      } finally {
        $('createBtn').disabled = false
      }
    })

    function showResult(email, pass) {
      const eEl = $('resultEmail')
      const pEl = $('resultPass')

      if (eEl) eEl.textContent = email
      if (pEl) pEl.textContent = pass

      const resEl = $('result')
      if (resEl) resEl.classList.add('show')

      const fillEl = $('fillBtn')
      if (fillEl) fillEl.style.display = 'block'

      // Set href for the <a> tag
      const apiUrl = getApiUrl()
      const url = apiUrl + '/login?email=' + encodeURIComponent(email) + '&password=' + encodeURIComponent(pass)
      const inboxEl = $('openInboxBtn')
      if (inboxEl) inboxEl.href = url
    }

    // Copy buttons
    $('copyEmail').addEventListener('click', () => {
      navigator.clipboard.writeText($('resultEmail').textContent)
      $('copyEmail').textContent = t('copied')
      setTimeout(() => { $('copyEmail').textContent = t('copy') }, 1500)
    })
    $('copyPass').addEventListener('click', () => {
      navigator.clipboard.writeText($('resultPass').textContent)
      $('copyPass').textContent = t('copied')
      setTimeout(() => { $('copyPass').textContent = t('copy') }, 1500)
    })

    // Manual fill email button
    $('fillBtn').addEventListener('click', async () => {
      const email = $('resultEmail').textContent
      try {
        const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
        if (tab) {
          chrome.tabs.sendMessage(tab.id, { action: 'fill_email', email })
          setStatus(t('emailFilled'), '')
        }
      } catch (ignored) {}
    })

    // Quick Login
    $('openInboxBtn').addEventListener('click', () => {
      try {
        const email = $('resultEmail').textContent
        const pass = $('resultPass').textContent
        const apiUrl = getApiUrl()
        if (email && pass) {
          $('openInboxBtn').href = apiUrl + '/login?email=' + encodeURIComponent(email) + '&password=' + encodeURIComponent(pass)
        } else {
          $('openInboxBtn').href = apiUrl + '/login'
        }
      } catch (err) {}
    })

    // ===== Poll for verification codes =====
    function startPolling() {
      if (pollTimer) clearInterval(pollTimer)
      pollCount = 0
      pollTimer = setInterval(pollForCode, 4000)
      pollForCode()
    }

    async function pollForCode() {
      pollCount++
      const { currentEmail } = await chrome.storage.local.get(['currentEmail'])
      const apiUrl = getApiUrl()
      const apiKey = $('apiKey').value
      if (!currentEmail) return

      const dots = '.'.repeat((pollCount % 3) + 1)
      setStatus(t('watching') + dots, 'polling')

      try {
        let emails = []

        if (apiKey) {
          const resp = await fetch(
            apiUrl + '/api/emails?to=' + encodeURIComponent(currentEmail) + '&page=1&size=3',
            { headers: { 'Authorization': 'Bearer ' + apiKey } }
          )
          if (resp.ok) {
            const data = await resp.json()
            emails = data.data || []
          }
        } else {
          const { currentPass } = await chrome.storage.local.get(['currentPass'])
          if (!currentPass) return

          let token = (await chrome.storage.local.get(['mailboxToken'])).mailboxToken
          if (!token) {
            const loginResp = await fetch(apiUrl + '/mailbox/login', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ email: currentEmail, password: currentPass })
            })
            if (loginResp.ok) {
              const loginData = await loginResp.json()
              token = loginData.token
              await chrome.storage.local.set({ mailboxToken: token })
            }
          }
          if (!token) return

          const resp = await fetch(apiUrl + '/mailbox/emails?page=1&size=3', {
            headers: { 'Authorization': 'Bearer ' + token }
          })
          if (resp.ok) {
            const data = await resp.json()
            emails = data.data || []
          }
        }

        for (const email of emails) {
          if (email.id) {
            $('codeSection').classList.add('show')
            $('codeSubject').textContent = email.subject || 'No Subject'

            if (email.code) {
              $('codeValue').textContent = email.code
              $('codeValue').style.fontSize = '26px'
              navigator.clipboard.writeText(email.code).catch(() => {})

              chrome.action.setBadgeText({ text: email.code }).catch(() => {})
              chrome.action.setBadgeBackgroundColor({ color: '#0aff9d' }).catch(() => {})

              try {
                const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
                if (tab) {
                  chrome.tabs.sendMessage(tab.id, { action: 'fill_code', code: email.code }).catch(() => {})
                }
              } catch (ignored) {}

              setStatus(t('codeFound'), '')
            } else {
              $('codeValue').textContent = t('newMailBadge')
              $('codeValue').style.fontSize = '18px'

              chrome.action.setBadgeText({ text: '1' }).catch(() => {})
              chrome.action.setBadgeBackgroundColor({ color: '#00f0ff' }).catch(() => {})

              setStatus(t('newMail'), '')
            }

            if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
            return
          }
        }

        if (pollCount > 75) {
          if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
          setStatus(t('pollTimeout'), '')
        }
      } catch (ignored) {}
    }

    function setStatus(msg, cls) {
      const el = $('status')
      if (el) {
        el.textContent = msg
        el.className = 'status' + (cls ? ' ' + cls : '')
      }
    }
  })()
})

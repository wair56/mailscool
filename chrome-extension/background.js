// MailerGW Chrome Extension — Background Service Worker v1.2
// Polls for new verification codes and auto-fills into active tab

let pollInterval = null
let lastCodeSeen = ''

chrome.runtime.onInstalled.addListener(() => {
  chrome.action.setBadgeBackgroundColor({ color: '#0aff9d' })
})

// Start polling when new mailbox is created
chrome.storage.onChanged.addListener((changes) => {
  if (changes.currentEmail?.newValue) {
    lastCodeSeen = ''
    startPolling()
  }
})

function startPolling() {
  stopPolling()
  pollInterval = setInterval(checkForCode, 5000)
  checkForCode()
}

function stopPolling() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

async function checkForCode() {
  try {
    const { apiUrl, apiKey, currentEmail, currentPass, mailboxToken } =
      await chrome.storage.local.get(['apiUrl', 'apiKey', 'currentEmail', 'currentPass', 'mailboxToken'])

    const server = (apiUrl || 'https://mails.cool').replace(/\/$/, '')
    if (!currentEmail) return

    let emails = []

    if (apiKey) {
      const resp = await fetch(
        `${server}/api/emails?to=${encodeURIComponent(currentEmail)}&page=1&size=3`,
        { headers: { 'Authorization': `Bearer ${apiKey}` } }
      )
      if (resp.ok) {
        const data = await resp.json()
        emails = data.data || []
      }
    } else if (mailboxToken) {
      const resp = await fetch(`${server}/mailbox/emails?page=1&size=3`, {
        headers: { 'Authorization': `Bearer ${mailboxToken}` }
      })
      if (resp.ok) {
        const data = await resp.json()
        emails = data.data || []
      }
    } else if (currentPass) {
      // Need to login first
      const loginResp = await fetch(`${server}/mailbox/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: currentEmail, password: currentPass })
      })
      if (loginResp.ok) {
        const loginData = await loginResp.json()
        await chrome.storage.local.set({ mailboxToken: loginData.token })
        // Retry on next poll
      }
      return
    }

    for (const email of emails) {
      if (email.id && email.id !== lastCodeSeen) {
        lastCodeSeen = email.id

        // Badge
        chrome.action.setBadgeText({ text: email.code || '1' })
        chrome.action.setBadgeBackgroundColor({ color: '#00f0ff' })

        // Desktop Notification
        chrome.notifications.create({
          type: 'basic',
          iconUrl: 'icons/icon128.png',
          title: '📨 You have new mail!',
          message: `${email.subject || 'No Subject'}\nFrom: ${email.from || 'Unknown'}`,
          priority: 2
        })

        // Auto-fill code into active tab if code exists
        if (email.code) {
          try {
            const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
            if (tab) {
              chrome.tabs.sendMessage(tab.id, { action: 'fill_code', code: email.code }).catch(() => {})
            }
          } catch {}
          
          chrome.action.setBadgeText({ text: email.code })
          chrome.action.setBadgeBackgroundColor({ color: '#0aff9d' })
        }

        // Stop polling after finding any new mail
        stopPolling()
        return
      }
    }
  } catch {
    // silent
  }
}

// Handle messages from content script (e.g., quick_create widget)
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'quick_create') {
    handleQuickCreate().then(res => sendResponse(res)).catch(err => sendResponse({ error: err.message }))
    return true // indicate async response
  }
})

async function handleQuickCreate() {
  const { apiUrl, apiKey, currentEmail } = await chrome.storage.local.get(['apiUrl', 'apiKey', 'currentEmail'])

  // 1. If we already have a mailbox, just return it so it fills instantly
  if (currentEmail) {
    return { success: true, email: currentEmail, isNew: false }
  }

  // 2. Otherwise, create a new one
  const server = (apiUrl || 'https://mails.cool').replace(/\/$/, '')
  
  let resp
  if (apiKey) {
    resp = await fetch(`${server}/api/mailboxes`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${apiKey}`, 'Content-Type': 'application/json' }
    })
  } else {
    resp = await fetch(`${server}/mailbox/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    })
  }

  if (!resp.ok) {
    const err = await resp.json().catch(() => ({}))
    throw new Error(err.error || `HTTP ${resp.status}`)
  }
  
  const data = await resp.json()

  // Save to storage (triggers popup sync and background polling automatically)
  await chrome.storage.local.set({
    currentEmail: data.email,
    currentPass: data.password,
    mailboxToken: data.token || ''
  })

  return { success: true, email: data.email, isNew: true }
}

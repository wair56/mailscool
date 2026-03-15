// MailerGW Chrome Extension - Content Script
// Detects email input fields and auto-fills with the current temp email
// I18N: Uses chrome.i18n API with _locales/

const I18N = {
  get(key, val) {
    val = val || '';
    return (chrome.i18n.getMessage(key) || key) + val;
  }
};

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.action === 'fill_email' && msg.email) {
    fillEmailFields(msg.email)
    sendResponse({ success: true })
  }
  if (msg.action === 'fill_code' && msg.code) {
    fillCodeFields(msg.code)
    sendResponse({ success: true })
  }
})

function fillEmailFields(email) {
  const selectors = [
    'input[type="email"]',
    'input[name*="email"]',
    'input[name*="mail"]',
    'input[id*="email"]',
    'input[id*="mail"]',
    'input[placeholder*="email" i]',
    'input[placeholder*="邮箱"]',
    'input[placeholder*="メール"]',
    'input[placeholder*="이메일"]',
    'input[autocomplete="email"]'
  ]

  let filled = 0
  for (const sel of selectors) {
    document.querySelectorAll(sel).forEach(input => {
      if (input.offsetParent !== null) { // visible
        setNativeValue(input, email)
        filled++
      }
    })
  }

  // Flash effect on filled fields
  if (filled > 0) {
    showToast(I18N.get('fillEmail', email))
  }
}

function fillCodeFields(code) {
  const selectors = [
    'input[name*="code"]',
    'input[name*="verify"]',
    'input[name*="otp"]',
    'input[name*="captcha"]',
    'input[id*="code"]',
    'input[id*="verify"]',
    'input[id*="otp"]',
    'input[placeholder*="code" i]',
    'input[placeholder*="验证码"]',
    'input[placeholder*="認証コード"]',
    'input[placeholder*="인증코드"]'
  ]

  let filled = 0
  for (const sel of selectors) {
    document.querySelectorAll(sel).forEach(input => {
      if (input.offsetParent !== null) {
        setNativeValue(input, code)
        filled++
      }
    })
  }

  if (filled > 0) {
    showToast(I18N.get('fillCode', code))
  }
}

// Set value in a way that triggers React/Vue change events
function setNativeValue(element, value) {
  const valueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
  valueSetter.call(element, value)
  element.dispatchEvent(new Event('input', { bubbles: true }))
  element.dispatchEvent(new Event('change', { bubbles: true }))
}

function showToast(msg) {
  const existing = document.getElementById('mailergw-toast')
  if (existing) existing.remove()

  const toast = document.createElement('div')
  toast.id = 'mailergw-toast'
  toast.textContent = msg
  Object.assign(toast.style, {
    position: 'fixed',
    bottom: '20px',
    right: '20px',
    zIndex: '2147483647',
    background: '#0d1520',
    color: '#00f0ff',
    padding: '10px 16px',
    borderRadius: '8px',
    border: '1px solid rgba(0, 240, 255, 0.3)',
    fontFamily: 'monospace',
    fontSize: '12px',
    boxShadow: '0 4px 20px rgba(0, 240, 255, 0.15)',
    transition: 'opacity 0.3s ease',
    opacity: '1'
  })
  document.body.appendChild(toast)
  setTimeout(() => {
    toast.style.opacity = '0'
    setTimeout(() => toast.remove(), 300)
  }, 3000)
}

// ==========================================
// In-page Widget (Floating Generator Button)
// ==========================================

const EMAIL_SELECTORS = [
  'input[type="email"]',
  'input[name*="email" i]',
  'input[name*="mail" i]',
  'input[name*="username" i]',
  'input[name*="login" i]',
  'input[name*="identifier" i]',
  'input[id*="email" i]',
  'input[id*="mail" i]',
  'input[id*="username" i]',
  'input[id*="login" i]',
  'input[placeholder*="email" i]',
  'input[placeholder*="邮箱"]',
  'input[placeholder*="メール"]',
  'input[placeholder*="이메일"]',
  'input[placeholder*="username" i]',
  'input[placeholder*="用户名"]',
  'input[placeholder*="账号"]',
  'input[autocomplete="email"]',
  'input[autocomplete="username"]'
]

let activeWidget = null
let activeInput = null
let hideTimeout = null

function handleInputActivation(e) {
  const target = e.target
  if (target && target.tagName === 'INPUT' && target.type !== 'hidden' && target.type !== 'button' && target.type !== 'submit' && target.type !== 'password') {
    // Check if it matches email selectors
    const matches = EMAIL_SELECTORS.some(sel => {
      try { return target.matches(sel) } catch { return false }
    })
    
    console.log(`[MailsCool] Input detected: type=${target.type}, name=${target.name}, id=${target.id}, matches=${matches}`)
    
    if (matches) {
      showWidget(target)
    }
  }
}

// Use capture phase (true) to catch events before frameworks like React can stop propagation
document.addEventListener('focus', handleInputActivation, true)
document.addEventListener('click', handleInputActivation, true)

document.addEventListener('blur', (e) => {
  if (activeWidget) {
    if (e.relatedTarget && e.relatedTarget.id === 'mailergw-widget-btn') {
      return // clicked the widget
    }
    const targetInput = activeInput
    // Delay hiding to allow click
    hideTimeout = setTimeout(() => {
      // React / Vue sometimes fires synthetic blurs. Check if we really lost focus
      if (document.activeElement !== targetInput) {
        removeWidget(targetInput)
      }
    }, 250)
  }
}, true)

const calculatePosition = () => {
  if (!activeInput || !activeWidget) return
  const rect = activeInput.getBoundingClientRect()
  // If element is not visible
  if (rect.width === 0 || rect.height === 0) {
    activeWidget.style.display = 'none'
    return
  }
  
  activeWidget.style.display = 'flex'
  // Position at the right edge, relative to viewport (fixed)
  const top = rect.top + (rect.height - 24) / 2
  const left = rect.right - 28 // 4px padding from right
  
  activeWidget.style.top = `${top}px`
  activeWidget.style.left = `${left}px`
}

// Global listeners for resize/scroll with capture true for scroll inside divs
window.addEventListener('scroll', calculatePosition, true)
window.addEventListener('resize', calculatePosition, true)

function showWidget(inputElement) {
  // If we already have a widget for this input, just recalculate
  if (activeWidget && activeInput === inputElement) {
    calculatePosition()
    return
  }

  removeWidget()
  activeInput = inputElement

  const widget = document.createElement('div')
  widget.id = 'mailergw-widget-btn'
  widget.title = I18N.get('widgetTitle')
  
  // Icon and minimal styling
  widget.innerHTML = '⚡'
  
  Object.assign(widget.style, {
    position: 'fixed',
    cursor: 'pointer',
    width: '24px',
    height: '24px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #00f0ff, #0aff9d)',
    color: '#000',
    borderRadius: '4px',
    fontSize: '14px',
    fontWeight: 'bold',
    zIndex: '2147483647',
    boxShadow: '0 2px 8px rgba(0, 240, 255, 0.4)',
    userSelect: 'none',
    transition: 'transform 0.1s'
  })

  activeWidget = widget
  calculatePosition()
  
  widget.addEventListener('mouseenter', () => { widget.style.transform = 'scale(1.1)' })
  widget.addEventListener('mouseleave', () => { widget.style.transform = 'scale(1)' })
  
  // Click handler
  widget.addEventListener('mousedown', (e) => {
    e.preventDefault() // Prevent blur on input
  })
  
  widget.addEventListener('click', async (e) => {
    e.preventDefault()
    e.stopPropagation()
    if (hideTimeout) clearTimeout(hideTimeout)
    
    const originalText = widget.innerHTML
    widget.innerHTML = '...'
    widget.style.pointerEvents = 'none'

    try {
      // Send message to background to create or fetch existing
      const response = await chrome.runtime.sendMessage({ action: 'quick_create' })
      if (response && response.success && response.email) {
        setNativeValue(activeInput, response.email)
        
        if (response.isNew) {
          showToast(I18N.get('widgetSuccess', response.email))
        } else {
          showToast(I18N.get('widgetExisting', response.email))
        }
        
        removeWidget()
      } else {
        throw new Error(response?.error || 'Unknown error')
      }
    } catch (err) {
      console.error('[MailsCool] Widget error:', err)
      widget.innerHTML = '❌'
      setTimeout(() => removeWidget(), 2000)
    }
  })

  document.body.appendChild(widget)
}

function removeWidget(forInput = null) {
  if (forInput && activeInput !== forInput) return // ignore stale removes
  if (activeWidget) {
    activeWidget.remove()
    activeWidget = null
    activeInput = null
  }
}

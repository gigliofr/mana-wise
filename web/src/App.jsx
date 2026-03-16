import { useEffect, useState } from 'react'
import Auth from './components/Auth'
import Analyzer from './components/Analyzer'
import { LOCALES, translations } from './i18n'

const TOKEN_KEY = 'manawise_token'
const USER_KEY  = 'manawise_user'
const LOCALE_KEY = 'manawise_locale'

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || '')
  const [locale, setLocale] = useState(() => localStorage.getItem(LOCALE_KEY) || 'it')
  const [user,  setUser]  = useState(() => {
    try { return JSON.parse(localStorage.getItem(USER_KEY) || 'null') } catch { return null }
  })
  const messages = translations[locale] || translations.it

  function handleLogin(token, user) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(USER_KEY, JSON.stringify(user))
    setToken(token)
    setUser(user)
  }

  function handleLogout() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    setToken('')
    setUser(null)
  }

  function handleLocaleChange(nextLocale) {
    localStorage.setItem(LOCALE_KEY, nextLocale)
    setLocale(nextLocale)
  }

  useEffect(() => {
    if (!token) return
    let cancelled = false

    async function syncUser() {
      try {
        const res = await fetch('/api/v1/auth/me', {
          headers: { 'Authorization': `Bearer ${token}` },
        })
        if (!res.ok) return
        const freshUser = await res.json()
        if (cancelled) return
        localStorage.setItem(USER_KEY, JSON.stringify(freshUser))
        setUser(freshUser)
      } catch {
        // Silent best-effort sync.
      }
    }

    syncUser()
    return () => {
      cancelled = true
    }
  }, [token])

  if (!token) {
    return <Auth onLogin={handleLogin} locale={locale} messages={messages} onLocaleChange={handleLocaleChange} />
  }

  return (
    <>
      <header>
        <div className="container inner">
          <div className="logo">🔮 Mana<span>Wise</span> AI</div>
          <div className="header-actions">
            <div className="locale-switch" aria-label="Language switcher">
              {LOCALES.map(item => (
                <button
                  key={item.code}
                  type="button"
                  className={`locale-btn${locale === item.code ? ' active' : ''}`}
                  onClick={() => handleLocaleChange(item.code)}
                  title={item.label}
                >
                  <span>{item.flag}</span>
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
            <span style={{ fontSize: '.85rem', color: 'var(--muted)', alignSelf: 'center' }}>
              {user?.name} · <strong style={{ color: user?.plan === 'pro' ? '#e5a22a' : 'var(--muted)' }}>
                {user?.plan?.toUpperCase()}
              </strong>
            </span>
            <button className="btn-ghost" onClick={handleLogout}>{messages.signOut}</button>
          </div>
        </div>
      </header>
      <main>
        <div className="container">
          <Analyzer token={token} user={user} locale={locale} messages={messages} />
        </div>
      </main>
    </>
  )
}

export default App

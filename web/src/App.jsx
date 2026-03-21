import { useEffect, useState } from 'react'
import Auth from './components/Auth'
import Analyzer from './components/Analyzer'
import MatchupSimulator from './components/MatchupSimulator'
import SideboardCoach from './components/SideboardCoach'
import MulliganAssistant from './components/MulliganAssistant'
import DeckLibrary from './components/DeckLibrary'
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
  const [authChecking, setAuthChecking] = useState(true)
  const messages = translations[locale] || translations.it
  const [activeTool, setActiveTool] = useState('analyzer')
  const [sharedDecklist, setSharedDecklist] = useState('')
  const [sharedFormat, setSharedFormat] = useState('standard')

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
    setAuthChecking(false)
  }

  function handleLocaleChange(nextLocale) {
    localStorage.setItem(LOCALE_KEY, nextLocale)
    setLocale(nextLocale)
  }

  useEffect(() => {
    if (!token) {
      setAuthChecking(false)
      return
    }
    let cancelled = false

    async function checkAuth() {
      try {
        const res = await fetch('/api/v1/auth/me', {
          headers: { 'Authorization': `Bearer ${token}` },
        })
        if (!res.ok) throw new Error('Invalid session')
        const freshUser = await res.json()
        if (cancelled) return
        localStorage.setItem(USER_KEY, JSON.stringify(freshUser))
        setUser(freshUser)
      } catch {
        if (cancelled) return
        localStorage.removeItem(TOKEN_KEY)
        localStorage.removeItem(USER_KEY)
        setToken('')
        setUser(null)
      } finally {
        if (!cancelled) setAuthChecking(false)
      }
    }

    checkAuth()
    return () => {
      cancelled = true
    }
  }, [token])

  if (authChecking) {
    return (
      <main>
        <div className="container">
          <div className="card" style={{ maxWidth: 520, margin: '72px auto', textAlign: 'center' }}>
            {messages.loading}
          </div>
        </div>
      </main>
    )
  }

  if (!token) {
    return <Auth onLogin={handleLogin} locale={locale} messages={messages} onLocaleChange={handleLocaleChange} />
  }

  const tools = [
    { key: 'analyzer', label: messages.navAnalyzer },
    { key: 'matchup', label: messages.navMatchup },
    { key: 'sideboard', label: messages.navSideboard },
    { key: 'mulligan', label: messages.navMulligan },
  ]

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
          <DeckLibrary
            token={token}
            user={user}
            messages={messages}
            currentDecklist={sharedDecklist}
            currentFormat={sharedFormat}
            onSelectDeck={(decklist, format) => {
              setSharedDecklist(decklist)
              setSharedFormat(format)
            }}
          />

          <div className="tool-links" aria-label={messages.toolLinksAria}>
            {tools.map(tool => (
              <button
                key={tool.key}
                type="button"
                className={`tool-link${activeTool === tool.key ? ' active' : ''}`}
                onClick={() => setActiveTool(tool.key)}
              >
                {tool.label}
              </button>
            ))}
          </div>

          {activeTool === 'analyzer' && (
            <Analyzer
              token={token}
              user={user}
              locale={locale}
              messages={messages}
              decklist={sharedDecklist}
              format={sharedFormat}
              onDeckChange={setSharedDecklist}
              onFormatChange={setSharedFormat}
            />
          )}
          {activeTool === 'matchup' && (
            <MatchupSimulator
              token={token}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
          {activeTool === 'sideboard' && (
            <SideboardCoach
              token={token}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
          {activeTool === 'mulligan' && (
            <MulliganAssistant
              token={token}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
        </div>
      </main>
    </>
  )
}

export default App

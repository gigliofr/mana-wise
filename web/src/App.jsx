import { useEffect, useState } from 'react'
import Auth from './components/Auth'
import Analyzer from './components/Analyzer'
import MatchupSimulator from './components/MatchupSimulator'
import SideboardCoach from './components/SideboardCoach'
import MulliganAssistant from './components/MulliganAssistant'
import DeckLibrary from './components/DeckLibrary'
import VisualDeckBuilder from './components/VisualDeckBuilder'
import PlansSupport from './components/PlansSupport'
import LegalFooter from './components/LegalFooter'
import { LegalPage } from './components/LegalPages'
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
  const [activeTool, setActiveTool] = useState('analyzer')
  const [sharedDecklist, setSharedDecklist] = useState('')
  const [sharedFormat, setSharedFormat] = useState('standard')
  const currentPath = window.location.pathname.toLowerCase()
  const isLegalPage = ['/privacy', '/cookie', '/contatti'].includes(currentPath)

  function handleSessionUpdate(nextToken, nextUser) {
    localStorage.setItem(TOKEN_KEY, nextToken)
    localStorage.setItem(USER_KEY, JSON.stringify(nextUser))
    setToken(nextToken)
    setUser(nextUser)
  }

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
        if (res.status === 401 || res.status === 403) {
          if (!cancelled) {
            localStorage.removeItem(TOKEN_KEY)
            localStorage.removeItem(USER_KEY)
            setToken('')
            setUser(null)
          }
          return
        }
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

  if (isLegalPage) {
    return (
      <>
        <header>
          <div className="container inner">
            <div className="logo">🔮 Mana<span>Wise</span> AI</div>
            <div className="header-actions">
              <a className="btn-ghost" href="/">{messages.backToApp}</a>
            </div>
          </div>
        </header>
        <main>
          <div className="container">
            <LegalPage path={currentPath} messages={messages} />
          </div>
        </main>
        <LegalFooter messages={messages} />
      </>
    )
  }

  if (!token) {
    return (
      <>
        <Auth onLogin={handleLogin} locale={locale} messages={messages} onLocaleChange={handleLocaleChange} />
        <LegalFooter messages={messages} />
      </>
    )
  }

  const tools = [
    { key: 'analyzer', label: messages.navAnalyzer },
    { key: 'builder', label: messages.navBuilder || 'Builder' },
    { key: 'matchup', label: messages.navMatchup },
    { key: 'sideboard', label: messages.navSideboard },
    { key: 'mulligan', label: messages.navMulligan },
    { key: 'plans', label: messages.navPlans },
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
          {activeTool === 'builder' && (
            <VisualDeckBuilder
              token={token}
              messages={messages}
              decklist={sharedDecklist}
              onDeckChange={setSharedDecklist}
            />
          )}
          {activeTool === 'matchup' && (
            <MatchupSimulator
              token={token}
              user={user}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
          {activeTool === 'sideboard' && (
            <SideboardCoach
              token={token}
              user={user}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
          {activeTool === 'mulligan' && (
            <MulliganAssistant
              token={token}
              user={user}
              decklist={sharedDecklist}
              format={sharedFormat}
              messages={messages}
            />
          )}
          {activeTool === 'plans' && (
            <PlansSupport
              token={token}
              user={user}
              messages={messages}
              onSessionUpdate={handleSessionUpdate}
            />
          )}
        </div>
      </main>
      <LegalFooter messages={messages} />
    </>
  )
}

export default App

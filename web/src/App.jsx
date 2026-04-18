import { useEffect, useState } from 'react'
import Auth from './components/Auth'
import Analyzer from './components/Analyzer'
import MatchupSimulator from './components/MatchupSimulator'
import SideboardCoach from './components/SideboardCoach'
import MulliganAssistant from './components/MulliganAssistant'
import DeckLibrary from './components/DeckLibrary'
import VisualDeckBuilder from './components/VisualDeckBuilder'
import PlansSupport from './components/PlansSupport'
import NotificationsCenter from './components/NotificationsCenter'
import AdminCommanderBrackets from './components/AdminCommanderBrackets'
import LegalFooter from './components/LegalFooter'
import { LegalPage } from './components/LegalPages'
import { LOCALES, translations } from './i18n'
import { configureApiAuthSession } from './lib/apiClient'

const TOKEN_KEY = 'manawise_token'
const REFRESH_TOKEN_KEY = 'manawise_refresh_token'
const USER_KEY  = 'manawise_user'
const LOCALE_KEY = 'manawise_locale'
const THEME_KEY = 'manawise_theme'
const SESSION_META_KEY = 'manawise_session_meta'

function safeReadJSON(raw) {
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function decodeJWTPayload(token) {
  const parts = String(token || '').split('.')
  if (parts.length < 2) return null
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    return JSON.parse(atob(padded))
  } catch {
    return null
  }
}

function buildSessionMetaFromToken(token, fallback = null) {
  const payload = decodeJWTPayload(token)
  const exp = Number(payload?.exp || 0)
  const iat = Number(payload?.iat || 0)
  if (exp > 0) {
    const fallbackDuration = Number(fallback?.durationMinutes || fallback?.session_ttl_minutes || 0)
    const durationMinutes = iat > 0 && exp > iat
      ? Math.max(1, Math.round((exp - iat) / 60))
      : (fallbackDuration > 0 ? fallbackDuration : 0)
    return {
      issuedAt: iat > 0 ? new Date(iat * 1000).toISOString() : '',
      expiresAt: new Date(exp * 1000).toISOString(),
      durationMinutes,
    }
  }

  const ttlMinutes = Number(fallback?.durationMinutes || fallback?.session_ttl_minutes || 0)
  if (ttlMinutes > 0) {
    const now = Date.now()
    return {
      issuedAt: new Date(now).toISOString(),
      expiresAt: new Date(now + ttlMinutes * 60 * 1000).toISOString(),
      durationMinutes: ttlMinutes,
    }
  }
  return null
}

function formatDurationMinutes(totalMinutes, localeCode) {
  const minutes = Math.max(0, Number(totalMinutes) || 0)
  const hours = Math.floor(minutes / 60)
  const rem = minutes % 60
  if (localeCode === 'it') {
    if (hours > 0 && rem > 0) return `${hours}h ${rem}m`
    if (hours > 0) return `${hours}h`
    return `${rem}m`
  }
  if (hours > 0 && rem > 0) return `${hours}h ${rem}m`
  if (hours > 0) return `${hours}h`
  return `${rem}m`
}

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || '')
  const [locale, setLocale] = useState(() => localStorage.getItem(LOCALE_KEY) || 'it')
  const [theme, setTheme] = useState(() => localStorage.getItem(THEME_KEY) || 'night')
  const [sessionMeta, setSessionMeta] = useState(() => safeReadJSON(localStorage.getItem(SESSION_META_KEY)))
  const [clockNow, setClockNow] = useState(() => Date.now())
  const [user,  setUser]  = useState(() => {
    try { return JSON.parse(localStorage.getItem(USER_KEY) || 'null') } catch { return null }
  })
  const messages = translations[locale] || translations.it
  const [activeTool, setActiveTool] = useState('analyzer')
  const [plansFocusKey, setPlansFocusKey] = useState(0)
  const [deckWorkspace, setDeckWorkspace] = useState({
    decklist: '',
    format: 'standard',
    deckId: '',
    deckName: '',
  })
  const currentPath = window.location.pathname.toLowerCase()
  const isLegalPage = ['/privacy', '/cookie', '/contatti'].includes(currentPath)
  const hasActivePro = user?.plan === 'pro' && (!user?.pro_until || new Date(user.pro_until) > new Date())
  const isHiddenAdmin = String(user?.email || '').toLowerCase().includes('admin')

  useEffect(() => {
    const intervalId = window.setInterval(() => setClockNow(Date.now()), 60_000)
    return () => window.clearInterval(intervalId)
  }, [])

  function resolveSessionMeta(nextToken, maybeMeta) {
    return buildSessionMetaFromToken(nextToken, maybeMeta)
  }

  function handleSessionUpdate(nextToken, nextRefreshToken, nextUser, nextSessionMeta = null) {
    localStorage.setItem(TOKEN_KEY, nextToken)
    if (nextRefreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, nextRefreshToken)
    }
    localStorage.setItem(USER_KEY, JSON.stringify(nextUser))
    const normalizedSession = resolveSessionMeta(nextToken, nextSessionMeta)
    if (normalizedSession) {
      localStorage.setItem(SESSION_META_KEY, JSON.stringify(normalizedSession))
      setSessionMeta(normalizedSession)
    } else {
      localStorage.removeItem(SESSION_META_KEY)
      setSessionMeta(null)
    }
    setToken(nextToken)
    setUser(nextUser)
  }

  function handleLogin(token, user, refreshToken = '', nextSessionMeta = null) {
    localStorage.setItem(TOKEN_KEY, token)
    if (refreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
    }
    localStorage.setItem(USER_KEY, JSON.stringify(user))
    const normalizedSession = resolveSessionMeta(token, nextSessionMeta)
    if (normalizedSession) {
      localStorage.setItem(SESSION_META_KEY, JSON.stringify(normalizedSession))
      setSessionMeta(normalizedSession)
    } else {
      localStorage.removeItem(SESSION_META_KEY)
      setSessionMeta(null)
    }
    setToken(token)
    setUser(user)
  }

  function handleLogout() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(SESSION_META_KEY)
    setToken('')
    setUser(null)
    setSessionMeta(null)
  }

  const sessionDurationMinutes = Number(sessionMeta?.durationMinutes || 0)
  const sessionDurationText = sessionDurationMinutes > 0
    ? formatDurationMinutes(sessionDurationMinutes, locale)
    : ''
  const expiresMs = sessionMeta?.expiresAt ? new Date(sessionMeta.expiresAt).getTime() : 0
  const remainingMinutes = expiresMs > 0 ? Math.max(0, Math.ceil((expiresMs - clockNow) / 60000)) : 0
  const sessionRemainingText = remainingMinutes > 0
    ? formatDurationMinutes(remainingMinutes, locale)
    : ''

  function handleLocaleChange(nextLocale) {
    localStorage.setItem(LOCALE_KEY, nextLocale)
    setLocale(nextLocale)
  }

  function handleThemeToggle() {
    setTheme(prev => {
      const next = prev === 'night' ? 'day' : 'night'
      localStorage.setItem(THEME_KEY, next)
      return next
    })
  }

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    document.documentElement.style.colorScheme = theme === 'day' ? 'light' : 'dark'
  }, [theme])

  useEffect(() => {
    configureApiAuthSession({
      getToken: () => localStorage.getItem(TOKEN_KEY) || '',
      getRefreshToken: () => localStorage.getItem(REFRESH_TOKEN_KEY) || '',
      onSessionUpdate: handleSessionUpdate,
      onUnauthorized: handleLogout,
    })
  }, [token, user])

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
            localStorage.removeItem(REFRESH_TOKEN_KEY)
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
    { key: 'notifications', label: messages.navNotifications || 'Notifiche' },
    ...(isHiddenAdmin ? [{ key: 'admin', label: messages.navAdmin || 'Admin' }] : []),
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
            <button type="button" className="btn-ghost" onClick={handleThemeToggle}>
              {theme === 'night' ? '☀️ Giorno' : '🌙 Notte'}
            </button>
            <span style={{ fontSize: '.85rem', color: 'var(--muted)', alignSelf: 'center', display: 'inline-flex', flexDirection: 'column', alignItems: 'flex-end', lineHeight: 1.2 }}>
              <span>
                {user?.name} · <strong style={{ color: user?.plan === 'pro' ? '#e5a22a' : 'var(--muted)' }}>
                  {user?.plan?.toUpperCase()}
                </strong>
              </span>
              {(sessionDurationText || sessionRemainingText) && (
                <span style={{ fontSize: '.72rem', opacity: .9 }}>
                  {messages.sessionDurationLabel?.(sessionDurationText || '--') || `Session ${sessionDurationText || '--'}`}
                  {sessionRemainingText ? ` · ${messages.sessionRemainingLabel?.(sessionRemainingText) || `left ${sessionRemainingText}`}` : ''}
                </span>
              )}
            </span>
            <button
              type="button"
              className={`btn-ghost${activeTool === 'plans' ? ' header-tool-active' : ''}`}
              onClick={() => setActiveTool('plans')}
            >
              {messages.navPlans}
            </button>
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
            currentDecklist={deckWorkspace.decklist}
            currentFormat={deckWorkspace.format}
            onSelectDeck={(decklist, format, deck) => {
              setDeckWorkspace(prev => ({
                ...prev,
                decklist,
                format,
                deckId: deck?.id || '',
                deckName: deck?.name || '',
              }))
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
              decklist={deckWorkspace.decklist}
              format={deckWorkspace.format}
              onDeckChange={nextDecklist => setDeckWorkspace(prev => ({ ...prev, decklist: nextDecklist }))}
              onFormatChange={nextFormat => setDeckWorkspace(prev => ({ ...prev, format: nextFormat }))}
              onUpgradeRequest={() => {
                setPlansFocusKey(prev => prev + 1)
                setActiveTool('plans')
              }}
            />
          )}
          {activeTool === 'builder' && (
            hasActivePro ? (
              <VisualDeckBuilder
                token={token}
                messages={messages}
                decklist={deckWorkspace.decklist}
                onDeckChange={nextDecklist => setDeckWorkspace(prev => ({ ...prev, decklist: nextDecklist }))}
              />
            ) : (
              <div className="card">
                <h2>{messages.builderProOnlyTitle || 'Builder Pro'}</h2>
                <p style={{ color: 'var(--muted)' }}>
                  {messages.builderProOnlyBody || 'Il Visual Deck Builder avanzato è disponibile nel piano Pro.'}
                </p>
                <button type="button" className="btn-primary" onClick={() => setActiveTool('plans')}>
                  {messages.navPlans}
                </button>
              </div>
            )
          )}
          {activeTool === 'matchup' && (
            <MatchupSimulator
              token={token}
              user={user}
              decklist={deckWorkspace.decklist}
              format={deckWorkspace.format}
              messages={messages}
            />
          )}
          {activeTool === 'sideboard' && (
            <SideboardCoach
              token={token}
              user={user}
              decklist={deckWorkspace.decklist}
              format={deckWorkspace.format}
              messages={messages}
            />
          )}
          {activeTool === 'mulligan' && (
            <MulliganAssistant
              token={token}
              user={user}
              decklist={deckWorkspace.decklist}
              format={deckWorkspace.format}
              messages={messages}
            />
          )}
          {activeTool === 'notifications' && (
            <NotificationsCenter
              token={token}
              locale={locale}
              messages={messages}
            />
          )}
          {activeTool === 'plans' && (
            <PlansSupport
              token={token}
              user={user}
              messages={messages}
              onSessionUpdate={handleSessionUpdate}
              focusProActivationKey={plansFocusKey}
            />
          )}
          {activeTool === 'admin' && isHiddenAdmin && (
            <AdminCommanderBrackets token={token} user={user} messages={messages} />
          )}
        </div>
      </main>
      <LegalFooter messages={messages} />
    </>
  )
}

export default App

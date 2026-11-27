import { useEffect, useState } from 'react'
import Auth from './components/Auth'
import Analyzer from './components/Analyzer'
import MatchupSimulator from './components/MatchupSimulator'
import SideboardCoach from './components/SideboardCoach'
import MulliganAssistant from './components/MulliganAssistant'
import MyDecks, { formatLabel } from './components/MyDecks'
import DeckListEditor from './components/DeckListEditor'
import { deckHealthScore, scoreTone } from './components/AnalysisDashboard'
import { ColorDots } from './components/ManaPips'
import PlansSupport from './components/PlansSupport'
import NotificationsCenter from './components/NotificationsCenter'
import AdminCommanderBrackets from './components/AdminCommanderBrackets'
import LegalFooter from './components/LegalFooter'
import { LegalPage } from './components/LegalPages'
import { LOCALES, translations } from './i18n'
import { workspaceMessages } from './i18nWorkspace'
import { parseDecklist, sum, mainText, listToText, loadCurrentDeck, saveCurrentDeck, loadInsights, saveInsight } from './lib/deck'
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

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || '')
  const [locale, setLocale] = useState(() => localStorage.getItem(LOCALE_KEY) || 'it')
  const [theme, setTheme] = useState(() => localStorage.getItem(THEME_KEY) || 'night')
  const [sessionMeta, setSessionMeta] = useState(() => safeReadJSON(localStorage.getItem(SESSION_META_KEY)))
  const [user,  setUser]  = useState(() => {
    try { return JSON.parse(localStorage.getItem(USER_KEY) || 'null') } catch { return null }
  })
  const messages = { ...(translations[locale] || translations.it), ...(workspaceMessages[locale] || workspaceMessages.it) }
  const [plansFocusKey, setPlansFocusKey] = useState(0)
  const [currentDeck, setCurrentDeck] = useState(() => loadCurrentDeck())
  const [view, setView] = useState(() => (loadCurrentDeck() ? 'deck' : 'home'))
  const [tool, setTool] = useState(() => localStorage.getItem('manawise_tool') || 'analysis')
  const [autoRun, setAutoRun] = useState(false)
  const [insightsNonce, setInsightsNonce] = useState(0)
  const currentPath = window.location.pathname.toLowerCase()
  const isLegalPage = ['/privacy', '/cookie', '/contatti'].includes(currentPath)
  const hasActivePro = user?.plan === 'pro' && (!user?.pro_until || new Date(user.pro_until) > new Date())
  const isHiddenAdmin = String(user?.email || '').toLowerCase().includes('admin')

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
  }, [])

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

  const deckParsed = currentDeck ? parseDecklist(currentDeck.text) : null
  const deckInsight = currentDeck?.id ? (loadInsights()[currentDeck.id] || {}) : (currentDeck?.insight || {})
  const deckColors = deckInsight.colors || []

  function openDeck(deck, opts = {}) {
    const next = { ...deck }
    setCurrentDeck(next)
    saveCurrentDeck(next)
    setView('deck')
    selectTool(opts.tool || 'analysis')
    setAutoRun(Boolean(opts.autoRun))
    window.scrollTo(0, 0)
  }

  function updateDeck(patch) {
    setCurrentDeck(prev => {
      const next = { ...prev, ...patch }
      saveCurrentDeck(next)
      return next
    })
  }

  function selectTool(key) {
    setTool(key)
    localStorage.setItem('manawise_tool', key)
  }

  function handleAnalyzed(result, fingerprint) {
    setAutoRun(false)
    const score = deckHealthScore(result)
    const mana = result?.deterministic?.mana || {}
    const colors = fingerprint?.color_identity?.length
      ? fingerprint.color_identity
      : Object.keys(mana.color_distribution || {}).filter(k => k !== 'C' && mana.color_distribution[k] > 0)
    const tone = scoreTone(score)
    const insight = { score, tone, colors, headline: messages[`wVerdict_${tone}`] }
    if (currentDeck?.id) saveInsight(currentDeck.id, insight)
    else updateDeck({ insight })
    setInsightsNonce(n => n + 1)
  }

  const deckTools = [
    { key: 'analysis', label: messages.navAnalyzer },
    { key: 'list', label: messages.wNavList },
    { key: 'matchup', label: messages.navMatchup },
    { key: 'sideboard', label: messages.navSideboard },
    { key: 'mulligan', label: messages.navMulligan },
  ]

  const mainNav = [
    { key: 'home', label: messages.wMyDecks },
    { key: 'notifications', label: messages.navNotifications || 'Notifiche' },
    { key: 'plans', label: messages.navPlans },
    ...(isHiddenAdmin ? [{ key: 'admin', label: messages.navAdmin || 'Admin' }] : []),
  ]
  const activeNav = view === 'deck' ? 'home' : view
  const initial = String(user?.name || user?.email || '?').trim().charAt(0).toUpperCase()

  return (
    <>
      <header className="mw-header">
        <div className="mw-header__inner">
          <button type="button" className="mw-logo" onClick={() => setView(currentDeck ? 'deck' : 'home')}>Mana<span>Wise</span></button>
          <nav className="mw-nav" aria-label={messages.wMainNav}>
            {mainNav.map(item => (
              <button key={item.key} type="button" className={`mw-nav__link${activeNav === item.key ? ' is-active' : ''}`} aria-current={activeNav === item.key ? 'page' : undefined}
                onClick={() => { setView(item.key); if (item.key === 'plans') setPlansFocusKey(k => k + 1) }}>
                {item.label}
              </button>
            ))}
          </nav>
          <details className="mw-user">
            <summary className="mw-user__btn">
              <span className="mw-avatar" aria-hidden="true">{initial}</span>
              <span className="mw-user__name">{user?.name}</span>
              {hasActivePro && <span className="mw-pro">PRO</span>}
            </summary>
            <div className="mw-user__panel">
              <p className="mw-muted mw-small">{user?.email}</p>
              <div className="mw-seg" role="group" aria-label={messages.wLanguage}>
                {LOCALES.map(item => (
                  <button key={item.code} type="button" className={locale === item.code ? 'is-active' : ''} aria-pressed={locale === item.code} onClick={() => handleLocaleChange(item.code)}>{item.label}</button>
                ))}
              </div>
              <button type="button" className="mw-user__item" onClick={handleThemeToggle}>{theme === 'night' ? messages.wThemeDay : messages.wThemeNight}</button>
              <button type="button" className="mw-user__item" onClick={() => setView('plans')}>{messages.navPlans}{user?.plan ? ` · ${String(user.plan).toUpperCase()}` : ''}</button>
              <button type="button" className="mw-user__item" onClick={handleLogout}>{messages.signOut}</button>
            </div>
          </details>
        </div>
      </header>

      {view === 'deck' && currentDeck && (
        <section className="mw-deckbar" aria-label={messages.wCurrentDeck}>
          <div className="mw-deckbar__inner">
            <div className="mw-deckbar__id">
              <ColorDots colors={deckColors} label={messages.wColorsAria} size="lg" />
              <div className="mw-deckbar__text">
                <h1>{currentDeck.name}</h1>
                <p className="mw-muted mw-small">
                  {messages.wDeckMeta(formatLabel(currentDeck.format), sum(deckParsed.main) + sum(deckParsed.commander), sum(deckParsed.side))}
                  {' · '}{currentDeck.dirty ? messages.wDirtyTag : (currentDeck.id ? messages.wSavedTag : messages.wUnsavedTag)}
                </p>
              </div>
              <div className="mw-deckbar__actions">
                <button type="button" className="mw-btn mw-btn--ghost mw-btn--sm" onClick={() => setView('home')}>{messages.wChangeDeck}</button>
              </div>
            </div>
            <nav className="mw-tools" aria-label={messages.wToolsNav}>
              {deckTools.map(t => (
                <button key={t.key} type="button" className={`mw-tools__link${tool === t.key ? ' is-active' : ''}`} aria-current={tool === t.key ? 'page' : undefined} onClick={() => selectTool(t.key)}>{t.label}</button>
              ))}
            </nav>
          </div>
        </section>
      )}

      <main className="mw-main">
        <div className="container mw-container">
          {view === 'home' && (
            <MyDecks key={insightsNonce} token={token} user={user} messages={messages} locale={locale} onOpenDeck={openDeck} />
          )}

          {view === 'deck' && currentDeck && tool === 'analysis' && (
            <Analyzer
              token={token}
              user={user}
              locale={locale}
              messages={messages}
              decklist={currentDeck.text}
              format={currentDeck.format}
              deckName={currentDeck.name}
              autoRun={autoRun}
              onAnalyzed={handleAnalyzed}
              onEditList={() => selectTool('list')}
              onUpgradeRequest={() => { setPlansFocusKey(k => k + 1); setView('plans') }}
            />
          )}
          {view === 'deck' && currentDeck && tool === 'list' && (
            <DeckListEditor
              token={token}
              user={user}
              deck={currentDeck}
              colors={deckColors}
              messages={messages}
              onChange={text => updateDeck({ text, dirty: true })}
              onSaved={id => updateDeck({ id, dirty: false })}
              onReanalyze={() => { setAutoRun(true); selectTool('analysis') }}
            />
          )}
          {view === 'deck' && currentDeck && tool === 'matchup' && (
            <MatchupSimulator token={token} user={user} messages={messages} deckName={currentDeck.name}
              decklist={mainText(deckParsed)} sideboard={listToText(deckParsed.side)} format={currentDeck.format} />
          )}
          {view === 'deck' && currentDeck && tool === 'sideboard' && (
            <SideboardCoach token={token} user={user} messages={messages} deckName={currentDeck.name}
              decklist={mainText(deckParsed)} sideboard={listToText(deckParsed.side)} format={currentDeck.format} />
          )}
          {view === 'deck' && currentDeck && tool === 'mulligan' && (
            <MulliganAssistant token={token} user={user} messages={messages} deckName={currentDeck.name}
              decklist={mainText(deckParsed)} format={currentDeck.format} />
          )}

          {view === 'notifications' && <NotificationsCenter token={token} locale={locale} messages={messages} />}
          {view === 'plans' && (
            <PlansSupport token={token} user={user} messages={messages} onSessionUpdate={handleSessionUpdate} focusProActivationKey={plansFocusKey} />
          )}
          {view === 'admin' && isHiddenAdmin && <AdminCommanderBrackets token={token} user={user} messages={messages} />}
        </div>
      </main>
      <LegalFooter messages={messages} />
    </>
  )
}

export default App

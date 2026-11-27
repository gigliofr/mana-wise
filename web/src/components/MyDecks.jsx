// Pagina iniziale: importa un mazzo e scegli tra quelli salvati.
import { useEffect, useMemo, useState } from 'react'
import { apiRequest } from '../lib/apiClient'
import {
  parseDecklist, sum, guessFormat, apiCardsFromParsed, parsedFromApiDeck, deckToText,
  defaultDeckName, loadInsights, relativeTime, SAMPLE_DECK,
} from '../lib/deck'
import { ColorDots } from './ManaPips'
import DeckLibrary from './DeckLibrary'

export const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

export function formatLabel(f) {
  return f ? f.charAt(0).toUpperCase() + f.slice(1) : ''
}

export default function MyDecks({ token, user, messages: m, locale, onOpenDeck }) {
  const [text, setText] = useState('')
  const [name, setName] = useState('')
  const [format, setFormat] = useState('')
  const [decks, setDecks] = useState([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('all')
  const [reload, setReload] = useState(0)
  const insights = useMemo(() => loadInsights(), [reload])

  const parsed = useMemo(() => parseDecklist(text), [text])
  const count = sum(parsed.main) + sum(parsed.commander)
  const guessed = count ? guessFormat(parsed) : ''
  const effectiveFormat = format || guessed || 'modern'

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      try {
        const { res, data } = await apiRequest('/decks', { token })
        if (!cancelled && res.ok) {
          const all = Array.isArray(data) ? data : (Array.isArray(data?.items) ? data.items : [])
          const own = user?.id ? all.filter(d => !d.user_id || d.user_id === user.id) : all
          own.sort((a, b) => new Date(b.updated_at || 0) - new Date(a.updated_at || 0))
          setDecks(own)
        }
      } catch {
        // l'elenco resta vuoto: si può comunque importare un mazzo
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    if (token) load()
    return () => { cancelled = true }
  }, [token, user?.id, reload])

  function useSample() {
    setText(SAMPLE_DECK)
    setName('Boros Burn')
    setFormat('modern')
    setError('')
  }

  async function importDeck(target) {
    setError('')
    if (!count) {
      setError(m.wPasteFirst)
      return
    }
    const deckName = name.trim() || defaultDeckName(parsed, locale)
    const fullText = deckToText(parsed)
    setSaving(true)
    let saved = null
    try {
      const { res, data } = await apiRequest('/decks', {
        token,
        method: 'POST',
        body: { name: deckName, format: effectiveFormat, cards: apiCardsFromParsed(parsed), is_public: false },
      })
      if (res.ok) saved = data
      else setError(`${m.wNotSaved} ${data?.error || ''}`.trim())
    } catch {
      setError(m.wNotSaved)
    } finally {
      setSaving(false)
    }
    onOpenDeck({ id: saved?.id || '', name: deckName, format: effectiveFormat, text: fullText }, { tool: target, autoRun: target === 'analysis' })
  }

  function openSaved(deck) {
    const p = parsedFromApiDeck(deck)
    onOpenDeck({ id: deck.id, name: deck.name, format: deck.format || 'modern', text: deckToText(p) }, { tool: 'analysis', autoRun: false })
  }

  function startEmpty() {
    onOpenDeck({ id: '', name: name.trim() || defaultDeckName({ main: [], commander: [] }, locale), format: effectiveFormat, text: '' }, { tool: 'list', autoRun: false })
  }

  const formats = Array.from(new Set(decks.map(d => d.format).filter(Boolean)))
  const visible = filter === 'all' ? decks : decks.filter(d => d.format === filter)

  return (
    <div className="mw-home">
      <section className="mw-import" aria-labelledby="mwImportTitle">
        <div className="mw-import__intro">
          <h1 id="mwImportTitle">{m.wImportTitle}</h1>
          <p className="mw-muted">{m.wImportText}</p>
          <p className="mw-muted">{m.wNoDeckAtHand} <button type="button" className="mw-link" onClick={useSample}>{m.wTrySample}</button></p>
        </div>
        <div className="mw-import__form">
          <label htmlFor="mwDeckText">{m.wDeckList}</label>
          <textarea id="mwDeckText" rows={7} value={text} onChange={e => setText(e.target.value)} placeholder={'4 Lightning Bolt\n4 Goblin Guide\n…'} spellCheck={false} />
          {count > 0 && (
            <p className="mw-detect" role="status">
              {m.wDetected(count, sum(parsed.side))}
              {guessed && !format ? ` · ${m.wFormatGuess(formatLabel(guessed))}` : ''}
            </p>
          )}
          <div className="mw-import__row">
            <div>
              <label htmlFor="mwDeckName">{m.wDeckName}</label>
              <input id="mwDeckName" value={name} onChange={e => setName(e.target.value)} placeholder={count ? defaultDeckName(parsed, locale) : m.wDeckNamePh} />
            </div>
            <div>
              <label htmlFor="mwDeckFormat">{m.format}</label>
              <select id="mwDeckFormat" value={format || guessed || 'modern'} onChange={e => setFormat(e.target.value)}>
                {FORMATS.map(f => <option key={f} value={f}>{formatLabel(f)}</option>)}
              </select>
            </div>
          </div>
          {error && <p className="mw-error" role="alert">{error}</p>}
          <div className="mw-import__actions">
            <button type="button" className="mw-btn mw-btn--primary" onClick={() => importDeck('analysis')} disabled={saving}>{saving ? m.wSaving : m.wImportAnalyze}</button>
            <button type="button" className="mw-btn mw-btn--ghost" onClick={startEmpty}>{m.wStartBuilder}</button>
          </div>
        </div>
      </section>

      <div className="mw-section-head">
        <h2>{m.wYourDecks(decks.length)}</h2>
        {formats.length > 1 && (
          <div className="mw-pills" role="group" aria-label={m.wFilterFormat}>
            <button type="button" className={`mw-pill${filter === 'all' ? ' is-active' : ''}`} aria-pressed={filter === 'all'} onClick={() => setFilter('all')}>{m.wAll}</button>
            {formats.map(f => (
              <button key={f} type="button" className={`mw-pill${filter === f ? ' is-active' : ''}`} aria-pressed={filter === f} onClick={() => setFilter(f)}>{formatLabel(f)}</button>
            ))}
          </div>
        )}
      </div>

      {loading ? (
        <p className="mw-muted">{m.loading || '…'}</p>
      ) : (
        <div className="mw-deck-grid">
          {visible.map(deck => {
            const info = insights[deck.id] || {}
            const p = parsedFromApiDeck(deck)
            const total = sum(p.main) + sum(p.commander)
            return (
              <button key={deck.id} type="button" className="mw-deck-card" onClick={() => openSaved(deck)}>
                <div className="mw-deck-card__top">
                  <ColorDots colors={info.colors || []} label={m.wColorsAria} />
                  <span className="mw-tag">{formatLabel(deck.format)}</span>
                </div>
                <div>
                  <p className="mw-deck-card__name">{deck.name}</p>
                  <p className="mw-muted mw-small">
                    {info.at ? m.wAnalyzedAgo(relativeTime(info.at, locale)) : m.wCardsCount(total, sum(p.side))}
                  </p>
                </div>
                <div className="mw-deck-card__foot">
                  {typeof info.score === 'number' ? (
                    <>
                      <span className={`mw-mini-score mw-mini-score--${info.tone || 'warn'}`}>{info.score.toLocaleString('it-IT', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}</span>
                      <span className="mw-muted mw-small">{info.headline || ''}</span>
                    </>
                  ) : (
                    <span className="mw-muted mw-small">{m.wNotAnalyzed}</span>
                  )}
                </div>
              </button>
            )
          })}
          <button type="button" className="mw-deck-card mw-deck-card--new" onClick={() => document.getElementById('mwDeckText')?.focus()}>
            <span className="mw-plus" aria-hidden="true">+</span>
            <span>{m.wNewDeck}</span>
          </button>
        </div>
      )}

      <details className="mw-advanced">
        <summary>{m.wManageLibrary}</summary>
        <DeckLibrary
          token={token}
          user={user}
          messages={m}
          currentDecklist={text}
          currentFormat={effectiveFormat}
          onSelectDeck={(decklist, fmt, deck) => {
            onOpenDeck({ id: deck?.id || '', name: deck?.name || defaultDeckName(parseDecklist(decklist), locale), format: fmt || 'modern', text: decklist }, { tool: 'analysis', autoRun: false })
          }}
        />
        <button type="button" className="mw-link" onClick={() => setReload(r => r + 1)}>{m.wRefreshList}</button>
      </details>
    </div>
  )
}

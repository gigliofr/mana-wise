import { useEffect, useMemo, useState } from 'react'
import CardHoverPreview from './CardHoverPreview'

const API = '/api/v1'

export default function DeckLibrary({
  token,
  user,
  messages,
  currentDecklist,
  currentFormat,
  onSelectDeck,
}) {
  const [decks, setDecks] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [saveStep, setSaveStep] = useState(0) // 0=hidden, 1=name, 2=confirm
  const [editedDecklist, setEditedDecklist] = useState('')
  const [page, setPage] = useState(0)
  const [expandedDecks, setExpandedDecks] = useState({})
  const [deckLegality, setDeckLegality] = useState({})

  const ITEMS_PER_PAGE = 3
  const paginatedDecks = decks.slice(page * ITEMS_PER_PAGE, (page + 1) * ITEMS_PER_PAGE)
  const totalPages = Math.ceil(decks.length / ITEMS_PER_PAGE)

  const isPro = (user?.plan || '').toLowerCase() === 'pro'
  const canSaveMore = isPro || decks.length < 1

  const activeSummary = useMemo(() => {
    const lines = (currentDecklist || '').split('\n').map(l => l.trim()).filter(Boolean)
    return {
      cards: lines.length,
      format: currentFormat || 'standard',
    }
  }, [currentDecklist, currentFormat])

  useEffect(() => {
    let cancelled = false
    async function loadDecks() {
      setLoading(true)
      setError('')
      try {
        const res = await fetch(`${API}/decks`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        const data = await res.json()
        if (!res.ok) throw new Error(data.error || messages.deckLoadFailed)
        if (!cancelled) {
          const allDecks = Array.isArray(data) ? data : []
          const ownedDecks = user?.id
            ? allDecks.filter(d => d?.user_id === user.id)
            : allDecks
          setDecks(ownedDecks)
        }
      } catch (err) {
        if (!cancelled) setError(err.message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    loadDecks()
    return () => {
      cancelled = true
    }
  }, [token, messages.deckLoadFailed])

  useEffect(() => {
    let cancelled = false

    async function ensureLegality() {
      const pending = paginatedDecks
        .filter(deck => deck?.id && !deckLegality[deck.id])
        .map(deck => deck.id)

      if (pending.length === 0) return

      if (!cancelled) {
        setDeckLegality(prev => {
          const next = { ...prev }
          pending.forEach(deckID => {
            if (!next[deckID]) {
              next[deckID] = {
                loading: true,
                formats: {},
                cardIllegalByFormat: {},
              }
            }
          })
          return next
        })
      }

      await Promise.all(
        pending.map(async deckID => {
          try {
            const res = await fetch(`${API}/decks/${deckID}/legality`, {
              headers: { Authorization: `Bearer ${token}` },
            })
            const data = await res.json()
            if (!res.ok) throw new Error(data.error || 'legality fetch failed')
            const formatMap = data?.formats || {}
            const cardIllegalByFormat = {}
            Object.keys(formatMap).forEach(fmt => {
              const illegal = formatMap?.[fmt]?.illegal_cards || []
              const names = {}
              illegal.forEach(item => {
                const n = (item?.card_name || '').trim().toLowerCase()
                if (n) names[n] = true
              })
              cardIllegalByFormat[fmt] = names
            })
            if (!cancelled) {
              setDeckLegality(prev => ({
                ...prev,
                [deckID]: {
                  loading: false,
                  unavailable: false,
                  formats: formatMap,
                  cardIllegalByFormat,
                },
              }))
            }
          } catch {
            if (!cancelled) {
              setDeckLegality(prev => ({
                ...prev,
                [deckID]: {
                  loading: false,
                  unavailable: true,
                  formats: {},
                  cardIllegalByFormat: {},
                },
              }))
            }
          }
        }),
      )
    }

    ensureLegality()
    return () => {
      cancelled = true
    }
  }, [paginatedDecks, deckLegality, token])

  function deckToDecklist(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    return cards
      .filter(c => !c.is_sideboard)
      .map(c => `${c.quantity || 1} ${c.card_name || c.name || ''}`.trim())
      .filter(Boolean)
      .join('\n')
  }

  function mainDeckCards(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    return cards.filter(c => !c.is_sideboard)
  }

  function isDeckExpanded(deckID) {
    return Boolean(expandedDecks[deckID])
  }

  function toggleDeckExpanded(deckID) {
    setExpandedDecks(prev => ({
      ...prev,
      [deckID]: !prev[deckID],
    }))
  }

  function parseDecklistToCards(decklist) {
    const rows = []
    const lines = (decklist || '').split('\n')
    for (const raw of lines) {
      const line = raw.trim()
      if (!line || line.startsWith('//')) continue
      const match = line.match(/^(\d+)x?\s+(.+)$/i)
      if (!match) continue
      rows.push({
        card_id: '',
        card_name: match[2].trim(),
        quantity: Math.max(1, Number(match[1]) || 1),
        is_sideboard: false,
        is_commander: false,
      })
    }
    return rows
  }

  async function saveCurrentDeck() {
    setError('')
    
    // Step 1: Validate name and decklist, move to confirm
    if (saveStep === 1) {
      const cards = parseDecklistToCards(editedDecklist)
      if (cards.length === 0) {
        setError(messages.deckEmptyCannotSave)
        return
      }
      if (!name.trim()) {
        setError(messages.deckNameRequired)
        return
      }
      setSaveStep(2)
      return
    }

    // Step 2: Confirm and save
    if (saveStep === 2) {
      if (!canSaveMore) {
        setError(messages.deckLimitReached)
        return
      }
      const cards = parseDecklistToCards(editedDecklist)
      if (cards.length === 0) {
        setError(messages.deckEmptyCannotSave)
        return
      }
      const deckName = (name || '').trim() || `${messages.defaultDeckName} ${new Date().toISOString().slice(0, 10)}`
      try {
        const res = await fetch(`${API}/decks`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: deckName,
            format: currentFormat || 'standard',
            cards,
            is_public: false,
          }),
        })
        const data = await res.json()
        if (!res.ok) throw new Error(data.error || messages.deckSaveFailed)
        setDecks(prev => [data, ...prev])
        setName('')
        setEditedDecklist('')
        setSaveStep(0)
      } catch (err) {
        setError(err.message)
      }
    }
  }

  async function deleteDeck(id) {
    setError('')
    try {
      const res = await fetch(`${API}/decks/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok && res.status !== 204) {
        let msg = messages.deckDeleteFailed
        try {
          const data = await res.json()
          msg = data.error || msg
        } catch {
          // ignore json parse failures
        }
        throw new Error(msg)
      }
      setDecks(prev => prev.filter(d => d.id !== id))
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="card">
      <h2>📚 {messages.deckLibraryTitle}</h2>

      <div className="decklib-meta">
        <div>
          <strong>{messages.currentDeck}</strong>
          <div>{messages.cardsCount(activeSummary.cards)} · {activeSummary.format}</div>
        </div>
        <div>
          <strong>{messages.planLabel}</strong>
          <div>{isPro ? messages.planPro : messages.planFree}</div>
        </div>
        <div>
          <strong>{messages.deckSlots}</strong>
          <div>{isPro ? messages.unlimited : `${decks.length}/1`}</div>
        </div>
      </div>

      {/* Step 0: Show save button */}
      {saveStep === 0 && (
        <div className="decklib-actions">
          <button 
            type="button" 
            className="btn-primary" 
            onClick={() => {
              setError('')
              setName('')
              setEditedDecklist(currentDecklist || '')
              setSaveStep(1)
            }}
            disabled={!canSaveMore}
          >
            {messages.saveDeck}
          </button>
        </div>
      )}

      {/* Step 1: Input deck name and decklist */}
      {saveStep === 1 && (
        <div className="decklib-save-step">
          <h3>{messages.saveDeck}</h3>
          <input
            type="text"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder={messages.deckNamePlaceholder}
            autoFocus
          />
          <textarea
            value={editedDecklist}
            onChange={e => setEditedDecklist(e.target.value)}
            placeholder={messages.decklistHint}
            style={{ maxHeight: '200px', marginTop: '12px' }}
          />
          <div style={{ display: 'flex', gap: '8px', marginTop: '12px' }}>
            <button type="button" className="btn-primary" onClick={saveCurrentDeck}>
              {messages.next || 'Avanti'}
            </button>
            <button 
              type="button" 
              className="btn-ghost" 
              onClick={() => {
                setSaveStep(0)
                setName('')
                setEditedDecklist('')
                setError('')
              }}
            >
              {messages.cancel || 'Annulla'}
            </button>
          </div>
        </div>
      )}

      {/* Step 2: Confirm and show deck preview */}
      {saveStep === 2 && (
        <div className="decklib-save-step">
          <h3>{messages.confirmSaveDeck || 'Conferma salvataggio'}</h3>
          <div style={{ marginBottom: '12px', padding: '12px', background: 'var(--bg-secondary)', borderRadius: '4px' }}>
            <strong>{name}</strong>
            <div style={{ fontSize: '.9rem', color: 'var(--muted)', marginTop: '4px' }}>
              {editedDecklist.split('\n').filter(l => l.trim()).length} {messages.cards} · {currentFormat || 'standard'}
            </div>
          </div>
          <textarea
            readOnly
            value={editedDecklist}
            placeholder={messages.noDecklistProvided}
            style={{ maxHeight: '200px', opacity: 0.7 }}
          />
          <div style={{ display: 'flex', gap: '8px', marginTop: '12px' }}>
            <button type="button" className="btn-primary" onClick={saveCurrentDeck}>
              {messages.confirmSave || 'Salva'}
            </button>
            <button 
              type="button" 
              className="btn-ghost" 
              onClick={() => setSaveStep(1)}
            >
              {messages.back || 'Indietro'}
            </button>
          </div>
        </div>
      )}

      <div className="decklib-actions">
      </div>

      {!isPro && !canSaveMore && (
        <div className="banner banner-warn" style={{ marginBottom: 12 }}>
          {messages.deckLimitReached}
        </div>
      )}

      {error && <div className="banner banner-error">{error}</div>}

      {loading ? (
        <div style={{ color: 'var(--muted)', fontSize: '.9rem' }}>{messages.loading}</div>
      ) : decks.length === 0 ? (
        <div style={{ color: 'var(--muted)', fontSize: '.9rem' }}>{messages.noSavedDecks}</div>
      ) : (
        <>
          <div className="decklib-list">
            {paginatedDecks.map(deck => (
              <div className="decklib-item" key={deck.id}>
                {(() => {
                  const normalizedFormat = (deck.format || 'standard').toLowerCase()
                  const deckLegalityEntry = deckLegality[deck.id]
                  const isLegalityLoading = deckLegalityEntry?.loading === true
                  const isLegalityUnavailable = deckLegalityEntry?.unavailable === true
                  const legality = deckLegalityEntry?.formats?.[normalizedFormat]
                  const formatIsLegal = legality?.is_legal
                  const chipColor = isLegalityUnavailable
                    ? 'var(--orange)'
                    : formatIsLegal === true
                    ? 'var(--green)'
                    : formatIsLegal === false
                      ? 'var(--red)'
                      : 'var(--muted)'
                  const chipText = isLegalityLoading
                    ? messages.loading
                    : isLegalityUnavailable
                      ? (messages.legalityUnavailableShort || 'N/D')
                    : formatIsLegal === true
                      ? messages.legalityLegalLabel
                      : formatIsLegal === false
                        ? messages.legalityIllegalLabel
                        : (messages.unknownLabel || 'N/A')
                  const chipTitle = isLegalityUnavailable
                    ? (messages.legalityUnavailableHint || 'Verifica legalita non disponibile per questo mazzo')
                    : undefined

                  return (
                <div>
                  <div className="decklib-name">{deck.name}</div>
                  <div className="decklib-sub" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{deck.format}</span>
                    <span
                      style={{
                        fontSize: '.68rem',
                        textTransform: 'uppercase',
                        letterSpacing: '.04em',
                        border: `1px solid ${chipColor}`,
                        color: chipColor,
                        borderRadius: 999,
                        padding: '1px 7px',
                        fontWeight: 700,
                      }}
                      title={chipTitle}
                    >
                      {chipText}
                    </span>
                  </div>
                  <div style={{ marginTop: 6, display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
                    {(isDeckExpanded(deck.id) ? mainDeckCards(deck) : mainDeckCards(deck).slice(0, 6)).map((c, idx) => {
                      const label = `${c.quantity || 1}x ${c.card_name || c.name || ''}`.trim()
                      const cardName = (c.card_name || c.name || '').trim()
                      if (!cardName) return null
                      const illegalByFormat = deckLegality[deck.id]?.cardIllegalByFormat?.[normalizedFormat] || {}
                      const isIllegalCard = Boolean(illegalByFormat[cardName.toLowerCase()])
                      return (
                        <span
                          key={`${deck.id}-card-${idx}`}
                          style={{
                            fontSize: '.74rem',
                            color: isIllegalCard ? 'var(--red)' : 'var(--muted)',
                            border: isIllegalCard ? '1px solid var(--red)' : '1px solid var(--border)',
                            borderRadius: 999,
                            padding: '2px 8px',
                            background: isIllegalCard ? 'rgba(255,0,0,0.08)' : 'rgba(255,255,255,0.02)',
                          }}
                          title={isIllegalCard ? `${messages.legalityIllegalLabel} (${normalizedFormat})` : undefined}
                        >
                          <CardHoverPreview cardName={cardName} token={token} messages={messages}>
                            {isIllegalCard ? `⚠ ${label}` : label}
                          </CardHoverPreview>
                        </span>
                      )
                    })}
                    {mainDeckCards(deck).length > 6 && (
                      <button
                        type="button"
                        className="btn-ghost"
                        onClick={() => toggleDeckExpanded(deck.id)}
                        style={{
                          fontSize: '.74rem',
                          lineHeight: 1,
                          padding: '4px 8px',
                          borderRadius: 999,
                        }}
                      >
                        {isDeckExpanded(deck.id)
                          ? (messages.showLessCards || 'Mostra meno')
                          : `+${mainDeckCards(deck).length - 6}`}
                      </button>
                    )}
                  </div>
                </div>
                  )
                })()}
                <div className="decklib-buttons">
                  <button
                    type="button"
                    className="btn-ghost"
                    onClick={() => onSelectDeck?.(deckToDecklist(deck), deck.format || 'standard')}
                  >
                    {messages.useDeck}
                  </button>
                  <button
                    type="button"
                    className="btn-ghost"
                    onClick={() => deleteDeck(deck.id)}
                  >
                    {messages.deleteDeck}
                  </button>
                </div>
              </div>
            ))}
          </div>
          {totalPages > 1 && (
            <div style={{ marginTop: '12px', display: 'flex', gap: '6px', justifyContent: 'center', alignItems: 'center' }}>
              <button
                type="button"
                className="btn-ghost"
                onClick={() => setPage(p => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                ← {messages.previous || 'Precedente'}
              </button>
              <span style={{ fontSize: '.9rem', color: 'var(--muted)' }}>
                {page + 1}/{totalPages}
              </span>
              <button
                type="button"
                className="btn-ghost"
                onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
                disabled={page >= totalPages - 1}
              >
                {messages.next || 'Successivo'} →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

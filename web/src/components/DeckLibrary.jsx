import { useEffect, useMemo, useState } from 'react'
import CardHoverPreview from './CardHoverPreview'
import { apiRequest, throwIfNotOK } from '../lib/apiClient'

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
  const [showSaveComposer, setShowSaveComposer] = useState(false)
  const [editDeckBeforeSave, setEditDeckBeforeSave] = useState(false)
  const [editedDecklist, setEditedDecklist] = useState('')
  const [page, setPage] = useState(0)
  const [totalDecks, setTotalDecks] = useState(0)
  const [reloadNonce, setReloadNonce] = useState(0)
  const [activeDeckId, setActiveDeckId] = useState('')
  const [expandedDecks, setExpandedDecks] = useState({})
  const [deckLegality, setDeckLegality] = useState({})
  const [deckSummaries, setDeckSummaries] = useState({})
  const [cardMetadata, setCardMetadata] = useState({})

  const ITEMS_PER_PAGE = 3
  const paginatedDecks = decks
  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(totalDecks / ITEMS_PER_PAGE)),
    [totalDecks],
  )

  const isPro = (user?.plan || '').toLowerCase() === 'pro'
  const canSaveMore = isPro || totalDecks < 1

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
        const requestPage = page + 1
        const { res, data } = await apiRequest(`/decks?page=${requestPage}&limit=${ITEMS_PER_PAGE}`, { token })
        throwIfNotOK(res, data, messages.deckLoadFailed)
        if (!cancelled) {
          const envelope = Array.isArray(data)
            ? { data, total: data.length, page: requestPage, limit: ITEMS_PER_PAGE }
            : (data || {})
          const pageDecks = Array.isArray(envelope.data) ? envelope.data : []
          const ownedDecks = user?.id
            ? pageDecks.filter(d => d?.user_id === user.id)
            : pageDecks
          const total = Number(envelope.total)
          setDecks(ownedDecks)
          setTotalDecks(Number.isFinite(total) && total >= 0 ? total : ownedDecks.length)
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
  }, [token, messages.deckLoadFailed, page, reloadNonce, user?.id])

  useEffect(() => {
    if (page > 0 && page >= totalPages) {
      setPage(totalPages - 1)
    }
  }, [page, totalPages])

  useEffect(() => {
    if (paginatedDecks.length === 0) {
      setActiveDeckId('')
      return
    }
    const exists = paginatedDecks.some(deck => deck.id === activeDeckId)
    if (!exists) {
      setActiveDeckId(paginatedDecks[0].id)
    }
  }, [paginatedDecks, activeDeckId])

  useEffect(() => {
    let cancelled = false

    async function ensureDeckSummaries() {
      const pending = paginatedDecks
        .filter(deck => deck?.id && !deckSummaries[deck.id])
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
            const { res: summaryRes, data: summaryData } = await apiRequest(`/decks/${deckID}/summary`, { token })
            let formatMap = {}

            if (summaryRes.ok) {
              if (!cancelled) {
                setDeckSummaries(prev => ({
                  ...prev,
                  [deckID]: summaryData,
                }))
              }
              formatMap = summaryData?.legality || {}
            } else {
              const { res, data } = await apiRequest(`/decks/${deckID}/legality`, { token })
              throwIfNotOK(res, data, 'legality fetch failed')
              formatMap = data?.formats || {}
              if (!cancelled) {
                setDeckSummaries(prev => ({
                  ...prev,
                  [deckID]: {
                    deck_id: deckID,
                    legality: formatMap,
                  },
                }))
              }
            }

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

    ensureDeckSummaries()
    return () => {
      cancelled = true
    }
  }, [paginatedDecks, token, deckSummaries])

  useEffect(() => {
    let cancelled = false

    async function loadMetadata() {
      const names = Array.from(new Set(
        paginatedDecks.flatMap(deck => mainDeckCards(deck))
          .map(card => String(card?.card_name || card?.name || '').trim())
          .filter(Boolean),
      ))

      if (!token || names.length === 0) {
        if (!cancelled) setCardMetadata({})
        return
      }

      try {
        const { res, data } = await apiRequest('/cards/metadata/batch', {
          token,
          method: 'POST',
          body: { names },
        })
        throwIfNotOK(res, data, 'metadata fetch failed')

        const next = {}
        for (const item of (data?.items || [])) {
          const key = String(item?.name || '').trim().toLowerCase()
          if (!key) continue
          next[key] = item
        }

        if (!cancelled) setCardMetadata(next)
      } catch {
        if (!cancelled) setCardMetadata({})
      }
    }

    loadMetadata()
    return () => {
      cancelled = true
    }
  }, [paginatedDecks, token])

  function deckToDecklist(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    const mainOrCommander = cards.filter(c => !c.is_sideboard)
    const commanderCards = mainOrCommander.filter(c => c.is_commander)
    const mainCards = mainOrCommander.filter(c => !c.is_commander)

    const formatLine = c => `${c.quantity || 1} ${c.card_name || c.name || ''}`.trim()
    const commanderLines = commanderCards.map(formatLine).filter(Boolean)
    const mainLines = mainCards.map(formatLine).filter(Boolean)

    if (commanderLines.length > 0) {
      return ['Commander', ...commanderLines, '', 'Deck', ...mainLines].join('\n').trim()
    }

    return mainLines.join('\n')
  }

  function mainDeckCards(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    return cards.filter(c => !c.is_sideboard && !c.is_commander)
  }

  function commanderDeckCards(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    return cards.filter(c => !c.is_sideboard && c.is_commander)
  }

  function badgeClassName(value) {
    return String(value || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
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
    function stripTrailingTags(value) {
      let cleaned = String(value || '').trim()
      while (cleaned) {
        const next = cleaned.replace(/\s*\[[^\]]+\]\s*$/g, '').trim()
        if (next === cleaned) break
        cleaned = next
      }
      return cleaned
    }

    function sanitizeCardName(value) {
      let name = stripTrailingTags(value)
      if (!name) return ''
      name = name.replace(/\s*\([A-Za-z0-9]{2,10}\)\s*[A-Za-z0-9-]+\s*$/i, '').trim()
      name = name.replace(/\s*\([A-Za-z0-9]{2,10}\)\s*$/i, '').trim()
      return name.replace(/\s+/g, ' ').trim()
    }

    const rows = []
    const lines = (decklist || '').split('\n')
    for (const raw of lines) {
      const line = raw.trim()
      if (!line || line.startsWith('//')) continue
      const match = line.match(/^(\d+)x?\s+(.+)$/i)
      if (!match) continue
      const isCommander = /\[[^\]]*commander[^\]]*\]/i.test(match[2])
      const normalizedName = sanitizeCardName(match[2])
      if (!normalizedName) continue
      rows.push({
        card_id: '',
        card_name: normalizedName,
        quantity: Math.max(1, Number(match[1]) || 1),
        is_sideboard: false,
        is_commander: isCommander,
      })
    }
    return rows
  }

  async function saveCurrentDeck() {
    setError('')
    if (!canSaveMore) {
      setError(messages.deckLimitReached)
      return
    }
    const cards = parseDecklistToCards(editedDecklist)
    if (cards.length === 0) {
      setError(messages.deckEmptyCannotSave)
      return
    }
    if (!name.trim()) {
      setError(messages.deckNameRequired)
      return
    }
    const deckName = (name || '').trim() || `${messages.defaultDeckName} ${new Date().toISOString().slice(0, 10)}`
    try {
      const { res, data } = await apiRequest('/decks', {
        token,
        method: 'POST',
        body: {
          name: deckName,
          format: currentFormat || 'standard',
          cards,
          is_public: false,
        },
      })
      throwIfNotOK(res, data, messages.deckSaveFailed)
      setTotalDecks(prev => prev + 1)
      setPage(0)
      setReloadNonce(prev => prev + 1)
      setName('')
      setEditedDecklist('')
      setShowSaveComposer(false)
      setEditDeckBeforeSave(false)
    } catch (err) {
      setError(err.message)
    }
  }

  async function deleteDeck(id) {
    setError('')
    try {
      const { res, data } = await apiRequest(`/decks/${id}`, {
        token,
        method: 'DELETE',
      })
      if (!res.ok && res.status !== 204) {
        throw new Error(data?.error || messages.deckDeleteFailed)
      }
      setTotalDecks(prev => Math.max(0, prev - 1))
      setReloadNonce(prev => prev + 1)
      setDeckLegality(prev => {
        const next = { ...prev }
        delete next[id]
        return next
      })
      setDeckSummaries(prev => {
        const next = { ...prev }
        delete next[id]
        return next
      })
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
          <div>{isPro ? messages.unlimited : `${totalDecks}/1`}</div>
        </div>
      </div>

      <div className="decklib-actions">
        {!showSaveComposer ? (
          <button
            type="button"
            className="btn-primary"
            onClick={() => {
              setError('')
              setName('')
              setEditedDecklist(currentDecklist || '')
              setShowSaveComposer(true)
              setEditDeckBeforeSave(false)
            }}
            disabled={!canSaveMore}
          >
            {messages.saveDeck}
          </button>
        ) : (
          <div className="decklib-save-quick">
            <div className="decklib-save-head">
              <h3>{messages.saveDeck}</h3>
              <span className="decklib-sub">
                {messages.cardsCount(activeSummary.cards)} · {activeSummary.format}
              </span>
            </div>
            <div className="decklib-save-row">
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                placeholder={messages.deckNamePlaceholder}
                autoFocus
              />
              <button type="button" className="btn-primary" onClick={saveCurrentDeck}>
                {messages.confirmSave || 'Salva'}
              </button>
              <button
                type="button"
                className="btn-ghost"
                onClick={() => {
                  setShowSaveComposer(false)
                  setName('')
                  setEditedDecklist('')
                  setEditDeckBeforeSave(false)
                  setError('')
                }}
              >
                {messages.cancel || 'Annulla'}
              </button>
            </div>
            <button
              type="button"
              className="btn-ghost decklib-edit-toggle"
              onClick={() => {
                if (!editDeckBeforeSave && !editedDecklist) {
                  setEditedDecklist(currentDecklist || '')
                }
                setEditDeckBeforeSave(prev => !prev)
              }}
            >
              {editDeckBeforeSave
                ? (messages.hideDecklistEditor || 'Nascondi editor lista')
                : (messages.editDeckBeforeSave || 'Modifica lista prima di salvare')}
            </button>
            {editDeckBeforeSave && (
              <textarea
                value={editedDecklist}
                onChange={e => setEditedDecklist(e.target.value)}
                placeholder={messages.decklistHint}
                style={{ maxHeight: '220px' }}
              />
            )}
          </div>
        )}
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
          <div className="decklib-tabs" role="tablist" aria-label={messages.deckLibraryTitle}>
            {paginatedDecks.map(deck => {
              const cards = mainDeckCards(deck)
              const commanderCards = commanderDeckCards(deck)
              const isActive = deck.id === activeDeckId
              return (
                <button
                  key={deck.id}
                  type="button"
                  role="tab"
                  aria-selected={isActive}
                  className={`decklib-tab-btn${isActive ? ' active' : ''}`}
                  onClick={() => setActiveDeckId(deck.id)}
                >
                  <span className="decklib-tab-name">{deck.name}</span>
                  <span className="decklib-tab-sub">{cards.length + commanderCards.length} · {deck.format}</span>
                </button>
              )
            })}
          </div>

          {(() => {
            const activeDeck = paginatedDecks.find(deck => deck.id === activeDeckId) || paginatedDecks[0]
            if (!activeDeck) return null

            const normalizedFormat = (activeDeck.format || 'standard').toLowerCase()
            const deckLegalityEntry = deckLegality[activeDeck.id]
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
            const summary = deckSummaries[activeDeck.id]
            const estimatedUSD = Number(summary?.estimated_usd || 0)
            const commanderBracket = summary?.commander_bracket
            const cards = mainDeckCards(activeDeck)
            const commanderCards = commanderDeckCards(activeDeck)
            const expanded = isDeckExpanded(activeDeck.id)
            const visibleCards = expanded ? cards : cards.slice(0, 14)
            const hiddenCards = Math.max(0, cards.length - visibleCards.length)

            return (
              <div className="decklib-item" role="tabpanel" key={activeDeck.id}>
                <div className="decklib-item-main">
                  <div className="decklib-item-top">
                    <div>
                      <div className="decklib-name">{activeDeck.name}</div>
                      <div className="decklib-sub">{cards.length + commanderCards.length} {messages.cards || 'cards'} · {activeDeck.format}</div>
                      {commanderCards.length > 0 && (
                        <div className="decklib-sub" style={{ marginTop: 4 }}>
                          Commander: {commanderCards.map(c => c.card_name || c.name).filter(Boolean).join(', ')}
                        </div>
                      )}
                    </div>
                    <div className="decklib-chip-row">
                      <span
                        className="decklib-chip"
                        style={{ borderColor: chipColor, color: chipColor }}
                        title={chipTitle}
                      >
                        {chipText}
                      </span>
                      {estimatedUSD > 0 && (
                        <span className="decklib-chip decklib-chip-muted" title="Estimated deck value in USD">
                          {`~$${estimatedUSD.toFixed(2)}`}
                        </span>
                      )}
                      {commanderBracket && (
                        <span
                          className="decklib-chip decklib-chip-bracket"
                          title={`Commander bracket ${commanderBracket.bracket} · ${commanderBracket.label}`}
                        >
                          Bracket {commanderBracket.bracket}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="decklib-item-actions-inline">
                    <button
                      type="button"
                      className="btn-ghost"
                      onClick={() => onSelectDeck?.(deckToDecklist(activeDeck), activeDeck.format || 'standard', activeDeck)}
                    >
                      {messages.useDeck}
                    </button>
                    <button
                      type="button"
                      className="btn-ghost"
                      onClick={() => deleteDeck(activeDeck.id)}
                    >
                      {messages.deleteDeck}
                    </button>
                  </div>

                  <div className="decklib-grid-head" aria-hidden="true">
                    <span>Qty</span>
                    <span>Card</span>
                  </div>
                  <div className="decklib-card-grid">
                    {visibleCards.map((c, idx) => {
                      const quantity = Math.max(1, Number(c.quantity) || 1)
                      const cardName = (c.card_name || c.name || '').trim()
                      if (!cardName) return null
                      const meta = cardMetadata[cardName.toLowerCase()]
                      const illegalByFormat = deckLegality[activeDeck.id]?.cardIllegalByFormat?.[normalizedFormat] || {}
                      const isIllegalCard = Boolean(illegalByFormat[cardName.toLowerCase()])
                      const rarity = String(meta?.rarity || '').trim().toUpperCase()
                      const setCode = String(meta?.set_code || '').trim().toUpperCase()
                      return (
                        <div
                          key={`${activeDeck.id}-card-${idx}`}
                          className={`decklib-card-row${isIllegalCard ? ' is-illegal' : ''}`}
                          title={isIllegalCard ? `${messages.legalityIllegalLabel} (${normalizedFormat})` : undefined}
                        >
                          <span className="decklib-card-qty">{quantity}x</span>
                          <div className="decklib-card-main">
                            <CardHoverPreview cardName={cardName} token={token} messages={messages} metadata={meta}>
                              <span className="decklib-card-name">
                                {isIllegalCard ? `Illegal - ${cardName}` : cardName}
                              </span>
                            </CardHoverPreview>
                            {(rarity || setCode) && (
                              <span className="decklib-card-tags">
                                {rarity && <span className={`builder-badge rarity-${badgeClassName(rarity)}`}>{rarity}</span>}
                                {setCode && <span className="builder-badge builder-badge-set">{setCode}</span>}
                              </span>
                            )}
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  {hiddenCards > 0 && (
                    <button
                      type="button"
                      className="btn-ghost decklib-expand-btn"
                      onClick={() => toggleDeckExpanded(activeDeck.id)}
                    >
                      {expanded
                        ? (messages.showLessCards || 'Mostra meno')
                        : `+${hiddenCards} ${messages.moreCards || 'altre carte'}`}
                    </button>
                  )}
                </div>
              </div>
            )
          })()}
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

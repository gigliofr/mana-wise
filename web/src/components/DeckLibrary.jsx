import { useEffect, useMemo, useState } from 'react'

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
          setDecks(Array.isArray(data) ? data : [])
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

  function deckToDecklist(deck) {
    const cards = Array.isArray(deck?.cards) ? deck.cards : []
    return cards
      .filter(c => !c.is_sideboard)
      .map(c => `${c.quantity || 1} ${c.card_name || ''}`.trim())
      .filter(Boolean)
      .join('\n')
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
    if (!canSaveMore) {
      setError(messages.deckLimitReached)
      return
    }
    const cards = parseDecklistToCards(currentDecklist)
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
    } catch (err) {
      setError(err.message)
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

      <div className="decklib-actions">
        <input
          type="text"
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder={messages.deckNamePlaceholder}
          disabled={!canSaveMore}
        />
        <button type="button" className="btn-primary" onClick={saveCurrentDeck} disabled={!canSaveMore}>
          {messages.saveDeck}
        </button>
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
        <div className="decklib-list">
          {decks.map(deck => (
            <div className="decklib-item" key={deck.id}>
              <div>
                <div className="decklib-name">{deck.name}</div>
                <div className="decklib-sub">{deck.format}</div>
              </div>
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
      )}
    </div>
  )
}

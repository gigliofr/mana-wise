import { useEffect, useState } from 'react'
import { apiRequest, throwIfNotOK } from '../lib/apiClient'

const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']
const ARCHETYPES = ['', 'aggro', 'midrange', 'control', 'combo', 'ramp']

export default function MulliganAssistant({ token, user, decklist: decklistProp, format: formatProp, messages }) {
  const [decklist, setDecklist] = useState(decklistProp || '')
  const [format, setFormat]     = useState(formatProp || 'standard')
  const [savedDecks, setSavedDecks] = useState([])
  const [loadingSavedDecks, setLoadingSavedDecks] = useState(false)
  const [archetype, setArchetype] = useState('')
  const [onPlay, setOnPlay]     = useState(true)
  const [iterations, setIterations] = useState(1000)
  const [loading, setLoading]   = useState(false)
  const [result, setResult]     = useState(null)
  const [error, setError]       = useState('')

  useEffect(() => {
    if (decklistProp !== undefined && decklistProp !== decklist) {
      setDecklist(decklistProp)
    }
  }, [decklistProp])

  useEffect(() => {
    if (formatProp && formatProp !== format) {
      setFormat(formatProp)
    }
  }, [formatProp])

  useEffect(() => {
    let cancelled = false
    async function loadDecks() {
      if (!token) return
      setLoadingSavedDecks(true)
      try {
        const { res, data } = await apiRequest('/decks', { token })
        if (!res.ok || cancelled) return
        const allDecks = Array.isArray(data) ? data : []
        const ownedDecks = user?.id
          ? allDecks.filter(d => d?.user_id === user.id)
          : allDecks
        setSavedDecks(ownedDecks)
      } catch {
        // Keep simulation flow usable even if saved-decks background load fails.
      } finally {
        if (!cancelled) setLoadingSavedDecks(false)
      }
    }
    loadDecks()
    return () => {
      cancelled = true
    }
  }, [token, user?.id])

  async function runSimulation(e) {
    e.preventDefault()
    setError('')
    setResult(null)
    setLoading(true)
    try {
      const payload = {
        decklist,
        format,
        on_play: onPlay,
        iterations: Number(iterations) || 1000,
      }
      if (archetype) payload.archetype = archetype
      const { res, data } = await apiRequest('/mulligan/simulate', {
        token,
        method: 'POST',
        body: payload,
      })
      throwIfNotOK(res, data, messages.mulliganFailed)
      setResult(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <h2>🎴 {messages.mulliganTitle}</h2>

      <form onSubmit={runSimulation}>
        <div className="form-row">
          <label>{messages.decklist}</label>
          <textarea
            value={decklist}
            onChange={e => setDecklist(e.target.value)}
            placeholder={messages.decklistHint}
            required
          />
        </div>

        {savedDecks.length > 0 && (
          <div className="form-row">
            <label>💾 {messages.selectADeck}</label>
            <select onChange={e => {
              const deck = savedDecks.find(d => d.id === e.target.value)
              if (deck) {
                const cards = Array.isArray(deck.cards) ? deck.cards : []
                const formatted = cards.map(c => `${c.quantity || 1} ${c.card_name || c.name || ''}`).join('\n')
                setDecklist(formatted)
                setFormat(deck.format)
                e.target.value = ''
              }
            }} defaultValue="">
              <option value="">
                {loadingSavedDecks ? messages.loading : messages.loadSavedDeck}
              </option>
              {savedDecks.map(d => (
                <option key={d.id} value={d.id}>
                  {d.name} ({d.format})
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="form-row-inline" style={{ gap: 16, flexWrap: 'wrap' }}>
          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 140, marginBottom: 0 }}>
            <label>{messages.format}</label>
            <select value={format} onChange={e => setFormat(e.target.value)}>
              {FORMATS.map(f => <option key={f} value={f}>{f.charAt(0).toUpperCase() + f.slice(1)}</option>)}
            </select>
          </div>

          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 140, marginBottom: 0 }}>
            <label>{messages.archetype} <span style={{ color: 'var(--muted)', fontWeight: 400 }}>({messages.optional})</span></label>
            <select value={archetype} onChange={e => setArchetype(e.target.value)}>
              {ARCHETYPES.map(a => (
                <option key={a} value={a}>{a ? messages.archetypeLabel(a) : messages.autoDetect}</option>
              ))}
            </select>
          </div>

          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 100, marginBottom: 0 }}>
            <label>{messages.iterations}</label>
            <select value={iterations} onChange={e => setIterations(e.target.value)}>
              <option value={500}>500</option>
              <option value={1000}>1 000</option>
              <option value={5000}>5 000</option>
              <option value={10000}>10 000</option>
            </select>
          </div>

          <div style={{ flex: '0 0 auto', display: 'flex', alignItems: 'flex-end', paddingBottom: 4 }}>
            <label className="chip-toggle" style={{ marginBottom: 0 }}>
              <input type="checkbox" checked={onPlay} onChange={e => setOnPlay(e.target.checked)} />
              <span>{messages.onPlay}</span>
            </label>
          </div>
        </div>

        <div style={{ marginTop: 16 }}>
          <button className="btn-primary" type="submit" disabled={loading} style={{ minWidth: 180 }}>
            {loading ? `⚙️ ${messages.simulating}` : `🎴 ${messages.runMulligan}`}
          </button>
        </div>
      </form>

      {error && <div className="banner banner-error" style={{ marginTop: 16 }}>{error}</div>}

      {result && <MulliganResults data={result} messages={messages} />}
    </div>
  )
}

function MulliganResults({ data, messages }) {
  const maxKeep = Math.max(...(data.summaries || []).map(s => s.keep_rate), 0.01)

  return (
    <div className="mulligan-results">
      {data.recommendation && (
        <div className="banner banner-info" style={{ marginBottom: 16 }}>
          <strong>{messages.recommendation}:</strong> {data.recommendation}
        </div>
      )}

      <p className="section-kicker">{messages.handSizeSummaries} ({data.iterations?.toLocaleString()} {messages.iterations})</p>

      <div style={{ overflowX: 'auto' }}>
        <table className="data-table">
          <thead>
            <tr>
              <th>{messages.handSize}</th>
              <th>{messages.keepRate}</th>
              <th></th>
              <th>{messages.avgLands}</th>
              <th>{messages.avgEarlyPlays}</th>
            </tr>
          </thead>
          <tbody>
            {(data.summaries || []).map((s, i) => {
              const barPct = maxKeep > 0 ? (s.keep_rate / maxKeep) * 100 : 0
              const color = s.keep_rate >= 0.6 ? 'var(--green)' : s.keep_rate >= 0.35 ? 'var(--orange)' : 'var(--red)'
              return (
                <tr key={i}>
                  <td style={{ fontWeight: 700, fontSize: '1rem' }}>{s.hand_size}</td>
                  <td style={{ fontWeight: 700, color }}>{Math.round(s.keep_rate * 100)}%</td>
                  <td style={{ width: 120 }}>
                    <div style={{ background: 'var(--border)', borderRadius: 4, height: 8, overflow: 'hidden' }}>
                      <div style={{ width: `${barPct}%`, height: '100%', background: color, borderRadius: 4 }} />
                    </div>
                  </td>
                  <td>{s.avg_lands?.toFixed(1)}</td>
                  <td>{s.avg_early_plays?.toFixed(1)}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

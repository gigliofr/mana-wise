import { useEffect, useState } from 'react'

const API = '/api/v1'
const ARCHETYPES = ['aggro', 'midrange', 'control', 'combo', 'ramp']
const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

export default function MatchupSimulator({ token, decklist: decklistProp, format: formatProp, messages }) {
  const [decklist, setDecklist]     = useState(decklistProp || '')
  const [sideboard, setSideboard]   = useState('')
  const [format, setFormat]         = useState(formatProp || 'standard')
  const [opponents, setOpponents]   = useState(['aggro', 'midrange', 'control'])
  const [onPlay, setOnPlay]         = useState(true)
  const [loading, setLoading]       = useState(false)
  const [result, setResult]         = useState(null)
  const [error, setError]           = useState('')

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
      try {
        const res = await fetch(`${API}/decks`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        const data = await res.json()
        if (res.ok && !cancelled) {
          setSavedDecks(Array.isArray(data) ? data : [])
        }
      } catch (err) {
        // Silently fail
      }
    }
    if (token) loadDecks()
    return () => { cancelled = true }
  }, [token])

  function toggleOpponent(a) {
    setOpponents(prev =>
      prev.includes(a) ? prev.filter(x => x !== a) : [...prev, a],
    )
  }

  async function runSimulation(e) {
    e.preventDefault()
    setError('')
    setResult(null)
    setLoading(true)
    try {
      const res = await fetch(`${API}/matchup/simulate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          decklist,
          sideboard_decklist: sideboard || undefined,
          format,
          opponents,
          on_play: onPlay,
        }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || messages.matchupFailed)
      setResult(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <h2>⚔️ {messages.matchupTitle}</h2>

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
            <label>{messages.loadSavedDeck}</label>
            <select onChange={e => {
              if (!e.target.value) return
              const deck = savedDecks.find(d => d.id === e.target.value)
              if (deck) {
                setDecklist(deck.cards?.map(c => `${c.quantity || 1} ${c.card_name || ''}`).join('\n') || '')
                setFormat(deck.format || 'standard')
                e.target.value = ''
              }
            }}>
              <option value="">{messages.selectADeck}</option>
              {savedDecks.map(d => <option key={d.id} value={d.id}>{d.name} ({d.format})</option>)}
            </select>
          </div>
        )}

        <div className="form-row">
          <label>{messages.sideboardDecklist} <span style={{ color: 'var(--muted)', fontWeight: 400 }}>({messages.optional})</span></label>
          <textarea
            value={sideboard}
            onChange={e => setSideboard(e.target.value)}
            placeholder={messages.sideboardHint}
            rows={4}
          />
        </div>

        <div className="form-row-inline" style={{ gap: 16, flexWrap: 'wrap' }}>
          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 140, marginBottom: 0 }}>
            <label>{messages.format}</label>
            <select value={format} onChange={e => setFormat(e.target.value)}>
              {FORMATS.map(f => <option key={f} value={f}>{f.charAt(0).toUpperCase() + f.slice(1)}</option>)}
            </select>
          </div>

          <div style={{ flex: '1 1 auto' }}>
            <label style={{ display: 'block', marginBottom: 8, fontSize: '.88rem', fontWeight: 600 }}>{messages.opponents}</label>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              {ARCHETYPES.map(a => (
                <label key={a} className="chip-toggle">
                  <input
                    type="checkbox"
                    checked={opponents.includes(a)}
                    onChange={() => toggleOpponent(a)}
                  />
                  <span>{messages.archetypeLabel(a)}</span>
                </label>
              ))}
            </div>
          </div>

          <div style={{ flex: '0 0 auto', display: 'flex', alignItems: 'center', gap: 8 }}>
            <label className="chip-toggle">
              <input type="checkbox" checked={onPlay} onChange={e => setOnPlay(e.target.checked)} />
              <span>{messages.onPlay}</span>
            </label>
          </div>
        </div>

        <div style={{ marginTop: 16 }}>
          <button className="btn-primary" type="submit" disabled={loading || opponents.length === 0} style={{ minWidth: 160 }}>
            {loading ? `⚙️ ${messages.simulating}` : `⚔️ ${messages.runSimulation}`}
          </button>
        </div>
      </form>

      {error && <div className="banner banner-error" style={{ marginTop: 16 }}>{error}</div>}

      {result && <MatchupResults data={result} messages={messages} />}
    </div>
  )
}

function verdictColor(verdict) {
  switch ((verdict || '').toLowerCase()) {
    case 'favored':     return 'var(--green)'
    case 'unfavored':   return 'var(--red)'
    case 'even':        return 'var(--muted)'
    default:            return 'var(--text)'
  }
}

function severityColor(s) {
  if (s === 'significant') return 'var(--red)'
  if (s === 'moderate')    return 'var(--orange)'
  return 'var(--muted)'
}

function MatchupResults({ data, messages }) {
  const mwwr = data.meta_weighted_win_rate ?? 0

  return (
    <div className="matchup-results">
      {/* Summary bar */}
      <div className="matchup-summary">
        <div className="matchup-summary-stat">
          <span>{messages.metaWeightedWR}</span>
          <strong style={{ color: mwwr >= 0.5 ? 'var(--green)' : 'var(--red)' }}>
            {Math.round(mwwr * 100)}%
          </strong>
        </div>
        <div className="matchup-summary-stat">
          <span>{messages.playerArchetype}</span>
          <strong style={{ textTransform: 'capitalize' }}>{data.player_archetype || messages.unknownLabel}</strong>
        </div>
        <div className="matchup-summary-stat">
          <span>{messages.onPlay}</span>
          <strong>{data.on_play ? messages.yes : messages.no}</strong>
        </div>
      </div>

      {data.summary && (
        <p style={{ fontSize: '.9rem', color: 'var(--muted)', marginBottom: 20 }}>{data.summary}</p>
      )}

      {/* Matchups table */}
      {data.matchups?.length > 0 && (
        <>
          <p className="section-kicker">{messages.matchupsTable}</p>
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>{messages.opponent}</th>
                  <th>{messages.metaShare}</th>
                  <th>{messages.winRate}</th>
                  <th>{messages.postBoardWR}</th>
                  <th>{messages.verdict}</th>
                </tr>
              </thead>
              <tbody>
                {data.matchups.map((m, i) => (
                  <tr key={i}>
                    <td style={{ textTransform: 'capitalize', fontWeight: 600 }}>{m.opponent_archetype}</td>
                    <td>{m.meta_share != null ? `${Math.round(m.meta_share * 100)}%` : '—'}</td>
                    <td style={{ color: m.win_rate >= 0.5 ? 'var(--green)' : 'var(--red)', fontWeight: 700 }}>
                      {Math.round(m.win_rate * 100)}%
                    </td>
                    <td>
                      {m.post_board_win_rate
                        ? <span style={{ color: m.post_board_win_rate >= m.win_rate ? 'var(--green)' : 'var(--red)', fontWeight: 600 }}>
                            {Math.round(m.post_board_win_rate * 100)}%
                          </span>
                        : '—'}
                    </td>
                    <td style={{ color: verdictColor(m.verdict), fontWeight: 700, textTransform: 'capitalize' }}>{m.verdict}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Weaknesses */}
      {data.weaknesses?.length > 0 && (
        <>
          <p className="section-kicker" style={{ marginTop: 20 }}>{messages.weaknesses}</p>
          <div className="weakness-list">
            {data.weaknesses.map((w, i) => (
              <div key={i} className="weakness-card">
                <div className="weakness-header">
                  <span style={{ textTransform: 'capitalize', fontWeight: 700 }}>{w.opponent}</span>
                  <span style={{ color: severityColor(w.severity), fontWeight: 600, fontSize: '.82rem', textTransform: 'uppercase' }}>
                    {w.severity}
                  </span>
                  <span style={{ color: 'var(--red)', fontWeight: 700, marginLeft: 'auto' }}>
                    {Math.round(w.win_rate * 100)}%
                  </span>
                </div>
                {w.gaps?.length > 0 && (
                  <ul className="weakness-items">
                    {w.gaps.map((g, j) => <li key={j}>{g}</li>)}
                  </ul>
                )}
                {w.remedies?.length > 0 && (
                  <ul className="weakness-items remedy">
                    {w.remedies.map((r, j) => <li key={j}>{r}</li>)}
                  </ul>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

  const [savedDecks, setSavedDecks] = useState([])

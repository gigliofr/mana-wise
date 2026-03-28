import { useEffect, useState } from 'react'
import CardHoverPreview from './CardHoverPreview'

const API = '/api/v1'
const ARCHETYPES = ['aggro', 'midrange', 'control', 'combo', 'ramp']
const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

export default function SideboardCoach({ token, user, decklist: decklistProp, format: formatProp, messages }) {
  const [mainDecklist, setMainDecklist]   = useState(decklistProp || '')
  const [sideboard, setSideboard]         = useState('')
  const [opponent, setOpponent]           = useState('aggro')
  const [format, setFormat]               = useState(formatProp || 'standard')
  const [loading, setLoading]             = useState(false)
  const [result, setResult]               = useState(null)
  const [error, setError]                 = useState('')
  const [savedDecks, setSavedDecks]       = useState([])
  const [loadingSavedDecks, setLoadingSavedDecks] = useState(false)

  useEffect(() => {
    if (decklistProp !== undefined && decklistProp !== mainDecklist) {
      setMainDecklist(decklistProp)
    }
  }, [decklistProp])

  useEffect(() => {
    if (formatProp && formatProp !== format) {
      setFormat(formatProp)
    }
  }, [formatProp])
  useEffect(() => {
    if (!token) return
    setLoadingSavedDecks(true)
    fetch(`${API}/decks`, { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(decks => {
        const allDecks = Array.isArray(decks) ? decks : []
        const ownedDecks = user?.id
          ? allDecks.filter(d => d?.user_id === user.id)
          : allDecks
        setSavedDecks(ownedDecks)
        setLoadingSavedDecks(false)
      })
      .catch(() => setLoadingSavedDecks(false))
  }, [token, user?.id])

  async function runPlan(e) {
    e.preventDefault()
    setError('')
    setResult(null)
    setLoading(true)
    try {
      const res = await fetch(`${API}/sideboard/plan`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          main_decklist: mainDecklist,
          sideboard_decklist: sideboard,
          opponent_archetype: opponent,
          format,
        }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || messages.sideboardFailed)
      setResult(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <h2>🃏 {messages.sideboardTitle}</h2>

      <form onSubmit={runPlan}>
        <div className="form-row">
          <label>{messages.mainDecklist}</label>
          <textarea
            value={mainDecklist}
            onChange={e => setMainDecklist(e.target.value)}
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
                const formatted = deck.cards.map(c => `${c.quantity || 1} ${c.card_name || c.name || ''}`).join('\n')
                setMainDecklist(formatted)
                setFormat(deck.format)
                e.target.value = ''
              }
            }} defaultValue="">
              <option value="">{loadingSavedDecks ? messages.loading : messages.loadSavedDeck}</option>
              {savedDecks.map(d => (
                <option key={d.id} value={d.id}>
                  {d.name} ({d.format})
                </option>
              ))}
            </select>
          </div>
        )}


        <div className="form-row">
          <label>{messages.sideboardDecklist}</label>
          <textarea
            value={sideboard}
            onChange={e => setSideboard(e.target.value)}
            placeholder={messages.sideboardHint}
            rows={4}
            required
          />
        </div>

        <div className="form-row-inline" style={{ gap: 16, flexWrap: 'wrap' }}>
          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 140, marginBottom: 0 }}>
            <label>{messages.format}</label>
            <select value={format} onChange={e => setFormat(e.target.value)}>
              {FORMATS.map(f => <option key={f} value={f}>{f.charAt(0).toUpperCase() + f.slice(1)}</option>)}
            </select>
          </div>

          <div className="form-row" style={{ flex: '0 0 auto', minWidth: 140, marginBottom: 0 }}>
            <label>{messages.opponentArchetype}</label>
            <select value={opponent} onChange={e => setOpponent(e.target.value)}>
              {ARCHETYPES.map(a => <option key={a} value={a}>{messages.archetypeLabel(a)}</option>)}
            </select>
          </div>
        </div>

        <div style={{ marginTop: 16 }}>
          <button className="btn-primary" type="submit" disabled={loading} style={{ minWidth: 160 }}>
            {loading ? `⚙️ ${messages.planning}` : `🃏 ${messages.runPlan}`}
          </button>
        </div>
      </form>

      {error && <div className="banner banner-error" style={{ marginTop: 16 }}>{error}</div>}

      {result && <SideboardPlanResult data={result} messages={messages} token={token} />}
    </div>
  )
}

function SideboardPlanResult({ data, messages, token }) {
  return (
    <div className="sideboard-result">
      <p className="sideboard-matchup-label">
        {messages.matchupLabel}: <strong style={{ textTransform: 'capitalize' }}>{data.matchup}</strong>
      </p>

      <div className="sideboard-columns">
        {/* Ins */}
        <div className="sideboard-col">
          <p className="section-kicker" style={{ color: 'var(--green)' }}>▲ {messages.sideboardIns}</p>
          {data.ins?.length > 0
            ? <SwapTable swaps={data.ins} messages={messages} token={token} />
            : <p style={{ color: 'var(--muted)', fontSize: '.88rem' }}>—</p>
          }
        </div>

        {/* Outs */}
        <div className="sideboard-col">
          <p className="section-kicker" style={{ color: 'var(--red)' }}>▼ {messages.sideboardOuts}</p>
          {data.outs?.length > 0
            ? <SwapTable swaps={data.outs} messages={messages} token={token} />
            : <p style={{ color: 'var(--muted)', fontSize: '.88rem' }}>—</p>
          }
        </div>
      </div>

      {data.notes?.length > 0 && (
        <div style={{ marginTop: 16 }}>
          <p className="section-kicker">{messages.sideboardNotes}</p>
          <ul className="suggestion-list">
            {data.notes.map((n, i) => <li key={i}>{n}</li>)}
          </ul>
        </div>
      )}
    </div>
  )
}

function SwapTable({ swaps, messages, token }) {
  return (
    <table className="data-table">
      <thead>
        <tr>
          <th>{messages.qty}</th>
          <th>{messages.card}</th>
          <th>{messages.reason}</th>
        </tr>
      </thead>
      <tbody>
        {swaps.map((s, i) => (
          <tr key={i}>
            <td style={{ fontWeight: 700, color: 'var(--accent)' }}>{s.qty}</td>
            <td style={{ fontWeight: 600 }}>
              <CardHoverPreview cardName={s.card} token={token} messages={messages}>
                {s.card}
              </CardHoverPreview>
            </td>
            <td style={{ color: 'var(--muted)', fontSize: '.83rem' }}>{s.reason}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

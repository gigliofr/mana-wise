import { useState } from 'react'

const API = '/api/v1'
const ARCHETYPES = ['aggro', 'midrange', 'control', 'combo', 'ramp']
const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

export default function SideboardCoach({ token, decklist: decklistProp, format: formatProp, messages }) {
  const [mainDecklist, setMainDecklist]   = useState(decklistProp || '')
  const [sideboard, setSideboard]         = useState('')
  const [opponent, setOpponent]           = useState('aggro')
  const [format, setFormat]               = useState(formatProp || 'standard')
  const [loading, setLoading]             = useState(false)
  const [result, setResult]               = useState(null)
  const [error, setError]                 = useState('')

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
              {ARCHETYPES.map(a => <option key={a} value={a}>{a.charAt(0).toUpperCase() + a.slice(1)}</option>)}
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

      {result && <SideboardPlanResult data={result} messages={messages} />}
    </div>
  )
}

function SideboardPlanResult({ data, messages }) {
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
            ? <SwapTable swaps={data.ins} />
            : <p style={{ color: 'var(--muted)', fontSize: '.88rem' }}>—</p>
          }
        </div>

        {/* Outs */}
        <div className="sideboard-col">
          <p className="section-kicker" style={{ color: 'var(--red)' }}>▼ {messages.sideboardOuts}</p>
          {data.outs?.length > 0
            ? <SwapTable swaps={data.outs} />
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

function SwapTable({ swaps }) {
  return (
    <table className="data-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Card</th>
          <th>Reason</th>
        </tr>
      </thead>
      <tbody>
        {swaps.map((s, i) => (
          <tr key={i}>
            <td style={{ fontWeight: 700, color: 'var(--accent)' }}>{s.qty}</td>
            <td style={{ fontWeight: 600 }}>{s.card}</td>
            <td style={{ color: 'var(--muted)', fontSize: '.83rem' }}>{s.reason}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

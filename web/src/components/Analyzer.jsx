import { useEffect, useState } from 'react'
import ManaCurveChart from './ManaCurveChart'
import InteractionPanel from './InteractionPanel'

const API = '/api/v1'
const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

const SAMPLE_DECK = `// Sample Modern Burn deck
4 Lightning Bolt
4 Rift Bolt
4 Lava Spike
4 Shard Volley
4 Goblin Guide
4 Monastery Swiftspear
4 Eidolon of the Great Revel
4 Searing Blaze
4 Searing Blood
4 Skullcrack
4 Light Up the Stage
4 Inspiring Vantage
4 Sacred Foundry
8 Mountain
4 Sunbaked Canyon
`

export default function Analyzer({ token, user, locale, messages, onDeckChange, onFormatChange }) {
  const [decklist, setDecklist] = useState('')
  const [format,   setFormat]   = useState('standard')

  function handleDecklistChange(val) {
    setDecklist(val)
    onDeckChange?.(val)
  }
  function handleFormatChange(val) {
    setFormat(val)
    onFormatChange?.(val)
  }
  const [result,   setResult]   = useState(null)
  const [fingerprint, setFingerprint] = useState(null)
  const [loading,  setLoading]  = useState(false)
  const [error,    setError]    = useState('')
  const [tab,      setTab]      = useState('mana') // 'mana' | 'interaction' | 'fingerprint' | 'ai'
  const [remaining, setRemaining] = useState(typeof user?.remaining === 'number' ? user.remaining : 3)

  useEffect(() => {
    if (typeof user?.remaining === 'number') {
      setRemaining(user.remaining)
    }
  }, [user?.remaining])

  async function onUpgradeClick(e) {
    e.preventDefault()
    try {
      await fetch(`${API}/analytics/upgrade-click`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ source: 'analyzer_banner' }),
      })
    } catch {
      // Best-effort tracking.
    }
    window.location.href = '/upgrade'
  }

  async function analyze(e) {
    e.preventDefault()
    setError('')
    setResult(null)
    setFingerprint(null)
    setLoading(true)
    try {
      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      }

      const [analysisOutcome, fingerprintOutcome] = await Promise.allSettled([
        fetch(`${API}/analyze`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ decklist, format, locale }),
        }),
        fetch(`${API}/deck/classify`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ decklist, format }),
        }),
      ])

      if (analysisOutcome.status !== 'fulfilled') {
        throw new Error(messages.analysisFailed)
      }

      const res = analysisOutcome.value
      const data = await res.json()
      if (!res.ok) {
        if (typeof data?.remaining === 'number') {
          setRemaining(data.remaining)
        }
        throw new Error(data.error || messages.analysisFailed)
      }

      setResult(data)
      setTab('mana')

      if (fingerprintOutcome.status === 'fulfilled') {
        const fingerprintRes = fingerprintOutcome.value
        const fingerprintData = await fingerprintRes.json()
        if (fingerprintRes.ok) {
          setFingerprint(fingerprintData)
        }
      }

      if (isFree) {
        setRemaining(prev => Math.max(0, (typeof prev === 'number' ? prev : 3) - 1))
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const isFree = user?.plan === 'free'

  return (
    <>
      {/* Input panel */}
      <div className="card">
        <h2>🃏 Deck Analyzer</h2>

        {isFree && (
          <div className="banner banner-info" style={{ marginBottom: 16 }}>
            {messages.freePlanBanner(remaining)}
            {' '}<a href="/upgrade" onClick={onUpgradeClick} style={{ color: 'var(--accent)' }}>{messages.upgradeToPro}</a>
          </div>
        )}

        <form onSubmit={analyze}>
          <div className="form-row">
            <label>{messages.decklist} <span style={{ color: 'var(--muted)', fontWeight: 400 }}>({messages.decklistHint})</span></label>
            <textarea
              value={decklist}
              onChange={e => handleDecklistChange(e.target.value)}
              placeholder={SAMPLE_DECK}
              required
            />
          </div>

          <div className="form-row-inline" style={{ alignItems: 'flex-end', gap: 12 }}>
            <div className="form-row" style={{ flex: 1, marginBottom: 0 }}>
              <label>{messages.format}</label>
              <select value={format} onChange={e => handleFormatChange(e.target.value)}>
                {FORMATS.map(f => <option key={f} value={f}>{f.charAt(0).toUpperCase() + f.slice(1)}</option>)}
              </select>
            </div>
            <button className="btn-primary" type="submit" disabled={loading} style={{ whiteSpace: 'nowrap', height: '42px' }}>
              {loading ? `⚙️ ${messages.analyzing}` : `🔍 ${messages.analyzeDeck}`}
            </button>
          </div>
        </form>
      </div>

      {/* Error */}
      {error && <div className="banner banner-error">{error}</div>}

      {/* Results */}
      {result && (
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
            <h2 style={{ marginBottom: 0 }}>📊 {messages.analysisResults}</h2>
            <span className="latency" style={{ marginTop: 0 }}>{messages.analyzedIn(result.latency_ms)}</span>
          </div>

          {/* Quick stats */}
          <div className="stats-grid">
            <div className="stat-item">
              <div className="stat-value">{result.deterministic.mana.total_cards}</div>
              <div className="stat-label">{messages.totalCards}</div>
            </div>
            <div className="stat-item">
              <div className="stat-value">{result.deterministic.mana.average_cmc}</div>
              <div className="stat-label">{messages.avgCmc}</div>
            </div>
            <div className="stat-item">
              <div className="stat-value">{result.deterministic.mana.land_count}</div>
              <div className="stat-label">{messages.landsIdeal(result.deterministic.mana.ideal_land_count)}</div>
            </div>
            <div className="stat-item">
              <div className="stat-value" style={{ color: scoreColor(result.deterministic.interaction.total_score) }}>
                {result.deterministic.interaction.total_score}
              </div>
              <div className="stat-label">{messages.interactionScore}</div>
            </div>
            <div className="stat-item">
              <div className="stat-value" style={{ color: 'var(--primary-h)', textTransform: 'capitalize' }}>
                {result.deterministic.interaction.archetype || 'unknown'}
              </div>
              <div className="stat-label">{messages.detectedArchetype}</div>
            </div>
            {fingerprint && (
              <div className="stat-item">
                <div className="stat-value" style={{ color: 'var(--primary-h)' }}>
                  {fingerprint.color_identity?.length ? fingerprint.color_identity.join('/') : messages.unknownLabel}
                </div>
                <div className="stat-label">{messages.colorIdentity}</div>
              </div>
            )}
          </div>

          <AnalysisLegend result={result} messages={messages} />

          {/* Tabs */}
          <div className="tabs" style={{ marginTop: 24 }}>
            {[
              { key: 'mana',        label: messages.manaCurveTab },
              { key: 'interaction', label: messages.interactionTab },
              { key: 'fingerprint', label: messages.fingerprintTab },
              { key: 'ai',         label: messages.aiTab },
            ].map(t => (
              <button key={t.key} className={`tab-btn${tab === t.key ? ' active' : ''}`} onClick={() => setTab(t.key)}>
                {t.label}
              </button>
            ))}
          </div>

          {tab === 'mana' && <ManaCurvePanel data={result.deterministic.mana} messages={messages} />}
          {tab === 'interaction' && <InteractionPanel data={{ ...result.deterministic.interaction, messages }} />}
          {tab === 'fingerprint' && <FingerprintPanel data={fingerprint} messages={messages} />} 
          {tab === 'ai' && <AIPanel text={result.ai_suggestions} error={result.ai_error} source={result.ai_source} result={result} messages={messages} locale={locale} />}
        </div>
      )}
    </>
  )
}

function scoreColor(score) {
  if (score >= 70) return 'var(--green)'
  if (score >= 40) return 'var(--orange)'
  return 'var(--red)'
}

function ManaCurvePanel({ data, messages }) {
  const maxCount = Math.max(...data.distribution.map(b => b.count), 1)
  const sourceRows = (data.source_requirements || []).filter(r => (r.required || 0) > 0)

  function sourceStatus(row) {
    if (row.gap <= 0) return { label: messages.rowGood, color: 'var(--green)' }
    if (row.gap === 1) return { label: messages.rowPartial, color: 'var(--orange)' }
    return { label: messages.rowLow, color: 'var(--red)' }
  }

  return (
    <div>
      <ManaCurveChart distribution={data.distribution} maxCount={maxCount} messages={messages} />

      {sourceRows.length > 0 && (
        <div style={{ marginTop: 16 }}>
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', marginBottom: 8 }}>{messages.sourceReqTitle}</p>
          <p style={{ fontSize: '.78rem', color: 'var(--muted)', marginBottom: 10 }}>{messages.sourceReqHint}</p>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '.86rem' }}>
              <thead>
                <tr>
                  <th style={thStyle}>{messages.sourceColor}</th>
                  <th style={thStyle}>{messages.sourceCurrent}</th>
                  <th style={thStyle}>{messages.sourceRequired}</th>
                  <th style={thStyle}>{messages.sourceGap}</th>
                  <th style={thStyle}>{messages.sourceStatus}</th>
                </tr>
              </thead>
              <tbody>
                {sourceRows.map((row, idx) => {
                  const status = sourceStatus(row)
                  return (
                    <tr key={`${row.color}-${idx}`}>
                      <td style={tdStyle}>{row.color}</td>
                      <td style={tdStyle}>{row.current}</td>
                      <td style={tdStyle}>{row.required}</td>
                      <td style={{ ...tdStyle, color: row.gap > 0 ? 'var(--red)' : 'var(--green)', fontWeight: 600 }}>{row.gap}</td>
                      <td style={{ ...tdStyle, color: status.color, fontWeight: 600 }}>{status.label}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {data.suggestions?.length > 0 && (
        <>
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', margin: '16px 0 8px' }}>{messages.suggestions}</p>
          <ul className="suggestion-list">
            {data.suggestions.map((s, i) => (
              <li key={i} className={s.urgency}>{s.reason}</li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}

function FingerprintPanel({ data, messages }) {
  if (!data) {
    return (
      <div className="banner banner-warn" style={{ marginBottom: 0 }}>
        {messages.fingerprintUnavailable}
      </div>
    )
  }

  const curveItems = [
    { label: '1', value: data.mana_curve?.one ?? 0 },
    { label: '2', value: data.mana_curve?.two ?? 0 },
    { label: '3', value: data.mana_curve?.three ?? 0 },
    { label: '4', value: data.mana_curve?.four ?? 0 },
    { label: '5+', value: data.mana_curve?.five_plus ?? 0 },
  ]

  return (
    <div className="fingerprint-panel">
      <div className="fingerprint-hero">
        <div>
          <div className="fingerprint-kicker">{messages.deckFingerprint}</div>
          <h3>{messages.fingerprintSummary(data.archetype || messages.unknownLabel, data.confidence ?? 0)}</h3>
          <p>{data.strategy_description || messages.fingerprintUnavailable}</p>
        </div>
        <div className="fingerprint-confidence">
          <span>{messages.confidenceLabel}</span>
          <strong>{Math.round((data.confidence || 0) * 100)}%</strong>
        </div>
      </div>

      <div className="fingerprint-grid">
        <div className="fingerprint-card">
          <div className="fingerprint-card-label">{messages.colorIdentity}</div>
          <div className="fingerprint-chip-row">
            {(data.color_identity?.length ? data.color_identity : [messages.unknownLabel]).map(item => (
              <span key={item} className="fingerprint-chip">{item}</span>
            ))}
          </div>
        </div>

        <div className="fingerprint-card fingerprint-card-wide">
          <div className="fingerprint-card-label">{messages.manaCurveFingerprint}</div>
          <div className="fingerprint-curve-grid">
            {curveItems.map(item => (
              <div key={item.label} className="fingerprint-curve-item">
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

const thStyle = {
  textAlign: 'left',
  borderBottom: '1px solid var(--border)',
  color: 'var(--muted)',
  fontWeight: 600,
  padding: '8px 8px',
}

const tdStyle = {
  borderBottom: '1px solid var(--border)',
  padding: '8px 8px',
}

function AIPanel({ text, error, source, result, messages }) {
  const sourceLabel = source
    ? messages.aiSourceUsed(source)
    : messages.aiSourceUsed(messages.aiSourceUnknown)

  if (!text) {
    const fallbackLines = buildLocalSummary(result, messages)
    return (
      <div>
        <div className="banner banner-warn" style={{ marginBottom: 12 }}>
          <strong>{messages.aiUnavailable}</strong>
          <div style={{ marginTop: 6 }}>{error || messages.aiFallbackNote}</div>
        </div>
        <span className="ai-badge">{messages.localSummaryBadge}</span>
        <div className="ai-source-label">{messages.aiSourceUsed(messages.aiSourceInternal)}</div>
        <div className="ai-box">
          <strong>{messages.localSummaryTitle}</strong>
          <ul className="suggestion-list" style={{ marginTop: 12 }}>
            {fallbackLines.map((line, index) => <li key={index}>{line}</li>)}
          </ul>
        </div>
      </div>
    )
  }
  return (
    <div>
      <span className="ai-badge">{messages.aiBadge}</span>
      <div className="ai-source-label">{sourceLabel}</div>
      <div className="ai-box">{text}</div>
    </div>
  )
}

function AnalysisLegend({ result, messages }) {
  const score = result?.deterministic?.interaction?.total_score ?? 0
  const scoreBand = score >= 70 ? messages.scoreBandGood : score >= 40 ? messages.scoreBandAverage : messages.scoreBandLow
  const archetype = result?.deterministic?.interaction?.archetype || 'unknown'

  return (
    <details className="legend" open>
      <summary>{messages.legendTitle}</summary>
      <div className="legend-grid">
        <div className="legend-item">
          <strong>{messages.legendTotalCardsTitle}</strong>
          <p>{messages.legendTotalCardsBody}</p>
        </div>
        <div className="legend-item">
          <strong>{messages.legendAvgCmcTitle}</strong>
          <p>{messages.legendAvgCmcBody}</p>
        </div>
        <div className="legend-item">
          <strong>{messages.legendLandsTitle}</strong>
          <p>{messages.legendLandsBody(result?.deterministic?.mana?.ideal_land_count)}</p>
        </div>
        <div className="legend-item">
          <strong>{messages.legendInteractionTitle}</strong>
          <p>{messages.legendInteractionBody(scoreBand)}</p>
        </div>
        <div className="legend-item">
          <strong>{messages.legendArchetypeTitle}</strong>
          <p>{messages.legendArchetypeBody(archetype)}</p>
        </div>
        <div className="legend-item">
          <strong>{messages.legendPriorityTitle}</strong>
          <p>{messages.legendPriorityBody}</p>
        </div>
        <div className="legend-item legend-item-wide">
          <strong>{messages.legendArchetypesTitle}</strong>
          <p>{messages.legendArchetypesBody}</p>
          <ul className="legend-list">
            <li>{messages.legendArchetypeAggro}</li>
            <li>{messages.legendArchetypeMidrange}</li>
            <li>{messages.legendArchetypeControl}</li>
            <li>{messages.legendArchetypeRamp}</li>
          </ul>
        </div>
      </div>
    </details>
  )
}

function buildLocalSummary(result, messages) {
  if (!result?.deterministic) {
    return [messages.aiFallbackNote]
  }

  const mana = result.deterministic.mana
  const interaction = result.deterministic.interaction
  const scoreBand = interaction.total_score >= 70
    ? messages.scoreBandGood
    : interaction.total_score >= 40
      ? messages.scoreBandAverage
      : messages.scoreBandLow
  const peakBucket = [...mana.distribution].sort((a, b) => b.count - a.count)[0]
  const peakLabel = peakBucket?.cmc >= 6 ? '6+' : String(peakBucket?.cmc ?? 0)

  return [
    messages.localSummaryLines.archetype(interaction.archetype || 'unknown'),
    messages.localSummaryLines.lands(mana.land_count, mana.ideal_land_count),
    messages.localSummaryLines.cmc(mana.average_cmc),
    messages.localSummaryLines.peak(peakLabel),
    messages.localSummaryLines.interaction(interaction.total_score, scoreBand),
  ]
}

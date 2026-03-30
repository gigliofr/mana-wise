import { Fragment, useEffect, useState } from 'react'
import ManaCurveChart from './ManaCurveChart'
import InteractionPanel from './InteractionPanel'
import { ManaSymbol, ManaSymbolGroup, isManaColorCode } from './ManaSymbol'

const API = '/api/v1'
const FORMATS = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']

const SAMPLE_DECK_STANDARD = `// Sample Modern Burn deck
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

const SAMPLE_DECK_COMMANDER = `// Sample Commander: Meren of Clan Nel Toth
1 Meren of Clan Nel Toth
1 Aatchik, Emerald Radian
1 Accursed Marauder
1 Aftermath Analyst
1 Archfiend of Sorrows
1 Armored Scrapgorger
1 Circle of the Land Druid
1 Erebos, Bleak-Hearted
1 Eternal Witness
1 Fiend Artisan
1 Golgari Grave-Troll
1 Golgari Thug
1 Gravelighter
1 Haywire Mite
1 Honest Rutstein
1 Izoni, Thousand-Eyed
1 Massacre Girl
1 Mirkwood Bats
1 Pharika, God of Affliction
1 Plaguecrafter
1 Priest of Forgotten Gods
1 Reclamation Sage
1 Sakura-Tribe Elder
1 Scavenging Ooze
1 Six
1 Skeleton Crew
1 Skull Prophet
1 Syr Konrad, the Grim
1 Teysa Karlov
1 Void Attendant
1 Ashnod's Altar
1 Skullclamp
1 Battlemage Bracers
1 Liliana, Death's Majesty
1 Vraska, Golgari Queen
1 Bala Ged Recovery
1 Blight Grenade
1 Buried Alive
1 Collective Resistance
1 Golgari Charm
1 Harrow
1 Temporary Lockdown
1 Chalk Outline
1 Chthonian Nightmare
1 Insidious Roots
1 Torment of Hailfire
1 Ash Barrens
1 Barren Moor
1 Command Tower
1 Cryptic Caves
1 Deadwood Enclave
1 Deathcap Glade
1 Duress
1 Evolving Wilds
1 Fetid Heath
1 Golgari Rot Farm
1 Jungle Hollow
1 Mortuary Mire
1 Overgrown Tomb
1 Phantasmal Image
1 Polluted Mire
1 Putrid Swamp
1 Reliquary Tower
1 Revitalize
1 Sandsteppe Citadel
1 Scoured Barrens
1 Suffocating Pit
1 Swamp
1 Swamp
1 Swamp
1 Swamp
1 Swamp
1 Swamp
1 Swamp
1 Swamp
1 Tainted Wood
1 Temple of Malady
1 Temple of the False God
1 The Ozolith
1 Urborg Tomb of Yawgmoth
1 Verdant Catacombs
1 Veinwitch Concoction
1 Wasteland
1 Woodland Cemetery
`

const SAMPLE_DECK = SAMPLE_DECK_STANDARD

export default function Analyzer({ token, user, locale, messages, decklist: decklistProp, format: formatProp, onDeckChange, onFormatChange }) {
  const [decklist, setDecklist] = useState('')
  const [format,   setFormat]   = useState('standard')
  const [savedDecks, setSavedDecks] = useState([])
  const [loadingSavedDecks, setLoadingSavedDecks] = useState(false)

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
    if (decklistProp !== undefined && decklistProp !== decklist) {
      setDecklist(decklistProp)
    }
  }, [decklistProp])

  // Load saved decks
  useEffect(() => {
    let cancelled = false
    async function loadDecks() {
      setLoadingSavedDecks(true)
      try {
        const res = await fetch(`${API}/decks`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        const data = await res.json()
        if (res.ok && !cancelled) {
          const allDecks = Array.isArray(data) ? data : []
          const ownedDecks = user?.id
            ? allDecks.filter(d => d?.user_id === user.id)
            : allDecks
          setSavedDecks(ownedDecks)
        }
      } catch (err) {
        // Silently fail
      } finally {
        if (!cancelled) setLoadingSavedDecks(false)
      }
    }
    if (token) loadDecks()
    return () => { cancelled = true }
  }, [token])

  useEffect(() => {
    if (formatProp && formatProp !== format) {
      setFormat(formatProp)
    }
  }, [formatProp])

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
        'Accept-Language': locale || 'it',
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
        <h2>🃏 {messages.deckAnalyzer}</h2>

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
              placeholder={format === 'commander' ? SAMPLE_DECK_COMMANDER : SAMPLE_DECK_STANDARD}
              required
            />

                    {savedDecks.length > 0 && (
                      <div className="form-row">
                        <label>{messages.loadSavedDeck}</label>
                        <select onChange={e => {
                          if (!e.target.value) return
                          const deck = savedDecks.find(d => d.id === e.target.value)
                          if (deck) {
                            const deckStr = deck.cards?.map(c => `${c.quantity || 1} ${c.card_name || c.name || ''}`).join('\n') || ''
                            handleDecklistChange(deckStr)
                            handleFormatChange(deck.format || 'standard')
                            e.target.value = ''
                          }
                        }}>
                          <option value="">{messages.selectADeck}</option>
                          {savedDecks.map(d => (
                            <option key={d.id} value={d.id}>{d.name} ({d.format})</option>
                          ))}
                        </select>
                      </div>
                    )}
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
                  {fingerprint.color_identity?.length
                    ? <ManaSymbolGroup colors={fingerprint.color_identity} size={20} gap={4} />
                    : messages.unknownLabel}
                </div>
                <div className="stat-label">{messages.colorIdentity}</div>
              </div>
            )}
          </div>

          <AnalysisLegend result={result} messages={messages} />
          <LegalityLegend legality={result.legality} messages={messages} />

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

function renderManaSymbolsInText(text, size = 14) {
  const raw = String(text || '')
  if (!raw) return raw

  const symbolPattern = /(\{?[WUBRGC]\}?)(?=[^A-Za-z]|$)/g
  const parts = raw.split(symbolPattern)

  return parts.map((part, idx) => {
    const normalized = part.replace(/[{}]/g, '').toUpperCase()
    if (isManaColorCode(normalized)) {
      return <ManaSymbol key={`mana-${normalized}-${idx}`} code={normalized} size={size} />
    }
    return <span key={`txt-${idx}`}>{part}</span>
  })
}

function ManaCurvePanel({ data, messages }) {
  const maxCount = Math.max(...data.distribution.map(b => b.count), 1)
  const totalSourceRow = {
    label: messages.sourceTotalLabel,
    current: data.current_total_sources ?? data.land_count ?? 0,
    required: data.target_total_sources ?? data.ideal_land_count ?? 0,
    gap: data.total_source_gap ?? ((data.target_total_sources ?? data.ideal_land_count ?? 0) - (data.current_total_sources ?? data.land_count ?? 0)),
  }
  const consistencyCards = [
    {
      key: 'screw',
      label: messages.manaScrewLabel,
      value: data.mana_screw_chance ?? 0,
      tone: 'var(--red)',
      description: messages.landConsistencyScrew((data.mana_screw_chance ?? 0).toFixed(1)),
    },
    {
      key: 'flood',
      label: messages.manaFloodLabel,
      value: data.mana_flood_chance ?? 0,
      tone: 'var(--orange)',
      description: messages.landConsistencyFlood((data.mana_flood_chance ?? 0).toFixed(1)),
    },
    {
      key: 'sweet',
      label: messages.sweetSpotLabel,
      value: data.sweet_spot_chance ?? 0,
      tone: 'var(--green)',
      description: messages.landConsistencySweet((data.sweet_spot_chance ?? 0).toFixed(1)),
    },
  ]

  function sourceStatus(row) {
    if (row.gap <= 0) return { label: messages.rowGood, color: 'var(--green)' }
    if (row.gap === 1) return { label: messages.rowPartial, color: 'var(--orange)' }
    return { label: messages.rowLow, color: 'var(--red)' }
  }

  return (
    <div>
      <ManaCurveChart distribution={data.distribution} maxCount={maxCount} messages={messages} />

      {(data.land_sample_draws || 0) > 0 && (
        <div style={{ marginTop: 16 }}>
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', marginBottom: 8 }}>{messages.landConsistencyTitle}</p>
          <p style={{ fontSize: '.78rem', color: 'var(--muted)', marginBottom: 10 }}>
            {messages.landConsistencyHint(data.land_sample_draws, data.sweet_spot_min_lands, data.sweet_spot_max_lands)}
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 10 }}>
            {consistencyCards.map(card => (
              <div key={card.key} style={{ border: '1px solid var(--border)', borderRadius: 10, padding: 10, background: 'rgba(255,255,255,0.02)' }}>
                <div style={{ fontSize: '.8rem', color: 'var(--muted)' }}>{card.label}</div>
                <div style={{ fontSize: '1.1rem', fontWeight: 700, color: card.tone, marginTop: 4 }}>{card.value.toFixed(1)}%</div>
                <div style={{ marginTop: 6, fontSize: '.76rem', color: 'var(--muted)' }}>{card.description}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {(totalSourceRow.required || 0) > 0 && (
        <div style={{ marginTop: 16 }}>
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', marginBottom: 8 }}>{messages.sourceReqTitle}</p>
          <p style={{ fontSize: '.78rem', color: 'var(--muted)', marginBottom: 10 }}>{messages.sourceReqHint}</p>
          <p style={{ fontSize: '.78rem', color: 'var(--muted)', marginBottom: 10 }}>
            {messages.sourceReqCountedLands(data.land_count, data.mana_producer_count || 0, data.ideal_land_count)}
          </p>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '.86rem' }}>
              <thead>
                <tr>
                  <th style={thStyle}>{messages.sourceMetric}</th>
                  <th style={thStyle}>{messages.sourceCurrent}</th>
                  <th style={thStyle}>{messages.sourceRequired}</th>
                  <th style={thStyle}>{messages.sourceGap}</th>
                  <th style={thStyle}>{messages.sourceStatus}</th>
                </tr>
              </thead>
              <tbody>
                {(() => {
                  const status = sourceStatus(totalSourceRow)
                  return (
                    <tr>
                      <td style={tdStyle}>{totalSourceRow.label}</td>
                      <td style={tdStyle}>{totalSourceRow.current}</td>
                      <td style={tdStyle}>{totalSourceRow.required}</td>
                      <td style={{ ...tdStyle, color: totalSourceRow.gap > 0 ? 'var(--red)' : 'var(--green)', fontWeight: 600 }}>{totalSourceRow.gap}</td>
                      <td style={{ ...tdStyle, color: status.color, fontWeight: 600 }}>{status.label}</td>
                    </tr>
                  )
                })()}
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
              <li key={i} className={s.urgency}>
                {renderManaSymbolsInText(s.reason, 14)}
              </li>
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
              <span key={item} className="fingerprint-chip" style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>
                {isManaColorCode(item) ? <ManaSymbol code={item} size={18} /> : item}
              </span>
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
            {fallbackLines.map((line, index) => <li key={index}>{renderManaSymbolsInText(line, 14)}</li>)}
          </ul>
        </div>
      </div>
    )
  }

  const aiTextLines = String(text || '').split('\n')
  return (
    <div>
      <span className="ai-badge">{messages.aiBadge}</span>
      <div className="ai-source-label">{sourceLabel}</div>
      <div className="ai-box">
        {aiTextLines.map((line, idx) => (
          <div key={`ai-line-${idx}`}>{renderManaSymbolsInText(line, 14)}</div>
        ))}
      </div>
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

function LegalityLegend({ legality, messages }) {
  const [expandedFormat, setExpandedFormat] = useState(null)
  const rows = ['standard', 'pioneer', 'modern', 'legacy', 'vintage', 'commander', 'pauper']
    .map(format => ({ format, data: legality?.[format] }))
    .filter(item => item.data)

  if (rows.length === 0) return null

  return (
    <details className="legend" style={{ marginTop: 12 }}>
      <summary>{messages.legalityLegendTitle}</summary>
      <p style={{ color: 'var(--muted)', fontSize: '.84rem', margin: '8px 0 12px' }}>
        {messages.legalityLegendBody}
      </p>

      <div className="legality-key">
        <span className="legality-chip legal">{messages.legalityLegalLabel}</span>
        <span className="legality-chip illegal">{messages.legalityIllegalLabel}</span>
      </div>

      <div style={{ overflowX: 'auto', marginTop: 10 }}>
        <table className="data-table">
          <thead>
            <tr>
              <th>{messages.format}</th>
              <th>{messages.verdict}</th>
              <th>{messages.totalCards}</th>
              <th>{messages.legalityIssuesLabel}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(({ format, data }) => {
              const issueCount = (data.issues?.length || 0) + (data.illegal_cards?.length || 0)
              const isExpanded = expandedFormat === format
              return (
                <Fragment key={format}>
                  <tr>
                    <td style={{ textTransform: 'capitalize', fontWeight: 600 }}>{format}</td>
                    <td>
                      <span className={`legality-chip ${data.is_legal ? 'legal' : 'illegal'}`}>
                        {data.is_legal ? messages.legalityLegalLabel : messages.legalityIllegalLabel}
                      </span>
                    </td>
                    <td>{data.deck_size}</td>
                    <td>
                      {issueCount > 0 ? (
                        <button
                          type="button"
                          className="tab-btn"
                          style={{ padding: '2px 10px', fontSize: '.8rem', borderBottom: 'none' }}
                          onClick={() => setExpandedFormat(isExpanded ? null : format)}
                        >
                          {issueCount} {isExpanded ? messages.hideDetailsLabel : messages.showDetailsLabel}
                        </button>
                      ) : (
                        issueCount
                      )}
                    </td>
                  </tr>
                  {isExpanded && issueCount > 0 && (
                    <tr>
                      <td colSpan={4} style={{ padding: '10px 12px', background: 'rgba(255,255,255,0.02)' }}>
                        <div style={{ fontSize: '.82rem', color: 'var(--muted)' }}>
                          {data.issues?.length > 0 && (
                            <div style={{ marginBottom: 8 }}>
                              <strong style={{ color: 'var(--text)' }}>{messages.legalityGeneralIssuesLabel}</strong>
                              <ul style={{ margin: '6px 0 0 18px' }}>
                                {data.issues.map((issue, idx) => (
                                  <li key={`issue-${format}-${idx}`}>{issue}</li>
                                ))}
                              </ul>
                            </div>
                          )}

                          {data.illegal_cards?.length > 0 && (
                            <div>
                              <strong style={{ color: 'var(--text)' }}>{messages.legalityCardIssuesLabel}</strong>
                              <ul style={{ margin: '6px 0 0 18px' }}>
                                {data.illegal_cards.map((item, idx) => (
                                  <li key={`illegal-${format}-${idx}`}>
                                    {item.card_name} x{item.quantity}: {item.reason}
                                  </li>
                                ))}
                              </ul>
                            </div>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              )
            })}
          </tbody>
        </table>
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
  const categories = interaction?.categories || []
  const scoreBand = interaction.total_score >= 70
    ? messages.scoreBandGood
    : interaction.total_score >= 40
      ? messages.scoreBandAverage
      : messages.scoreBandLow
  const peakBucket = [...mana.distribution].sort((a, b) => b.count - a.count)[0]
  const peakLabel = peakBucket?.cmc >= 6 ? '6+' : String(peakBucket?.cmc ?? 0)
  const deckArchetype = interaction.archetype || 'unknown'
  const colorCount = (mana.color_requirements || []).filter(req => (req?.required_sources || 0) > 0).length
  const colorSpeedRisk = getColorSpeedRisk(deckArchetype, colorCount)
  const [landMin, landMax] = getArchetypeLandRange(deckArchetype, mana.total_cards || 60, mana.ideal_land_count || mana.land_count)
  const topDeficit = [...categories]
    .map(cat => ({ ...cat, deficit: (cat.ideal || 0) - (cat.count || 0) }))
    .filter(cat => cat.deficit > 0)
    .sort((a, b) => b.deficit - a.deficit)[0]
  const focusCategoryLabel = topDeficit?.category
    ? (messages.categoryLabels?.[topDeficit.category] || topDeficit.category)
    : null

  return [
    messages.localSummaryLines.archetype(deckArchetype),
    messages.localSummaryLines.colorSpeed(deckArchetype, colorCount, colorSpeedRisk),
    messages.localSummaryLines.landRange(mana.land_count, landMin, landMax),
    messages.localSummaryLines.lands(mana.land_count, mana.ideal_land_count),
    messages.localSummaryLines.consistency(mana.mana_screw_chance || 0, mana.mana_flood_chance || 0, mana.sweet_spot_chance || 0),
    messages.localSummaryLines.cmc(mana.average_cmc),
    messages.localSummaryLines.peak(peakLabel),
    messages.localSummaryLines.interaction(interaction.total_score, scoreBand),
    focusCategoryLabel
      ? messages.localSummaryLines.topGap(focusCategoryLabel, topDeficit.deficit)
      : messages.localSummaryLines.topGapNone(),
    messages.localSummaryLines.playtestingLoop(),
  ]
}

function getColorSpeedRisk(archetype, colorCount) {
  if (archetype === 'aggro') {
    if (colorCount >= 3) return 'high'
    if (colorCount === 2) return 'medium'
    return 'low'
  }
  if (archetype === 'combo') {
    if (colorCount >= 4) return 'high'
    if (colorCount === 3) return 'medium'
    return 'low'
  }
  if (archetype === 'midrange') {
    if (colorCount >= 4) return 'high'
    if (colorCount === 3) return 'medium'
    return 'low'
  }
  if (archetype === 'control' || archetype === 'ramp') {
    if (colorCount >= 5) return 'high'
    if (colorCount === 4) return 'medium'
    return 'low'
  }
  if (colorCount >= 4) return 'medium'
  return 'low'
}

function getArchetypeLandRange(archetype, totalCards, idealLandCount) {
  if (totalCards >= 80) {
    if (archetype === 'aggro') return [34, 37]
    if (archetype === 'control') return [36, 39]
    if (archetype === 'midrange') return [35, 38]
    if (archetype === 'ramp') return [37, 40]
    if (archetype === 'combo') return [34, 38]
    return [Math.max(32, idealLandCount - 1), Math.max(34, idealLandCount + 2)]
  }

  if (archetype === 'aggro') return [20, 24]
  if (archetype === 'control') return [26, 28]
  if (archetype === 'midrange') return [24, 26]
  if (archetype === 'ramp') return [25, 28]
  if (archetype === 'combo') return [22, 25]
  return [Math.max(20, idealLandCount - 1), idealLandCount + 1]
}

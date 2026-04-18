import { Fragment, useEffect, useState } from 'react'
import ManaCurveChart from './ManaCurveChart'
import InteractionPanel from './InteractionPanel'
import CardHoverPreview from './CardHoverPreview'
import { ManaSymbol, ManaSymbolGroup, isManaColorCode } from './ManaSymbol'
import { apiRequest } from '../lib/apiClient'

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

export default function Analyzer({ token, user, locale, messages, decklist: decklistProp, format: formatProp, onDeckChange, onFormatChange, onUpgradeRequest }) {
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
  const [commanderScore, setCommanderScore] = useState(null)
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
        const { res, data } = await apiRequest('/decks', { token })
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
      await apiRequest('/analytics/upgrade-click', {
        token,
        method: 'POST',
        body: { source: 'analyzer_banner' },
      })
    } catch {
      // Best-effort tracking.
    }
    if (typeof onUpgradeRequest === 'function') {
      onUpgradeRequest('analyzer_banner')
      return
    }
    window.location.href = '/?tool=plans'
  }

  async function analyze(e) {
    e.preventDefault()
    setError('')
    setResult(null)
    setFingerprint(null)
    setCommanderScore(null)
    setLoading(true)
    try {
      const commanderMode = format === 'commander'
      const requests = [
        apiRequest('/analyze', {
          token,
          method: 'POST',
          headers: { 'Accept-Language': locale || 'it' },
          body: { decklist, format, locale },
        }),
        apiRequest('/deck/classify', {
          token,
          method: 'POST',
          headers: { 'Accept-Language': locale || 'it' },
          body: { decklist, format },
        }),
      ]
      if (commanderMode) {
        requests.push(apiRequest('/score', {
          token,
          method: 'POST',
          headers: { 'Accept-Language': locale || 'it' },
          body: { decklist, format },
        }))
      }

      const outcomes = await Promise.allSettled(requests)
      const analysisOutcome = outcomes[0]
      const fingerprintOutcome = outcomes[1]
      const scoreOutcome = outcomes[2]

      if (analysisOutcome.status !== 'fulfilled') {
        const reason = analysisOutcome.reason
        const reasonMessage = reason instanceof Error
          ? reason.message
          : typeof reason === 'string'
            ? reason
            : ''
        throw new Error(reasonMessage || messages.analysisFailed)
      }

      const { res, data } = analysisOutcome.value
      if (!res.ok) {
        if (typeof data?.remaining === 'number') {
          setRemaining(data.remaining)
        }
        throw new Error(data?.error || messages.analysisFailed)
      }

      setResult(data)
      setTab('mana')

      if (fingerprintOutcome.status === 'fulfilled') {
        const { res: fingerprintRes, data: fingerprintData } = fingerprintOutcome.value
        if (fingerprintRes.ok) {
          setFingerprint(fingerprintData)
        }
      }

      if (scoreOutcome?.status === 'fulfilled') {
        const { res: scoreRes, data: scoreData } = scoreOutcome.value
        if (scoreRes.ok && typeof scoreData?.score_detail?.score === 'number') {
          setCommanderScore(scoreData.score_detail.score)
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
                            const cards = Array.isArray(deck.cards) ? deck.cards : []
                            const deckStr = cards.map(c => `${c.quantity || 1} ${c.card_name || c.name || ''}`).join('\n')
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

      {!error && result?.warnings?.length > 0 && (
        <div className="banner banner-info" style={{ marginTop: 16 }}>
          {result.warnings.join(' ')}
        </div>
      )}

      {/* Results */}
      {result && (
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
            <h2 style={{ marginBottom: 0 }}>📊 {messages.analysisResults}</h2>
            <span className="latency" style={{ marginTop: 0 }}>{messages.analyzedIn(result.latency_ms)}</span>
          </div>

          {result.commander?.cards?.length > 0 && (
            <div className="banner banner-info" style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
              <strong>{messages.commanderSectionTitle || 'Commander'}</strong>
              <span>{result.commander.cards.map(card => card.name).join(' + ')}</span>
              {typeof commanderScore === 'number' && (
                <span>
                  {messages.commanderBracketLabel || 'Bracket'} {commanderBracketForScore(commanderScore).bracket} · {commanderBracketForScore(commanderScore).label} · {commanderScore.toFixed(1)}/10
                </span>
              )}
            </div>
          )}

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

          {result.commander?.cards?.length > 0 && (
            <div className="card" style={{ marginTop: 24 }}>
              <h3 style={{ marginBottom: 12 }}>{messages.commanderSectionTitle || 'Commander'}</h3>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {result.commander.cards.map(card => (
                  <CardHoverPreview
                    key={card.id}
                    cardName={card.name}
                    token={token}
                    messages={messages}
                    metadata={{ rarity: card.rarity, set_code: card.set_code, collector_number: card.collector_number }}
                  >
                    <span className="builder-badge" style={{ fontSize: '.78rem' }}>{card.name}</span>
                  </CardHoverPreview>
                ))}
              </div>
            </div>
          )}

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

          {tab === 'mana' && <ManaCurvePanel data={result.deterministic.mana} detectedArchetype={result.deterministic.interaction?.archetype} fingerprint={fingerprint} decklist={decklist} messages={messages} />}
          {tab === 'interaction' && <InteractionPanel data={{ ...result.deterministic.interaction, messages }} />}
          {tab === 'fingerprint' && <FingerprintPanel data={fingerprint} messages={messages} />} 
          {tab === 'ai' && <AIPanel text={result.ai_suggestions} error={result.ai_error} source={result.ai_source} result={result} commanderScore={commanderScore} messages={messages} locale={locale} />}
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

function commanderBracketForScore(score) {
  if (score >= 8.5) return { bracket: 5, label: 'cEDH' }
  if (score >= 6.5) return { bracket: 4, label: 'Optimized' }
  if (score >= 4.5) return { bracket: 3, label: 'Tuned' }
  if (score >= 2.5) return { bracket: 2, label: 'Upgraded' }
  return { bracket: 1, label: 'Casual' }
}

function renderManaSymbolsInText(text, size = 14) {
  const raw = String(text || '')
  if (!raw) return raw

  const symbolPattern = /(\{[WUBRGC]\}|(?<![A-Za-z])[WUBRGC](?![A-Za-z]))/g
  const parts = raw.split(symbolPattern)

  return parts.map((part, idx) => {
    const normalized = part.replace(/[{}]/g, '').toUpperCase()
    if (isManaColorCode(normalized) && (part.startsWith('{') || part.length === 1)) {
      return <ManaSymbol key={`mana-${normalized}-${idx}`} code={normalized} size={size} />
    }
    return <span key={`txt-${idx}`}>{part}</span>
  })
}

const MOCK_ARCHETYPE_PROFILES = {
  aggro: { idealCmc: 1.8, idealLandRatio: 0.40, creature12: 78, fastSpells: 65, curveLe2: 80, finisher4: 22 },
  midrange: { idealCmc: 2.9, idealLandRatio: 0.42, creature12: 55, fastSpells: 52, curveLe2: 60, finisher4: 35 },
  control: { idealCmc: 3.2, idealLandRatio: 0.45, creature12: 35, fastSpells: 58, curveLe2: 45, finisher4: 42 },
  combo: { idealCmc: 2.4, idealLandRatio: 0.41, creature12: 42, fastSpells: 68, curveLe2: 62, finisher4: 28 },
  ramp: { idealCmc: 3.3, idealLandRatio: 0.44, creature12: 38, fastSpells: 48, curveLe2: 46, finisher4: 48 },
}

const ARCHETYPE_CURVE_TARGETS = {
  aggro: [8, 28, 34, 18, 8, 3, 1],
  midrange: [4, 18, 30, 24, 14, 7, 3],
  control: [3, 12, 24, 26, 20, 10, 5],
  combo: [6, 22, 31, 21, 12, 6, 2],
  ramp: [3, 12, 20, 24, 22, 12, 7],
}

const CMC_BUCKET_LABELS = ['0', '1', '2', '3', '4', '5', '6+']

const KNOWN_LAND_TERMS = ['plains', 'island', 'swamp', 'mountain', 'forest', 'wastes', 'tower', 'temple', 'cavern', 'sanctuary', 'tomb', 'wilds', 'fetch', 'marsh', 'shore', 'garden', 'catacomb', 'citadel']

function normalizeMockArchetype(archetype) {
  const key = String(archetype || '').toLowerCase().trim()
  return MOCK_ARCHETYPE_PROFILES[key] ? key : 'aggro'
}

function clampPercent(value) {
  return Math.max(0, Math.min(100, value))
}

function scoreByTarget(value, target) {
  if (value >= target + 10) return 'great'
  if (value >= target) return 'good'
  return 'low'
}

function calcMetaMatch(profile, avgCmc, landRatio, curveLe2, finisher4) {
  const cmcFit = Math.max(0, 100 - Math.abs(avgCmc - profile.idealCmc) * 35)
  const landFit = Math.max(0, 100 - Math.abs(landRatio - profile.idealLandRatio) * 220)
  const curveFit = Math.max(0, 100 - Math.abs(curveLe2 - profile.curveLe2) * 1.8)
  const finisherFit = Math.max(0, 100 - Math.abs(finisher4 - profile.finisher4) * 1.8)
  return Math.round((cmcFit + landFit + curveFit + finisherFit) / 4)
}

function getCurveShares(distribution, nonLandCards) {
  return (distribution || []).map(bucket => {
    if (nonLandCards <= 0) return 0
    return ((bucket.count || 0) / nonLandCards) * 100
  })
}

function calcCurveBenchmark(archetypeKey, distribution, nonLandCards) {
  const target = ARCHETYPE_CURVE_TARGETS[archetypeKey] || ARCHETYPE_CURVE_TARGETS.aggro
  const actual = getCurveShares(distribution, nonLandCards)
  const deltas = target.map((t, idx) => ({
    idx,
    label: CMC_BUCKET_LABELS[idx] || String(idx),
    target: t,
    actual: Math.round((actual[idx] || 0) * 10) / 10,
    delta: Math.round(((actual[idx] || 0) - t) * 10) / 10,
  }))
  const meanAbs = deltas.reduce((sum, row) => sum + Math.abs(row.delta), 0) / Math.max(1, deltas.length)
  const score = Math.max(0, Math.round(100 - (meanAbs * 3.2)))
  return { score, deltas }
}

function buildCurveDeltaSuggestions(benchmark, messages) {
  const rows = benchmark?.deltas || []
  if (rows.length === 0) return []

  const over = [...rows].sort((a, b) => b.delta - a.delta).filter(r => r.delta >= 6).slice(0, 2)
  const under = [...rows].sort((a, b) => a.delta - b.delta).filter(r => r.delta <= -6).slice(0, 2)
  const out = []

  over.forEach(row => {
    out.push(messages.manaMockCurveCutSuggestion(row.label, row.delta.toFixed(1), row.target.toFixed(0), row.actual.toFixed(1)))
  })
  under.forEach(row => {
    out.push(messages.manaMockCurveAddSuggestion(row.label, Math.abs(row.delta).toFixed(1), row.target.toFixed(0), row.actual.toFixed(1)))
  })

  return out.slice(0, 4)
}

function estimateTypeDistribution(nonLandCards, archetypeKey) {
  const shares = {
    aggro: { creature: 0.52, spell: 0.33, enchantArtifact: 0.10, planeswalker: 0.05 },
    midrange: { creature: 0.45, spell: 0.30, enchantArtifact: 0.15, planeswalker: 0.10 },
    control: { creature: 0.18, spell: 0.54, enchantArtifact: 0.16, planeswalker: 0.12 },
    combo: { creature: 0.30, spell: 0.48, enchantArtifact: 0.17, planeswalker: 0.05 },
    ramp: { creature: 0.33, spell: 0.34, enchantArtifact: 0.18, planeswalker: 0.15 },
  }[archetypeKey] || { creature: 0.40, spell: 0.38, enchantArtifact: 0.14, planeswalker: 0.08 }

  const creature = Math.round(nonLandCards * shares.creature)
  const spell = Math.round(nonLandCards * shares.spell)
  const enchantArtifact = Math.round(nonLandCards * shares.enchantArtifact)
  const planeswalker = Math.max(0, nonLandCards - creature - spell - enchantArtifact)
  return { creature, spell, enchantArtifact, planeswalker }
}

function parseDeckPool(decklistText) {
  const pool = []
  const lines = String(decklistText || '').split(/\r?\n/)
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('//')) continue
    const match = trimmed.match(/^(\d+)\s+(.+)$/)
    if (!match) continue
    const qty = Number.parseInt(match[1], 10)
    const name = match[2].trim()
    if (!name || Number.isNaN(qty) || qty <= 0) continue
    const lower = name.toLowerCase()
    const isLand = KNOWN_LAND_TERMS.some(term => lower.includes(term))
    for (let i = 0; i < qty; i++) {
      pool.push({ name, isLand })
    }
  }
  return pool
}

function randomHand(pool, size) {
  if (!Array.isArray(pool) || pool.length === 0 || size <= 0) return []
  const copy = [...pool]
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    const tmp = copy[i]
    copy[i] = copy[j]
    copy[j] = tmp
  }
  return copy.slice(0, Math.min(size, copy.length))
}

function evaluateOpeningHand(hand, archetypeKey, messages) {
  const lands = hand.filter(card => card.isLand).length
  const limits = {
    aggro: { min: 2, max: 4 },
    midrange: { min: 2, max: 4 },
    control: { min: 2, max: 5 },
    combo: { min: 2, max: 4 },
    ramp: { min: 3, max: 5 },
  }[archetypeKey] || { min: 2, max: 4 }

  if (lands < limits.min || lands > limits.max) {
    return {
      tone: 'var(--orange)',
      text: messages.manaMockMulliganAdvice(lands, limits.min, limits.max),
      shouldMulligan: true,
    }
  }

  return {
    tone: 'var(--green)',
    text: messages.manaMockKeepAdvice(lands),
    shouldMulligan: false,
  }
}

// London Mulligan: Draw N cards, optionally mulligan to N-1, scry 1
function londonMulliganRedraw(pool, currentHand, mulliganCount) {
  // Next hand size: 7, 6, 5, 4, 3 (after 0, 1, 2, 3, 4 mulligans)
  const nextSize = 7 - mulliganCount
  // If already at minimum (3 cards after 4 mulls), don't draw
  if (nextSize < 3) return currentHand
  // Draw a new hand
  return randomHand(pool, nextSize)
}

// London mulligan scry: put 1 card to bottom of library (remove from hand)
function applyScry(hand, scryIndex) {
  if (scryIndex < 0 || scryIndex >= hand.length) return hand
  // Remove the card at scryIndex (it goes to bottom of deck)
  return hand.filter((_, idx) => idx !== scryIndex)
}

function ManaCurvePanel({ data, detectedArchetype, fingerprint, decklist, messages }) {
  const [selectedArchetype, setSelectedArchetype] = useState(normalizeMockArchetype(detectedArchetype))
  const [openingHand, setOpeningHand] = useState([])
  const [handSize, setHandSize] = useState(7)
  const [mulliganCount, setMulliganCount] = useState(0)
  const [inScryPhase, setInScryPhase] = useState(false)
  const [graveyard, setGraveyard] = useState([])

  useEffect(() => {
    setSelectedArchetype(normalizeMockArchetype(detectedArchetype))
  }, [detectedArchetype])

  const pool = parseDeckPool(decklist)

  useEffect(() => {
    setHandSize(7)
    setOpeningHand(randomHand(pool, 7))
  }, [decklist])

  const totalCards = data.total_cards || 0
  const nonLandCards = Math.max(0, totalCards - (data.land_count || 0))
  const curveLe2 = clampPercent((nonLandCards > 0
    ? ((data.distribution?.filter(b => (b.cmc || 0) <= 2).reduce((sum, b) => sum + (b.count || 0), 0) / nonLandCards) * 100)
    : 0))
  const finisher4 = clampPercent((nonLandCards > 0
    ? ((data.distribution?.filter(b => (b.cmc || 0) >= 4).reduce((sum, b) => sum + (b.count || 0), 0) / nonLandCards) * 100)
    : 0))
  const profile = MOCK_ARCHETYPE_PROFILES[selectedArchetype] || MOCK_ARCHETYPE_PROFILES.aggro
  const landRatio = totalCards > 0 ? (data.land_count || 0) / totalCards : 0
  const metaMatch = calcMetaMatch(profile, data.average_cmc || 0, landRatio, curveLe2, finisher4)
  const backendTypeDist = data.type_distribution || {}
  const hasBackendTypeDist = (backendTypeDist.creature || 0) + (backendTypeDist.spell || 0) + (backendTypeDist.enchant_artifact || 0) + (backendTypeDist.planeswalker || 0) > 0
  const estimatedTypeDist = estimateTypeDistribution(nonLandCards, selectedArchetype)
  const typeDist = hasBackendTypeDist
    ? {
      creature: backendTypeDist.creature || 0,
      spell: backendTypeDist.spell || 0,
      enchantArtifact: backendTypeDist.enchant_artifact || 0,
      planeswalker: backendTypeDist.planeswalker || 0,
    }
    : estimatedTypeDist

  const creature12 = clampPercent(curveLe2 + (selectedArchetype === 'aggro' ? 8 : selectedArchetype === 'control' ? -10 : 0))
  const fastSpells = clampPercent(100 - ((data.average_cmc || 0) * 16) + (selectedArchetype === 'combo' ? 10 : 0))
  const qualityRows = [
    { label: messages.manaMockCreature12Label, value: Math.round(creature12), target: profile.creature12 },
    { label: messages.manaMockFastSpellsLabel, value: Math.round(fastSpells), target: profile.fastSpells },
    { label: messages.manaMockCurveLe2Label, value: Math.round(curveLe2), target: profile.curveLe2 },
    { label: messages.manaMockFinisher4Label, value: Math.round(finisher4), target: profile.finisher4 },
  ]

  const curveBenchmark = calcCurveBenchmark(selectedArchetype, data.distribution || [], nonLandCards)
  const curveFitScore = curveBenchmark.score
  const curveDeltaSuggestions = buildCurveDeltaSuggestions(curveBenchmark, messages)
  const classifierConfidencePct = Math.round((fingerprint?.confidence || 0) * 100)

  const handEval = evaluateOpeningHand(openingHand, selectedArchetype, messages)

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
      <div style={{ border: '1px solid var(--border)', borderRadius: 10, padding: 12, background: 'rgba(255,255,255,0.02)', marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <div>
            <p style={{ margin: 0, fontSize: '.9rem', fontWeight: 700 }}>{messages.manaMockTitle}</p>
            <p style={{ margin: '4px 0 0', fontSize: '.8rem', color: 'var(--muted)' }}>{messages.manaMockSubtitle}</p>
          </div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {['aggro', 'midrange', 'control', 'combo', 'ramp'].map(arch => (
              <button
                key={arch}
                type="button"
                className={`tab-btn${selectedArchetype === arch ? ' active' : ''}`}
                style={{ borderBottom: 'none', padding: '6px 10px', fontSize: '.8rem' }}
                onClick={() => setSelectedArchetype(arch)}
              >
                {messages.archetypeLabel(arch)}
              </button>
            ))}
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))', gap: 10, marginTop: 12 }}>
          <div className="stat-item"><div className="stat-value">{(data.average_cmc || 0).toFixed(1)}</div><div className="stat-label">{messages.avgCmc}</div></div>
          <div className="stat-item"><div className="stat-value">{nonLandCards}</div><div className="stat-label">{messages.manaMockNonLandLabel}</div></div>
          <div className="stat-item"><div className="stat-value">{data.land_count || 0}</div><div className="stat-label">{messages.manaMockLandsLabel}</div></div>
          <div className="stat-item"><div className="stat-value" style={{ color: scoreColor(metaMatch) }}>{metaMatch}%</div><div className="stat-label">{messages.manaMockMetaMatchLabel}</div></div>
          <div className="stat-item"><div className="stat-value" style={{ color: scoreColor(curveFitScore) }}>{curveFitScore}%</div><div className="stat-label">{messages.manaMockCurveFitLabel}</div></div>
          <div className="stat-item"><div className="stat-value">{classifierConfidencePct}%</div><div className="stat-label">{messages.manaMockClassifierConfidenceLabel}</div></div>
        </div>

        <div style={{ marginTop: 12 }}>
          <p style={{ margin: 0, fontSize: '.82rem', color: 'var(--muted)' }}>{messages.manaMockCurveBenchmarkLabel}</p>
          <div style={{ overflowX: 'auto', marginTop: 8 }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '.82rem' }}>
              <thead>
                <tr>
                  <th style={thStyle}>{messages.manaMockCurveBucketLabel}</th>
                  <th style={thStyle}>{messages.manaMockCurveActualLabel}</th>
                  <th style={thStyle}>{messages.manaMockCurveTargetLabel}</th>
                  <th style={thStyle}>{messages.manaMockCurveDeltaLabel}</th>
                </tr>
              </thead>
              <tbody>
                {curveBenchmark.deltas.map(row => (
                  <tr key={`benchmark-${row.label}`}>
                    <td style={tdStyle}>CMC {row.label}</td>
                    <td style={tdStyle}>{row.actual.toFixed(1)}%</td>
                    <td style={tdStyle}>{row.target.toFixed(0)}%</td>
                    <td style={{ ...tdStyle, color: Math.abs(row.delta) >= 6 ? 'var(--orange)' : 'var(--muted)', fontWeight: Math.abs(row.delta) >= 6 ? 700 : 500 }}>
                      {row.delta > 0 ? '+' : ''}{row.delta.toFixed(1)}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {curveDeltaSuggestions.length > 0 && (
            <ul className="suggestion-list" style={{ marginTop: 10 }}>
              {curveDeltaSuggestions.map((line, idx) => (
                <li key={`curve-delta-suggestion-${idx}`} className="moderate">{line}</li>
              ))}
            </ul>
          )}
        </div>

        <div style={{ marginTop: 12 }}>
          <p style={{ margin: 0, fontSize: '.82rem', color: 'var(--muted)' }}>{messages.manaMockTypeDistLabel}</p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, minmax(110px, 1fr))', gap: 8, marginTop: 8 }}>
            <div className="stat-item"><div className="stat-value" style={{ fontSize: '1rem' }}>{typeDist.creature}</div><div className="stat-label">{messages.manaMockTypeCreature}</div></div>
            <div className="stat-item"><div className="stat-value" style={{ fontSize: '1rem' }}>{typeDist.spell}</div><div className="stat-label">{messages.manaMockTypeSpell}</div></div>
            <div className="stat-item"><div className="stat-value" style={{ fontSize: '1rem' }}>{typeDist.enchantArtifact}</div><div className="stat-label">{messages.manaMockTypeEnchantArtifact}</div></div>
            <div className="stat-item"><div className="stat-value" style={{ fontSize: '1rem' }}>{typeDist.planeswalker}</div><div className="stat-label">{messages.manaMockTypePlaneswalker}</div></div>
          </div>
        </div>

        {data.draw_probabilities && (
          <div style={{ marginTop: 12 }}>
            <p style={{ margin: 0, fontSize: '.82rem', color: 'var(--muted)' }}>{messages.drawProbTitle}</p>
            <p style={{ margin: '4px 0 0', fontSize: '.75rem', color: 'var(--muted)' }}>{messages.drawProbHint}</p>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 8, marginTop: 8 }}>
              <div className="stat-item" title={messages.drawProbT1Hint}>
                <div className="stat-value" style={{ fontSize: '.95rem' }}>{(data.draw_probabilities.turn1_land_prob || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbT1Label}</div>
              </div>
              <div className="stat-item" title={messages.drawProbT2Hint}>
                <div className="stat-value" style={{ fontSize: '.95rem' }}>{(data.draw_probabilities.turn2_lands_prob || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbT2Label}</div>
              </div>
              <div className="stat-item" title={messages.drawProbT3Hint}>
                <div className="stat-value" style={{ fontSize: '.95rem' }}>{(data.draw_probabilities.turn3_lands_prob || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbT3Label}</div>
              </div>
              <div className="stat-item" title={messages.drawProbPerfectHint}>
                <div className="stat-value" style={{ fontSize: '.95rem' }}>{(data.draw_probabilities.perfect_curve_t1_t4 || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbPerfectLabel}</div>
              </div>
              <div className="stat-item" title={messages.drawProbScrewHint}>
                <div className="stat-value" style={{ fontSize: '.95rem', color: 'var(--red)' }}>{(data.draw_probabilities.mana_screw_risk || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbScrewLabel}</div>
              </div>
              <div className="stat-item" title={messages.drawProbFloodHint}>
                <div className="stat-value" style={{ fontSize: '.95rem', color: 'var(--orange)' }}>{(data.draw_probabilities.mana_flood_risk || 0).toFixed(1)}%</div>
                <div className="stat-label">{messages.drawProbFloodLabel}</div>
              </div>
            </div>
          </div>
        )}

        <div style={{ marginTop: 14 }}>
          <p style={{ margin: 0, fontSize: '.82rem', color: 'var(--muted)' }}>{messages.manaMockVsMetaLabel}</p>
          <div style={{ display: 'grid', gap: 8, marginTop: 8 }}>
            {qualityRows.map(row => {
              const status = scoreByTarget(row.value, row.target)
              const statusColor = status === 'great' ? 'var(--green)' : status === 'good' ? 'var(--orange)' : 'var(--red)'
              const statusLabel = status === 'great' ? messages.manaMockStatusGreat : status === 'good' ? messages.manaMockStatusGood : messages.manaMockStatusLow
              return (
                <div key={row.label} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', alignItems: 'center', gap: 8, fontSize: '.84rem' }}>
                  <span>{row.label}</span>
                  <strong>{row.value}%</strong>
                  <span style={{ color: statusColor, fontWeight: 700 }}>{statusLabel}</span>
                </div>
              )
            })}
          </div>
        </div>

        <div style={{ marginTop: 14, border: '1px solid var(--border)', borderRadius: 10, padding: 12, background: 'rgba(255,255,255,0.02)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
            <div>
              <p style={{ margin: 0, fontSize: '.82rem', color: 'var(--muted)' }}>{messages.manaMockOpeningLabel(openingHand.length)}</p>
              <p style={{ margin: '4px 0 0', fontSize: '.75rem', color: 'var(--muted)' }}>
                {messages.londonMulliganCountLabel(mulliganCount)} {graveyard.length > 0 && `· GY: ${graveyard.length}`}
              </p>
            </div>
            {inScryPhase && (
              <div style={{ padding: '4px 8px', background: 'var(--orange)', color: 'white', borderRadius: 4, fontSize: '.75rem', fontWeight: 700 }}>
                {messages.londonScryPhaseLabel}
              </div>
            )}
          </div>
          <div style={{ marginTop: 8, display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 8 }}>
            {openingHand.length > 0 ? openingHand.map((card, idx) => (
              <div
                key={`open-hand-${idx}`}
                onClick={() => {
                  if (inScryPhase && mulliganCount < 4) {
                    const afterScry = applyScry(openingHand, idx)
                    setOpeningHand(afterScry)
                    setGraveyard([...graveyard, card])
                    setInScryPhase(false)
                  }
                }}
                style={{
                  border: inScryPhase ? '2px dashed var(--orange)' : '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '6px 8px',
                  fontSize: '.82rem',
                  background: inScryPhase ? 'rgba(255,165,0,0.1)' : 'var(--bg)',
                  cursor: inScryPhase ? 'pointer' : 'default',
                  opacity: inScryPhase ? 0.9 : 1,
                  transition: 'all 0.2s',
                }}
                title={inScryPhase ? messages.londonScryHint : ''}
              >
                {card.name}
              </div>
            )) : (
              <div style={{ color: 'var(--muted)', fontSize: '.82rem' }}>{messages.manaMockNoCards}</div>
            )}
          </div>
          <p style={{ margin: '10px 0 0', color: handEval.tone, fontSize: '.82rem', fontWeight: 600 }}>{handEval.text}</p>
          <div style={{ marginTop: 10, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {!inScryPhase && (
              <>
                <button
                  type="button"
                  className="btn-primary"
                  onClick={() => {
                    setMulliganCount(0)
                    setGraveyard([])
                    setOpeningHand(randomHand(pool, 7))
                  }}
                  style={{ padding: '8px 12px', fontSize: '.82rem' }}
                >
                  {messages.londonNewGameLabel}
                </button>
                <button
                  type="button"
                  className="btn-ghost"
                  onClick={() => {
                    if (mulliganCount >= 4) {
                      alert(messages.londonFullMulliganLabel)
                      return
                    }
                    setMulliganCount(mulliganCount + 1)
                    setInScryPhase(true)
                  }}
                  disabled={mulliganCount >= 4}
                  style={{ opacity: mulliganCount >= 4 ? 0.5 : 1, cursor: mulliganCount >= 4 ? 'not-allowed' : 'pointer' }}
                >
                  {messages.londonMulliganLabel}
                </button>
                <span style={{ marginLeft: 'auto', fontSize: '.8rem', color: 'var(--green)', fontWeight: 700 }}>{messages.londonKeepLabel}</span>
              </>
            )}
            {inScryPhase && (
              <span style={{ fontSize: '.8rem', color: 'var(--orange)', fontWeight: 700, width: '100%' }}>
                {messages.londonSelectScryLabel}
              </span>
            )}
          </div>
        </div>
      </div>

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

export function AIPanel({ text, error, source, result, commanderScore, messages }) {
  const normalizedSource = String(source || '').trim()
  const sourceLabel = normalizedSource
    ? messages.aiSourceUsed(normalizedSource)
    : messages.aiSourceUsed(messages.aiSourceUnknown)
  const internalSource = normalizedSource.startsWith('internal_rules')
  const fallbackActive = Boolean(error) && internalSource
  const statusLabel = fallbackActive
    ? messages.aiStatusFallback
    : internalSource
      ? messages.aiStatusInternal
      : messages.aiStatusExternal
  const statusToneClass = fallbackActive ? 'warn' : internalSource ? 'info' : 'ok'

  if (!text) {
    const fallbackLines = buildLocalSummary(result, commanderScore, messages)
    return (
      <div>
        <div className="ai-meta-row">
          <span className="ai-badge">{messages.localSummaryBadge}</span>
          <span className={`ai-status-pill ${statusToneClass}`}>{statusLabel}</span>
        </div>
        <div className="banner banner-warn" style={{ marginBottom: 12 }}>
          <strong>{messages.aiUnavailable}</strong>
          <div style={{ marginTop: 6 }}>{error || messages.aiFallbackNote}</div>
        </div>
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
      <div className="ai-meta-row">
        <span className="ai-badge">{messages.aiBadge}</span>
        <span className={`ai-status-pill ${statusToneClass}`}>{statusLabel}</span>
      </div>
      <div className="ai-source-label">{sourceLabel}</div>
      {fallbackActive && (
        <div className="banner banner-warn" style={{ marginBottom: 12 }}>
          <strong>{messages.aiFallbackActiveTitle}</strong>
          <div style={{ marginTop: 6 }}>{error}</div>
        </div>
      )}
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
                                    <CardHoverPreview cardName={item.card_name} token={token} messages={messages}>
                                      {item.card_name}
                                    </CardHoverPreview>
                                    {' '}x{item.quantity}: {item.reason}
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

function buildLocalSummary(result, commanderScore, messages) {
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

  const lines = [
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

  if (typeof commanderScore === 'number') {
    const bracket = commanderBracketForScore(commanderScore)
    lines.push(`${messages.commanderBracketLabel || 'Commander bracket'}: ${bracket.bracket} · ${bracket.label} · ${commanderScore.toFixed(1)}/10`)
  }

  return lines
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

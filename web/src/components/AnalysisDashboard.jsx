// Riepilogo visivo dell'analisi: punteggio, terre nelle prime carte,
// numeri chiave, curva di mana e modifiche consigliate.
import { ManaSymbol } from './ManaSymbol'



function clamp(v, lo = 0, hi = 100) {
  return Math.max(lo, Math.min(hi, v))
}

function fmtNum(value, locale, digits = 1) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '–'
  return n.toLocaleString(locale === 'en' ? 'en-GB' : 'it-IT', { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

function pct(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 0
  return n <= 1 && n > 0 ? n * 100 : n
}

// Indice di salute del mazzo (0–10), calcolato nel browser dai dati
// deterministici dell'analisi: equilibrio delle terre, fonti di colore,
// probabilità di una mano giocabile e densità d'interazione.
export function deckHealthScore(result) {
  const mana = result?.deterministic?.mana
  if (!mana) return null
  const sweet = pct(mana.sweet_spot_chance) || 60
  const landDiff = Math.abs((mana.land_count || 0) - (mana.ideal_land_count || mana.land_count || 0))
  const landFit = clamp(100 - landDiff * 15)
  const gaps = (mana.source_requirements || []).filter(r => r.color !== 'TOTAL').reduce((acc, r) => acc + Math.max(0, r.gap || 0), 0)
  const sourceFit = clamp(100 - gaps * 8)
  const interaction = clamp(Number(result?.deterministic?.interaction?.total_score) || 50)
  const total = 0.45 * sweet + 0.25 * landFit + 0.15 * sourceFit + 0.15 * interaction
  return Math.round(total) / 10
}

export function scoreTone(score) {
  if (score >= 7.5) return 'good'
  if (score >= 5.5) return 'warn'
  return 'bad'
}

function ScoreRing({ score, label }) {
  const r = 52
  const c = 2 * Math.PI * r
  const filled = c * clamp(score * 10) / 100
  return (
    <div className={`mw-score mw-score--${scoreTone(score)}`} role="img" aria-label={label}>
      <svg viewBox="0 0 128 128" width="128" height="128" aria-hidden="true">
        <circle cx="64" cy="64" r={r} className="mw-score__track" />
        <circle cx="64" cy="64" r={r} className="mw-score__value" strokeDasharray={`${filled} ${c - filled}`} transform="rotate(-90 64 64)" />
      </svg>
      <div className="mw-score__text">
        <span className="mw-score__num">{score.toLocaleString('it-IT', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}</span>
        <span className="mw-score__max">/ 10</span>
      </div>
    </div>
  )
}

function suggestionTitle(s, m) {
  const cmcLabel = s.cmc >= 7 ? '7+' : String(s.cmc)
  if (/land count/i.test(s.reason) || (s.cmc === 0 && /land/i.test(s.reason))) {
    return s.type === 'add' ? m.wSugAddLands : m.wSugCutLands
  }
  if (/sources/i.test(s.reason)) return m.wSugSources
  if (/screw/i.test(s.reason)) return m.wSugScrew
  if (/flood/i.test(s.reason)) return m.wSugFlood
  if (/fixing lands/i.test(s.reason)) return m.wSugFixing
  if (/average cmc/i.test(s.reason)) return s.type === 'add' ? m.wSugRaiseCurve : m.wSugLowerCurve
  if (s.cmc > 0) return s.type === 'add' ? m.wSugAddAt(cmcLabel) : m.wSugCutAt(cmcLabel)
  return s.type === 'add' ? m.wSugAdd : m.wSugCut
}

// Traduce le motivazioni note del motore di analisi; le altre restano originali.
export function localizeReason(reason, m, locale) {
  if (locale !== 'it' || !reason) return reason
  const rules = [
    [/^Land count is higher than the format ideal/i, () => 'Le terre superano il numero ideale per il formato: valuta di togliere qualche base per più magie.'],
    [/^Land count is below the format ideal/i, () => 'Le terre sono sotto il numero ideale: rischi pescate poco costanti.'],
    [/^Too many high-CMC cards/i, () => 'Troppe carte costose: alleggerire la parte alta migliora i primi turni.'],
    [/^Average CMC ([\d.]+) is high for (\w+)/i, (x) => `Costo medio ${x[1].replace('.', ',')} alto per ${x[2]}: sostituisci qualche magia costosa con interazione più economica.`],
    [/^Average CMC is very low/i, () => 'Costo medio molto basso: valuta qualche minaccia di metà partita.'],
    [/^Detected (\d+) flexible fixing lands/i, (x) => `${x[1]} terre di fixing flessibili: migliorano la costanza dei colori e le sinergie con le terre.`],
    [/^High mana screw risk \(([\d.]+)%\) in the first (\d+) draws/i, (x) => `Rischio alto di restare senza terre (${x[1].replace('.', ',')}%) nelle prime ${x[2]} carte: aggiungi fonti stabili o alleggerisci i costi colorati iniziali.`],
    [/^Mana screw risk is elevated \(([\d.]+)%\)/i, (x) => `Rischio di poche terre elevato (${x[1].replace('.', ',')}%): valuta +1 fonte o una curva iniziale più morbida.`],
    [/^High mana flood risk \(([\d.]+)%\)/i, (x) => `Rischio alto di troppe terre (${x[1].replace('.', ',')}%): togli qualche terra o aggiungi modi di usare il mana in eccesso.`],
    [/^You are short on mana sources \((\d+) current, (\d+) target\)/i, (x) => `Fonti di mana insufficienti (${x[1]} su ${x[2]}): aggiungi terre o produttori di mana.`],
    [/^You are short on (\w) sources \((\d+) current, (\d+) required\)/i, (x) => `Poche fonti di ${m.wColorName?.[x[1]] || x[1]} (${x[2]} su ${x[3]}): aggiungi terre doppie o base.`],
  ]
  for (const [re, fn] of rules) {
    const match = reason.match(re)
    if (match) return fn(match)
  }
  return reason
}

const URGENCY_ORDER = { critical: 0, moderate: 1, minor: 2 }

export default function AnalysisDashboard({ result, fingerprint, messages: m, locale, onEditList }) {
  const mana = result?.deterministic?.mana
  if (!mana) return null
  const score = deckHealthScore(result)
  const tone = scoreTone(score)

  const screw = pct(mana.mana_screw_chance)
  const flood = pct(mana.mana_flood_chance)
  const sweet = pct(mana.sweet_spot_chance) || Math.max(0, 100 - screw - flood)
  const draws = mana.land_sample_draws || 9

  const buckets = [0, 1, 2, 3, 4, 5, 6, 7].map(cmc => ({ cmc, count: 0 }))
  for (const b of mana.distribution || []) {
    const idx = Math.min(7, Math.max(0, Number(b.cmc) || 0))
    buckets[idx].count += Number(b.count) || 0
  }
  // Le carte a costo 0 sono quasi sempre terre: la colonna "Terre" le sostituisce.
  const bars = [
    ...buckets.slice(1).map(b => ({ label: b.cmc === 7 ? '7+' : String(b.cmc), count: b.count })),
    { label: m.wLands, count: mana.land_count || 0, land: true },
  ]
  const maxBar = Math.max(1, ...bars.map(b => b.count))

  const sources = (mana.source_requirements || []).filter(r => r.color && r.color !== 'TOTAL' && r.color !== 'C')
  const perfect = mana.draw_probabilities?.perfect_curve_t1_t4

  const suggestions = [...(mana.suggestions || [])]
    .sort((a, b) => (URGENCY_ORDER[a.urgency] ?? 3) - (URGENCY_ORDER[b.urgency] ?? 3))
    .slice(0, 5)
  const interactionTips = (result?.deterministic?.interaction?.suggestions || []).slice(0, Math.max(0, 5 - suggestions.length))

  const colors = fingerprint?.color_identity || Object.keys(mana.color_distribution || {}).filter(k => k !== 'C' && mana.color_distribution[k] > 0)

  return (
    <section className="mw-dash" aria-label={m.wDashAria}>
      <div className="mw-dash__col">
        <div className="mw-panel mw-panel--score">
          <ScoreRing score={score} label={m.wScoreAria(fmtNum(score, locale))} />
          <div>
            <h2 className="mw-panel__title">{m[`wVerdict_${tone}`]}</h2>
            <p className="mw-muted">{m[`wVerdictText_${tone}`]}</p>
            <p className="mw-hint">{m.wScoreHint}</p>
          </div>
        </div>

        <div className="mw-panel">
          <h2 className="mw-panel__title">{m.wLandsInFirst(draws)}</h2>
          <div className="mw-stack" role="img" aria-label={m.wLandsAria(Math.round(screw), Math.round(sweet), Math.round(flood))}>
            <span className="mw-stack__seg mw-stack__seg--bad" style={{ flexBasis: `${clamp(screw)}%` }} />
            <span className="mw-stack__seg mw-stack__seg--good" style={{ flexBasis: `${clamp(sweet)}%` }} />
            <span className="mw-stack__seg mw-stack__seg--warn" style={{ flexBasis: `${clamp(flood)}%` }} />
          </div>
          <div className="mw-stack__legend">
            <div><p>{m.wTooFew}</p><strong>{Math.round(screw)}%</strong></div>
            <div><p>{m.wRight}</p><strong>{Math.round(sweet)}%</strong></div>
            <div><p>{m.wTooMany}</p><strong>{Math.round(flood)}%</strong></div>
          </div>
        </div>

        <div className="mw-panel">
          <h2 className="mw-panel__title">{m.wInShort}</h2>
          <dl className="mw-facts">
            <div><dt>{m.avgCmc}</dt><dd>{fmtNum(mana.average_cmc, locale, 2)}</dd></div>
            <div><dt>{m.wLands}</dt><dd>{mana.land_count} <span className="mw-muted">{m.wIdeal(mana.ideal_land_count)}</span></dd></div>
            {typeof perfect === 'number' && <div><dt>{m.wPerfectCurve}</dt><dd>{Math.round(pct(perfect))}%</dd></div>}
            {sources.length > 0 && (
              <div>
                <dt>{m.wColorSources}</dt>
                <dd className="mw-sources">
                  {sources.map(r => (
                    <span key={r.color} className={r.gap > 0 ? 'is-short' : ''} title={m.wSourcesTitle(r.current, r.required)}>
                      <ManaSymbol code={r.color} size={16} /> {r.current}/{r.required}
                    </span>
                  ))}
                </dd>
              </div>
            )}
            {colors.length > 0 && (
              <div><dt>{m.colorIdentity}</dt><dd className="mw-sources">{colors.map(c => <ManaSymbol key={c} code={c} size={18} />)}</dd></div>
            )}
          </dl>
        </div>
      </div>

      <div className="mw-dash__col">
        <div className="mw-panel">
          <h2 className="mw-panel__title">{m.wCurve}</h2>
          <div className="mw-bars" role="img" aria-label={m.wCurveAria(bars.map(b => `${b.label}: ${b.count}`).join(', '))}>
            {bars.map(b => (
              <div key={b.label} className={`mw-bars__col${b.land ? ' is-land' : ''}`}>
                <span className="mw-bars__count">{b.count}</span>
                <span className="mw-bars__bar" style={{ height: `${Math.round((b.count / maxBar) * 100)}%` }} />
              </div>
            ))}
          </div>
          <div className="mw-bars__axis">{bars.map(b => <span key={b.label}>{b.label}</span>)}</div>
        </div>

        <div className="mw-panel">
          <div className="mw-panel__head">
            <h2 className="mw-panel__title">{m.wWhatToChange}</h2>
            {onEditList && <button type="button" className="mw-btn mw-btn--ghost mw-btn--sm" onClick={onEditList}>{m.wTryChanges}</button>}
          </div>
          {suggestions.length === 0 && interactionTips.length === 0 ? (
            <p className="mw-muted">{m.wNoChanges}</p>
          ) : (
            <ul className="mw-sugs">
              {suggestions.map((s, i) => (
                <li key={`m${i}`}>
                  <span className={`mw-urg mw-urg--${s.urgency || 'minor'}`}>{m[`wUrg_${s.urgency || 'minor'}`]}</span>
                  <div>
                    <p className="mw-sugs__title">{suggestionTitle(s, m)}</p>
                    <p className="mw-muted">{localizeReason(s.reason, m, locale)}</p>
                  </div>
                </li>
              ))}
              {interactionTips.map((tip, i) => (
                <li key={`i${i}`}>
                  <span className="mw-urg mw-urg--minor">{m.wUrg_minor}</span>
                  <div><p className="mw-sugs__title">{m.wInteraction}</p><p className="mw-muted">{tip}</p></div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </section>
  )
}

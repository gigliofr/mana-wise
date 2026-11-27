// Utilità per le liste di carte: parsing, testo, formato, mazzo corrente.

const SECTION_SIDEBOARD = /^(sideboard|side|sb)\s*:?\s*$/i
const SECTION_MAIN = /^(deck|main|mainboard|maindeck)\s*:?\s*$/i
const SECTION_COMMANDER = /^(commander|comandante)\s*:?\s*$/i
const SECTION_MAYBE = /^(maybeboard|maybe|considering|forse)\s*:?\s*$/i
const SECTION_IGNORED = /^(companion|about|name\s.*)$/i

// Suffissi che gli export aggiungono dopo il nome, tolti in qualsiasi ordine:
// tag "[Commander]", marcatori foil/etched "*F*", set e numero di collezione
// "(MH3) 123", anche con numeri come "247p", "D25" o "M15-7" (The List).
const TRAILING_TAG = /\s*\[[^\]]+\]\s*$/
const TRAILING_FOIL = /\s*\*(?:f|e|foil|etched)\*\s*$/i
const TRAILING_SET = /\s*\([A-Za-z0-9]{2,10}\)(?:\s+[A-Za-z0-9][A-Za-z0-9★†-]*)?\s*$/

// cleanCardName toglie dal nome tag, marcatori foil, set e numero di collezione.
export function cleanCardName(value) {
  let name = String(value || '').trim()
  for (;;) {
    const next = name.replace(TRAILING_TAG, '').replace(TRAILING_FOIL, '').replace(TRAILING_SET, '').trim()
    if (next === name) break
    name = next
  }
  name = name.replace(/\s+/g, ' ')
  // Carte doppie: Moxfield scrive "Ready / Willing", Scryfall "Ready // Willing".
  if (!name.includes('//')) name = name.replace(/\s+\/\s+/g, ' // ')
  return name
}

const collator = new Intl.Collator('en', { sensitivity: 'base', ignorePunctuation: true })

// Moxfield e Archidekt esportano i mazzi Commander senza intestazioni, con il
// comandante sulla prima riga e il resto in ordine alfabetico: se la prima
// carta rompe quell'ordine, è il comandante.
function inferCommanderFromOrder(out) {
  if (out.commander.length || out.side.length || out.main.length < 3) return
  const [first, ...rest] = out.main
  if (first.qty !== 1) return
  const total = sum(out.main)
  const singleton = rest.every(c => c.qty === 1 || isBasicLand(c.name))
  if (total < 98 || total > 101 || !singleton) return
  for (let i = 1; i < rest.length; i++) {
    if (collator.compare(rest[i - 1].name, rest[i].name) > 0) return
  }
  if (collator.compare(first.name, rest[0].name) <= 0) return
  out.commander.push(first)
  out.main = rest
}

// parseDecklist riconosce le liste di Arena, Moxfield e testo semplice.
// Restituisce { main, side, commander, maybe } con elementi { name, qty }.
// "maybe" sono le carte in valutazione: non fanno parte del mazzo.
export function parseDecklist(text) {
  const out = { main: [], side: [], commander: [], maybe: [] }
  let section = 'main'
  let sawCards = false
  let sawHeader = false
  const lines = String(text || '').replace(/\r/g, '').split('\n')
  for (const raw of lines) {
    const line = raw.trim()
    if (!line) {
      // Una riga vuota dopo le carte, nei formati senza intestazioni, introduce la sideboard.
      if (sawCards && section === 'main') section = 'pending-side'
      continue
    }
    if (line.startsWith('//') || line.startsWith('#')) continue
    if (SECTION_SIDEBOARD.test(line)) { section = 'side'; sawHeader = true; continue }
    if (SECTION_MAIN.test(line)) { section = 'main'; sawHeader = true; continue }
    if (SECTION_COMMANDER.test(line)) { section = 'commander'; sawHeader = true; continue }
    if (SECTION_MAYBE.test(line)) { section = 'maybe'; sawHeader = true; continue }
    if (SECTION_IGNORED.test(line)) continue
    let target = section === 'pending-side' ? 'side' : section
    let body = line
    if (/^sb:\s*/i.test(body)) { target = 'side'; body = body.replace(/^sb:\s*/i, '') }
    const m = body.match(/^(\d+)\s*x?\s+(.+)$/i)
    const qty = m ? Math.max(1, Number(m[1]) || 1) : 1
    const rawName = m ? m[2] : body
    const isCommanderTag = /\[[^\]]*commander[^\]]*\]/i.test(rawName)
    const name = cleanCardName(rawName)
    if (!name) continue
    if (isCommanderTag) target = 'commander'
    const list = out[target]
    const existing = list.find(c => c.name.toLowerCase() === name.toLowerCase())
    if (existing) existing.qty += qty
    else list.push({ name, qty })
    sawCards = true
  }
  // Una "sideboard" da riga vuota con oltre 15 carte era in realtà main.
  if (out.side.length && sum(out.side) > 15 && section !== 'side') {
    out.main.push(...out.side)
    out.side = []
  }
  if (!sawHeader) inferCommanderFromOrder(out)
  return out
}

export function sum(list) {
  return list.reduce((acc, c) => acc + (Number(c.qty) || 0), 0)
}

export function listToText(list) {
  return list.map(c => `${c.qty} ${c.name}`).join('\n')
}

// Testo completo con le sezioni, accettato dall'analisi.
export function deckToText(parsed) {
  const parts = []
  if (parsed.commander?.length) parts.push(`Commander\n${listToText(parsed.commander)}`)
  parts.push(`${parsed.commander?.length ? 'Deck\n' : ''}${listToText(parsed.main)}`)
  if (parsed.side?.length) parts.push(`Sideboard\n${listToText(parsed.side)}`)
  if (parsed.maybe?.length) parts.push(`Maybeboard\n${listToText(parsed.maybe)}`)
  return parts.join('\n\n')
}

// Testo per gli strumenti che vogliono solo il main deck (comandante incluso).
export function mainText(parsed) {
  return listToText([...(parsed.commander || []), ...parsed.main])
}

export function guessFormat(parsed, fallback = 'modern') {
  const total = sum(parsed.main) + sum(parsed.commander || [])
  if (parsed.commander?.length || (total >= 98 && total <= 101 && parsed.main.every(c => c.qty === 1 || isBasicLand(c.name)))) return 'commander'
  return fallback
}

const BASICS = ['plains', 'island', 'swamp', 'mountain', 'forest', 'wastes']
export function isBasicLand(name) {
  const n = String(name || '').toLowerCase().replace(/^snow-covered\s+/, '')
  return BASICS.includes(n)
}

// Conversione da/verso il formato delle API /decks.
export function apiCardsFromParsed(parsed) {
  return [
    ...(parsed.commander || []).map(c => ({ card_id: '', card_name: c.name, quantity: c.qty, is_sideboard: false, is_commander: true })),
    ...parsed.main.map(c => ({ card_id: '', card_name: c.name, quantity: c.qty, is_sideboard: false, is_commander: false })),
    ...parsed.side.map(c => ({ card_id: '', card_name: c.name, quantity: c.qty, is_sideboard: true, is_commander: false })),
  ]
}

// Carte "forse" nel formato delle API /decks (campo maybeboard).
export function apiMaybeboardFromParsed(parsed) {
  return (parsed.maybe || []).map(c => ({ card_id: '', card_name: c.name, quantity: c.qty, is_sideboard: false, is_commander: false }))
}

export function parsedFromApiDeck(deck) {
  const out = { main: [], side: [], commander: [], maybe: [] }
  for (const c of Array.isArray(deck?.maybeboard) ? deck.maybeboard : []) {
    const name = c.card_name || c.name || ''
    if (name) out.maybe.push({ name, qty: Math.max(1, Number(c.quantity) || 1) })
  }
  for (const c of Array.isArray(deck?.cards) ? deck.cards : []) {
    const name = c.card_name || c.name || ''
    if (!name) continue
    const item = { name, qty: Math.max(1, Number(c.quantity) || 1) }
    if (c.is_commander) out.commander.push(item)
    else if (c.is_sideboard) out.side.push(item)
    else out.main.push(item)
  }
  return out
}

export function defaultDeckName(parsed, locale = 'it') {
  const lead = parsed.commander?.[0]?.name
  if (lead) return lead
  const date = new Date().toLocaleDateString(locale === 'en' ? 'en-GB' : 'it-IT', { day: 'numeric', month: 'short' })
  return locale === 'en' ? `Deck of ${date}` : `Mazzo del ${date}`
}

// ---------- Mazzo corrente e riepiloghi salvati in locale ----------

const CURRENT_KEY = 'manawise_current_deck'
const INSIGHTS_KEY = 'manawise_deck_insights'

export function loadCurrentDeck() {
  try {
    const v = JSON.parse(localStorage.getItem(CURRENT_KEY) || 'null')
    return v && typeof v.text === 'string' ? v : null
  } catch {
    return null
  }
}

export function saveCurrentDeck(deck) {
  try {
    if (deck) localStorage.setItem(CURRENT_KEY, JSON.stringify(deck))
    else localStorage.removeItem(CURRENT_KEY)
  } catch {
    // archiviazione non disponibile
  }
}

export function loadInsights() {
  try {
    return JSON.parse(localStorage.getItem(INSIGHTS_KEY) || '{}') || {}
  } catch {
    return {}
  }
}

export function saveInsight(deckKey, insight) {
  if (!deckKey) return
  const all = loadInsights()
  all[deckKey] = { ...insight, at: new Date().toISOString() }
  try {
    localStorage.setItem(INSIGHTS_KEY, JSON.stringify(all))
  } catch {
    // archiviazione piena o non disponibile
  }
}

export function relativeTime(iso, locale = 'it') {
  const t = iso ? new Date(iso).getTime() : 0
  if (!t) return ''
  const diff = Math.max(0, Date.now() - t)
  const rtf = new Intl.RelativeTimeFormat(locale === 'en' ? 'en' : 'it', { numeric: 'auto' })
  const min = Math.round(diff / 60000)
  if (min < 60) return rtf.format(-Math.max(1, min), 'minute')
  const h = Math.round(min / 60)
  if (h < 24) return rtf.format(-h, 'hour')
  const d = Math.round(h / 24)
  if (d < 7) return rtf.format(-d, 'day')
  return rtf.format(-Math.round(d / 7), 'week')
}

// ---------- Mazzo di esempio ----------
// Boros Burn per Modern, lista completa (60 + 15).
export const SAMPLE_DECK = `Deck
4 Goblin Guide
4 Monastery Swiftspear
4 Eidolon of the Great Revel
4 Lightning Bolt
4 Rift Bolt
4 Lava Spike
4 Boros Charm
4 Lightning Helix
4 Searing Blaze
4 Skullcrack
4 Inspiring Vantage
4 Sacred Foundry
2 Sunbaked Canyon
2 Arid Mesa
6 Mountain
2 Bloodstained Mire

Sideboard
3 Path to Exile
2 Deflecting Palm
3 Kor Firewalker
3 Rest in Peace
2 Smash to Smithereens
2 Roiling Vortex`

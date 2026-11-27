// Raggruppamento e ordinamento delle carte per le viste del mazzo.

export const TYPE_ORDER = ['creature', 'planeswalker', 'instant', 'sorcery', 'artifact', 'enchantment', 'battle', 'land', 'other']
export const COLOR_ORDER = ['W', 'U', 'B', 'R', 'G', 'M', 'C']

// typeGroup assegna una carta al suo tipo principale (le creature artefatto
// sono creature, le terre restano terre anche se sono anche altro).
export function typeGroup(typeLine) {
  const t = String(typeLine || '').toLowerCase().split('//')[0]
  if (!t.trim()) return 'other'
  if (t.includes('land')) return 'land'
  if (t.includes('creature')) return 'creature'
  for (const k of TYPE_ORDER) if (t.includes(k)) return k
  return 'other'
}

export function colorGroup(info) {
  const colors = Array.isArray(info?.colors) ? info.colors : []
  if (colors.length > 1) return 'M'
  if (colors.length === 1) return colors[0]
  return 'C'
}

// costGroup: le terre a parte, il resto per valore di mana (7 e oltre insieme).
export function costGroup(info) {
  if (typeGroup(info?.typeLine) === 'land') return 'land'
  return String(Math.min(7, Math.floor(Number(info?.cmc) || 0)))
}

export function groupKey(entry, info, by) {
  if (entry.zone === 'commander') return 'commander'
  if (by === 'cost') return costGroup(info)
  if (by === 'color') return colorGroup(info)
  if (by === 'none') return 'all'
  return typeGroup(info?.typeLine)
}

function groupRank(key, by) {
  if (key === 'commander') return -1
  if (by === 'cost') return key === 'land' ? 99 : Number(key)
  if (by === 'color') return COLOR_ORDER.indexOf(key)
  if (by === 'none') return 0
  return TYPE_ORDER.indexOf(key)
}

const collator = new Intl.Collator('en', { sensitivity: 'base' })

export function sortEntries(entries, infoOf, by) {
  const byName = (a, b) => collator.compare(a.name, b.name)
  const list = [...entries]
  if (by === 'cost') {
    list.sort((a, b) => (infoOf(a)?.cmc ?? 99) - (infoOf(b)?.cmc ?? 99) || byName(a, b))
  } else if (by === 'price') {
    // Prima le più care; le carte senza prezzo in fondo.
    list.sort((a, b) => (infoOf(b)?.priceEur ?? -1) - (infoOf(a)?.priceEur ?? -1) || byName(a, b))
  } else {
    list.sort(byName)
  }
  return list
}

// groupEntries restituisce [{ key, entries, count }] nell'ordine di lettura.
// count somma le copie, non le righe.
export function groupEntries(entries, infoOf, groupBy, sortBy) {
  const groups = new Map()
  for (const e of entries) {
    const key = groupKey(e, infoOf(e), groupBy)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key).push(e)
  }
  return [...groups.entries()]
    .sort((a, b) => groupRank(a[0], groupBy) - groupRank(b[0], groupBy))
    .map(([key, list]) => ({
      key,
      entries: sortEntries(list, infoOf, sortBy),
      count: list.reduce((acc, e) => acc + (Number(e.qty) || 0), 0),
    }))
}

export function totalPrice(entries, infoOf) {
  let total = 0
  let priced = 0
  for (const e of entries) {
    const p = infoOf(e)?.priceEur
    if (p == null) continue
    total += p * (Number(e.qty) || 0)
    priced++
  }
  return { total, priced }
}

export function formatEur(value, locale = 'it') {
  return new Intl.NumberFormat(locale === 'en' ? 'en-GB' : 'it-IT', { style: 'currency', currency: 'EUR' }).format(value)
}

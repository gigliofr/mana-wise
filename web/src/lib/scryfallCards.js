// Dati delle carte da Scryfall per la vista visuale: immagine, costo, tipo,
// colori, rarità, edizione e prezzo. Usa /cards/collection (75 carte per
// richiesta), con una cache in memoria e nella sessione del browser.

const COLLECTION_URL = 'https://api.scryfall.com/cards/collection'
const BATCH_SIZE = 75
const STORAGE_KEY = 'manawise_scryfall_cards_v1'
const STORAGE_TTL_MS = 12 * 60 * 60 * 1000

const memory = new Map()
let storageLoaded = false

function keyOf(name) {
  return String(name || '').trim().toLowerCase()
}

function loadStorage() {
  if (storageLoaded) return
  storageLoaded = true
  try {
    const raw = JSON.parse(sessionStorage.getItem(STORAGE_KEY) || 'null')
    if (!raw || Date.now() - raw.savedAt > STORAGE_TTL_MS) return
    for (const [k, v] of Object.entries(raw.cards || {})) memory.set(k, v)
  } catch {
    // la cache è solo un aiuto: se manca si ricarica da Scryfall
  }
}

function saveStorage() {
  try {
    const cards = {}
    for (const [k, v] of memory) if (v) cards[k] = v
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ savedAt: Date.now(), cards }))
  } catch {
    // sessionStorage pieno o non disponibile
  }
}

function faceImage(card, size) {
  if (card?.image_uris?.[size]) return card.image_uris[size]
  const face = Array.isArray(card?.card_faces) ? card.card_faces.find(f => f?.image_uris?.[size]) : null
  return face?.image_uris?.[size] || ''
}

function toPrice(value) {
  const n = Number.parseFloat(value)
  return Number.isFinite(n) ? n : null
}

// toCardInfo riduce una carta di Scryfall ai campi che servono alla vista.
export function toCardInfo(card) {
  const front = Array.isArray(card?.card_faces) ? card.card_faces[0] : null
  return {
    name: card?.name || '',
    image: faceImage(card, 'normal'),
    imageLarge: faceImage(card, 'large') || faceImage(card, 'normal'),
    manaCost: card?.mana_cost || front?.mana_cost || '',
    cmc: Number(card?.cmc) || 0,
    typeLine: card?.type_line || front?.type_line || '',
    colors: card?.colors || front?.colors || [],
    colorIdentity: card?.color_identity || [],
    rarity: card?.rarity || '',
    set: card?.set || '',
    priceEur: toPrice(card?.prices?.eur),
    priceUsd: toPrice(card?.prices?.usd),
  }
}

function remember(info, requestedName) {
  memory.set(keyOf(info.name), info)
  if (requestedName) memory.set(keyOf(requestedName), info)
  // "Fire // Ice" si trova anche come "Fire".
  const front = info.name.split(' // ')[0]
  if (front && front !== info.name) memory.set(keyOf(front), info)
}

// fetchCardInfos restituisce una mappa nome-in-minuscolo → info per i nomi
// richiesti. I nomi che Scryfall non trova valgono null.
export async function fetchCardInfos(names, { signal, fetchImpl = fetch } = {}) {
  loadStorage()
  const wanted = Array.from(new Set(names.map(n => String(n || '').trim()).filter(Boolean)))
  const missing = wanted.filter(n => !memory.has(keyOf(n)))

  for (let i = 0; i < missing.length; i += BATCH_SIZE) {
    const batch = missing.slice(i, i + BATCH_SIZE)
    const res = await fetchImpl(COLLECTION_URL, {
      method: 'POST',
      signal,
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ identifiers: batch.map(name => ({ name })) }),
    })
    if (!res.ok) throw new Error(`Scryfall ${res.status}`)
    const data = await res.json()
    const found = Array.isArray(data?.data) ? data.data : []
    for (const card of found) {
      const info = toCardInfo(card)
      const requested = batch.find(n => keyOf(n) === keyOf(info.name) || keyOf(n) === keyOf(info.name.split(' // ')[0]))
      remember(info, requested)
    }
    for (const nf of Array.isArray(data?.not_found) ? data.not_found : []) {
      if (nf?.name && !memory.has(keyOf(nf.name))) memory.set(keyOf(nf.name), null)
    }
    // Scryfall chiede di non superare ~10 richieste al secondo.
    if (i + BATCH_SIZE < missing.length) await new Promise(r => setTimeout(r, 120))
  }
  if (missing.length) saveStorage()

  const out = {}
  for (const n of wanted) out[keyOf(n)] = memory.get(keyOf(n)) ?? null
  return out
}

export function clearCardInfoCache() {
  memory.clear()
  storageLoaded = true
  try { sessionStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
}

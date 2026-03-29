import { useMemo, useState } from 'react'

const API = '/api/v1'
const previewCache = new Map()
const FAILED_CACHE_TTL_MS = 90 * 1000

function normalizeCardName(name) {
  return String(name || '')
    .trim()
    .replace(/\s+/g, ' ')
    .toLowerCase()
}

function deriveCardNameCandidates(rawName) {
  const base = String(rawName || '').trim()
  if (!base) return []

  const out = new Set()
  const push = value => {
    const v = String(value || '').trim().replace(/\s+/g, ' ')
    if (v) out.add(v)
  }

  push(base)
  // Strip Arena-style quantity prefix (e.g. "4 Lightning Bolt").
  push(base.replace(/^\d+x?\s+/i, ''))
  // Strip set/collector suffix (e.g. "Card Name (SET) 123").
  push(base.replace(/\s*\([A-Za-z0-9]{2,6}\)\s*[A-Za-z0-9-]+\s*$/i, ''))

  const cleaned = Array.from(out)
  for (const name of cleaned) {
    // Normalize split card separators for better fuzzy matching.
    push(name.replace(/\s*\/\/\s*/g, ' // '))
  }

  return Array.from(out)
}

function pickImageUrl(card) {
  if (card?.image_uris?.normal) return card.image_uris.normal
  if (card?.image_uris?.large) return card.image_uris.large
  if (Array.isArray(card?.card_faces)) {
    for (const face of card.card_faces) {
      if (face?.image_uris?.normal) return face.image_uris.normal
      if (face?.image_uris?.large) return face.image_uris.large
    }
  }
  return ''
}

async function fetchFromScryfall(cardName) {
  const url = `https://api.scryfall.com/cards/named?fuzzy=${encodeURIComponent(cardName)}`
  const res = await fetch(url)
  if (!res.ok) throw new Error('scryfall_not_found')
  const data = await res.json()
  return {
    name: data.name || cardName,
    mana_cost: data.mana_cost || '',
    type_line: data.type_line || '',
    oracle_text: data.oracle_text || '',
    image_url: pickImageUrl(data),
    source: 'scryfall',
  }
}

async function fetchFromScryfallSearch(cardName) {
  const query = `!"${cardName}"`
  const url = `https://api.scryfall.com/cards/search?q=${encodeURIComponent(query)}`
  const res = await fetch(url)
  if (!res.ok) throw new Error('scryfall_search_not_found')
  const data = await res.json()
  const first = Array.isArray(data?.data) ? data.data[0] : null
  if (!first) throw new Error('scryfall_search_empty')
  return {
    name: first.name || cardName,
    mana_cost: first.mana_cost || '',
    type_line: first.type_line || '',
    oracle_text: first.oracle_text || '',
    image_url: pickImageUrl(first),
    source: 'scryfall_search',
  }
}

async function fetchFromBackend(cardName, token) {
  if (!token) throw new Error('missing_token')
  const url = `${API}/cards/search?name=${encodeURIComponent(cardName)}`
  const res = await fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
  if (!res.ok) throw new Error('backend_not_found')
  const data = await res.json()
  return {
    name: data?.name || cardName,
    mana_cost: data?.mana_cost || '',
    type_line: data?.type_line || '',
    oracle_text: data?.oracle_text || '',
    image_url: '',
    source: 'backend',
  }
}

async function resolvePreview(cardName, token) {
  const candidates = deriveCardNameCandidates(cardName)

  for (const candidate of candidates) {
    try {
      return await fetchFromScryfall(candidate)
    } catch {
      // Continue through fallbacks.
    }
  }

  if (token) {
    for (const candidate of candidates) {
      try {
        return await fetchFromBackend(candidate, token)
      } catch {
        // Continue through fallbacks.
      }
    }
  }

  for (const candidate of candidates) {
    try {
      return await fetchFromScryfallSearch(candidate)
    } catch {
      // Continue through fallbacks.
    }
  }

  throw new Error('preview_not_found')
}

export default function CardHoverPreview({ cardName, token, messages, children }) {
  const normalized = useMemo(() => normalizeCardName(cardName), [cardName])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState(null)
  const [error, setError] = useState('')
  const [position, setPosition] = useState({ x: 0, y: 0 })

  async function loadPreview() {
    if (!normalized) return

    if (previewCache.has(normalized)) {
      const cached = previewCache.get(normalized)
      if (cached?.ok) {
        setPreview(cached.data)
        setError('')
        return
      }
      if (cached?.failedAt && Date.now()-cached.failedAt < FAILED_CACHE_TTL_MS) {
        setPreview(null)
        setError('not_found')
        return
      }
    }

    setLoading(true)
    setError('')
    try {
      const data = await resolvePreview(cardName, token)
      previewCache.set(normalized, { ok: true, data })
      setPreview(data)
    } catch {
      previewCache.set(normalized, { ok: false, failedAt: Date.now() })
      setError('not_found')
      setPreview(null)
    } finally {
      setLoading(false)
    }
  }

  function updatePosition(e) {
    const offset = 16
    const maxX = window.innerWidth - 380
    const maxY = window.innerHeight - 320
    setPosition({
      x: Math.max(8, Math.min(maxX, e.clientX + offset)),
      y: Math.max(8, Math.min(maxY, e.clientY + offset)),
    })
  }

  function openPreview(e) {
    updatePosition(e)
    setOpen(true)
    if (!preview && !loading) {
      loadPreview()
    }
  }

  const trigger = children || cardName

  return (
    <span
      style={{ cursor: 'help', textDecoration: 'underline dotted', textUnderlineOffset: 2 }}
      onMouseEnter={openPreview}
      onMouseMove={updatePosition}
      onMouseLeave={() => setOpen(false)}
      onFocus={e => openPreview(e)}
      onBlur={() => setOpen(false)}
      tabIndex={0}
      role="button"
      aria-label={messages?.cardPreviewAria ? messages.cardPreviewAria(cardName) : `Preview ${cardName}`}
    >
      {trigger}

      {open && (
        <div
          style={{
            position: 'fixed',
            left: position.x,
            top: position.y,
            width: 360,
            maxWidth: 'calc(100vw - 16px)',
            background: 'var(--card)',
            border: '1px solid var(--border)',
            borderRadius: 12,
            boxShadow: '0 10px 30px rgba(0,0,0,0.35)',
            padding: 10,
            zIndex: 9999,
            pointerEvents: 'none',
          }}
        >
          {loading && (
            <div style={{ color: 'var(--muted)', fontSize: '.82rem' }}>
              {messages?.cardPreviewLoading || 'Loading card preview...'}
            </div>
          )}

          {!loading && error && (
            <div style={{ color: 'var(--muted)', fontSize: '.82rem' }}>
              {messages?.cardPreviewUnavailable || 'Card preview unavailable'}
            </div>
          )}

          {!loading && !error && preview && (
            <div>
              <div style={{ fontWeight: 700, fontSize: '.92rem', marginBottom: 2 }}>{preview.name}</div>
              {(preview.mana_cost || preview.type_line) && (
                <div style={{ color: 'var(--muted)', fontSize: '.78rem', marginBottom: 8 }}>
                  {[preview.mana_cost, preview.type_line].filter(Boolean).join(' · ')}
                </div>
              )}
              {preview.image_url && (
                <img
                  src={preview.image_url}
                  alt={preview.name}
                  loading="lazy"
                  style={{ width: '100%', borderRadius: 8, border: '1px solid var(--border)', marginBottom: 8 }}
                />
              )}
              <div style={{ whiteSpace: 'pre-line', fontSize: '.8rem', lineHeight: 1.35, color: 'var(--text)' }}>
                {preview.oracle_text || (messages?.cardPreviewNoRules || 'No oracle text available')}
              </div>
            </div>
          )}
        </div>
      )}
    </span>
  )
}

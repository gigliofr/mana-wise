import { useEffect, useMemo, useState } from 'react'

const API = '/api/v1'
const previewCache = new Map()
const FAILED_CACHE_TTL_MS = 90 * 1000
const STICKY_PREF_KEY = 'mw_card_preview_sticky_default'

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

function extractSetCollector(rawName) {
  const text = String(rawName || '').trim()
  if (!text) return null

  const arena = text.match(/\(([A-Za-z0-9]{2,6})\)\s*([A-Za-z0-9-]+)/)
  if (arena) {
    return { setCode: arena[1].toLowerCase(), collectorNumber: arena[2].toLowerCase() }
  }
  return null
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

async function fetchFromScryfallSetCollector(setCode, collectorNumber) {
  if (!setCode || !collectorNumber) throw new Error('missing_set_collector')
  const url = `https://api.scryfall.com/cards/${encodeURIComponent(setCode)}/${encodeURIComponent(collectorNumber)}`
  const res = await fetch(url)
  if (!res.ok) throw new Error('scryfall_set_collector_not_found')
  const data = await res.json()
  return {
    name: data.name || `${setCode.toUpperCase()} ${collectorNumber}`,
    mana_cost: data.mana_cost || '',
    type_line: data.type_line || '',
    oracle_text: data.oracle_text || '',
    image_url: pickImageUrl(data),
    source: 'scryfall_set_collector',
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
  const meta = extractSetCollector(cardName)

  if (meta) {
    try {
      return await fetchFromScryfallSetCollector(meta.setCode, meta.collectorNumber)
    } catch {
      // Continue through fallbacks.
    }
  }

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

  // Final fallback: always return a minimal preview instead of hard error.
  const bestName = candidates[0] || String(cardName || '').trim() || 'Unknown card'
  return {
    name: bestName,
    mana_cost: '',
    type_line: '',
    oracle_text: '',
    image_url: '',
    source: 'fallback',
  }
}

export default function CardHoverPreview({ cardName, token, messages, children }) {
  const normalized = useMemo(() => normalizeCardName(cardName), [cardName])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState(null)
  const [error, setError] = useState('')
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const [pinned, setPinned] = useState(false)
  const [stickyDefault, setStickyDefault] = useState(() => {
    try {
      return window.localStorage.getItem(STICKY_PREF_KEY) === '1'
    } catch {
      return false
    }
  })

  useEffect(() => {
    if (!open) return
    function onKeyDown(e) {
      if (e.key === 'Escape') {
        setOpen(false)
        setPinned(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open])

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
    if (stickyDefault) {
      setPinned(true)
    }
    if (!preview && !loading) {
      loadPreview()
    }
  }

  function closePreview() {
    setOpen(false)
    setPinned(false)
  }

  const trigger = children || cardName

  return (
    <span
      style={{ cursor: 'help', textDecoration: 'underline dotted', textUnderlineOffset: 2 }}
      onMouseEnter={openPreview}
      onMouseMove={updatePosition}
      onMouseLeave={() => {
        if (!pinned && !stickyDefault) setOpen(false)
      }}
      onFocus={e => openPreview(e)}
      onBlur={() => {
        if (!pinned && !stickyDefault) setOpen(false)
      }}
      onClick={() => {
        setOpen(true)
        setPinned(true)
      }}
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
            pointerEvents: 'auto',
          }}
          onMouseEnter={() => setOpen(true)}
          onClick={e => e.stopPropagation()}
        >
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 6, marginBottom: 6 }}>
            <button
              type="button"
              onClick={e => {
                e.stopPropagation()
                setStickyDefault(prev => {
                  const next = !prev
                  try {
                    window.localStorage.setItem(STICKY_PREF_KEY, next ? '1' : '0')
                  } catch {
                    // Best effort persistence only.
                  }
                  if (next) {
                    setPinned(true)
                  }
                  return next
                })
              }}
              style={{
                fontSize: '.72rem',
                color: stickyDefault ? 'var(--green)' : 'var(--muted)',
                border: '1px solid var(--border)',
                background: 'transparent',
                borderRadius: 8,
                padding: '2px 6px',
                cursor: 'pointer',
              }}
            >
              {stickyDefault
                ? (messages?.cardPreviewStickyOn || 'Sticky ON')
                : (messages?.cardPreviewStickyOff || 'Sticky OFF')}
            </button>
            <button
              type="button"
              onClick={e => {
                e.stopPropagation()
                setPinned(v => !v)
              }}
              style={{
                fontSize: '.72rem',
                color: 'var(--muted)',
                border: '1px solid var(--border)',
                background: 'transparent',
                borderRadius: 8,
                padding: '2px 6px',
                cursor: 'pointer',
              }}
            >
              {pinned
                ? (messages?.cardPreviewUnpin || 'Unpin')
                : (messages?.cardPreviewPin || 'Pin')}
            </button>
            <button
              type="button"
              onClick={e => {
                e.stopPropagation()
                closePreview()
              }}
              style={{
                fontSize: '.72rem',
                color: 'var(--muted)',
                border: '1px solid var(--border)',
                background: 'transparent',
                borderRadius: 8,
                padding: '2px 6px',
                cursor: 'pointer',
              }}
            >
              {messages?.cardPreviewClose || 'Close'}
            </button>
          </div>

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

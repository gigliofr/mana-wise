import { useEffect, useMemo, useState } from 'react'
import { apiRequest } from '../lib/apiClient'

const previewCache = new Map()
const STICKY_PREF_KEY = 'mw_card_preview_sticky_default'
const CACHE_MAX_ENTRIES = 220
const CACHE_TTL_MS = 1000 * 60 * 60 * 24

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

function pickCardFaces(card) {
  const faces = []
  if (card?.image_uris?.normal || card?.image_uris?.large) {
    faces.push({
      name: card?.name || '',
      image_url: card?.image_uris?.normal || card?.image_uris?.large || '',
    })
  }
  if (Array.isArray(card?.card_faces)) {
    for (const face of card.card_faces) {
      const url = face?.image_uris?.normal || face?.image_uris?.large || ''
      if (!url) continue
      faces.push({
        name: face?.name || '',
        image_url: url,
      })
    }
  }

  const deduped = []
  const seen = new Set()
  for (const face of faces) {
    if (!face.image_url || seen.has(face.image_url)) continue
    seen.add(face.image_url)
    deduped.push(face)
  }
  return deduped
}

function badgeClassName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
}

function readCache(key) {
  const record = previewCache.get(key)
  if (!record) return null
  if (Date.now() - record.at > CACHE_TTL_MS) {
    previewCache.delete(key)
    return null
  }
  previewCache.delete(key)
  previewCache.set(key, { ...record, at: Date.now() })
  return record.value
}

function writeCache(key, value) {
  if (!key) return
  if (previewCache.has(key)) previewCache.delete(key)
  previewCache.set(key, { value, at: Date.now() })
  while (previewCache.size > CACHE_MAX_ENTRIES) {
    const oldestKey = previewCache.keys().next().value
    if (!oldestKey) break
    previewCache.delete(oldestKey)
  }
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
    faces: pickCardFaces(data),
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
    faces: pickCardFaces(data),
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
    faces: pickCardFaces(first),
    source: 'scryfall_search',
  }
}

async function fetchFromBackend(cardName, token) {
  if (!token) throw new Error('missing_token')
  const { res, data } = await apiRequest(`/cards/search?name=${encodeURIComponent(cardName)}`, { token })
  if (!res.ok) throw new Error('backend_not_found')
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

export default function CardHoverPreview({ cardName, token, messages, metadata, children }) {
  const normalized = useMemo(() => normalizeCardName(cardName), [cardName])
  const [isTouchMode, setIsTouchMode] = useState(false)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState(null)
  const [faceIndex, setFaceIndex] = useState(0)
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

  useEffect(() => {
    function updateMode() {
      try {
        const coarse = window.matchMedia('(pointer: coarse)').matches
        setIsTouchMode(coarse || window.innerWidth < 768)
      } catch {
        setIsTouchMode(window.innerWidth < 768)
      }
    }

    updateMode()
    window.addEventListener('resize', updateMode)
    return () => window.removeEventListener('resize', updateMode)
  }, [])

  useEffect(() => {
    setFaceIndex(0)
  }, [normalized, preview?.name])

  async function loadPreview() {
    if (!normalized) return

    const cached = readCache(normalized)
    if (cached) {
      setPreview(cached)
      setError('')
      return
    }

    setLoading(true)
    setError('')
    try {
      const data = await resolvePreview(cardName, token)
      writeCache(normalized, data)
      setPreview(data)
    } catch {
      // Last-resort UI-safe fallback: never leave the user without a preview shell.
      const fallback = {
        name: String(cardName || '').trim() || 'Unknown card',
        mana_cost: '',
        type_line: '',
        oracle_text: '',
        image_url: '',
        faces: [],
        source: 'fallback_runtime',
      }
      writeCache(normalized, fallback)
      setPreview(fallback)
      setError('')
    } finally {
      setLoading(false)
    }
  }

  function updatePosition(e) {
    if (!e || typeof e.clientX !== 'number' || typeof e.clientY !== 'number') {
      setPosition({
        x: Math.max(8, Math.round((window.innerWidth - 360) / 2)),
        y: Math.max(8, Math.round((window.innerHeight - 320) / 2)),
      })
      return
    }
    if (isTouchMode) {
      setPosition({ x: 8, y: 8 })
      return
    }
    const offset = 16
    const maxX = window.innerWidth - 380
    const maxY = window.innerHeight - 320
    setPosition({
      x: Math.max(8, Math.min(maxX, e.clientX + offset)),
      y: Math.max(8, Math.min(maxY, e.clientY + offset)),
    })
  }

  function openPreview(e) {
    if (!e) return
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
  const faces = Array.isArray(preview?.faces) ? preview.faces : []
  const hasMultipleFaces = faces.length > 1
  const shownFace = hasMultipleFaces ? faces[faceIndex % faces.length] : null
  const shownImage = shownFace?.image_url || preview?.image_url || ''
  const shownFaceName = shownFace?.name || preview?.name || ''
  const meta = metadata || {}
  const rarity = String(meta.rarity || '').trim().toUpperCase()
  const setCode = String(meta.set_code || '').trim().toUpperCase()
  const collectorNumber = String(meta.collector_number || '').trim().toUpperCase()
  const rarityClass = badgeClassName(rarity)
  const hasMetaBadges = Boolean(rarity || setCode || collectorNumber)
  const dialogStyle = isTouchMode ? {
    position: 'fixed',
    inset: 10,
    display: 'flex',
    flexDirection: 'column',
    gap: 10,
    maxHeight: 'calc(100vh - 20px)',
    overflow: 'hidden',
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    borderRadius: 18,
    boxShadow: '0 24px 80px rgba(0,0,0,0.5)',
    padding: 12,
    zIndex: 9999,
    pointerEvents: 'auto',
  } : {
    position: 'fixed',
    left: position.x,
    top: position.y,
    width: 404,
    maxWidth: 'calc(100vw - 16px)',
    maxHeight: 'calc(100vh - 16px)',
    display: 'flex',
    flexDirection: 'column',
    gap: 10,
    overflow: 'hidden',
    background: 'var(--card)',
    border: '1px solid var(--border)',
    borderRadius: 14,
    boxShadow: '0 18px 48px rgba(0,0,0,0.42)',
    padding: 12,
    zIndex: 9999,
    pointerEvents: 'auto',
  }

  return (
    <span
      style={{ cursor: 'help', textDecoration: 'underline dotted', textUnderlineOffset: 2 }}
      onMouseEnter={isTouchMode ? undefined : openPreview}
      onMouseMove={isTouchMode ? undefined : updatePosition}
      onMouseLeave={() => {
        if (!isTouchMode && !pinned && !stickyDefault) setOpen(false)
      }}
      onFocus={e => openPreview(e)}
      onBlur={() => {
        if (!isTouchMode && !pinned && !stickyDefault) setOpen(false)
      }}
      onClick={() => {
        setOpen(true)
        setPinned(true)
        if (isTouchMode) {
          setPosition({ x: 8, y: 8 })
        }
        if (!preview && !loading) {
          loadPreview()
        }
      }}
      tabIndex={0}
      role="button"
      aria-label={messages?.cardPreviewAria ? messages.cardPreviewAria(cardName) : `Preview ${cardName}`}
    >
      {trigger}

      {open && (
        <>
          {isTouchMode && (
            <div
              aria-hidden="true"
              onClick={closePreview}
              style={{
                position: 'fixed',
                inset: 0,
                background: 'rgba(0,0,0,.55)',
                zIndex: 9998,
              }}
            />
          )}
          <div
            className="card-hover-preview"
            role={isTouchMode ? 'dialog' : 'tooltip'}
            aria-modal={isTouchMode ? 'true' : undefined}
            style={dialogStyle}
            onMouseEnter={() => setOpen(true)}
            onClick={e => e.stopPropagation()}
          >
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 6, alignItems: 'center' }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, minWidth: 0 }}>
              {hasMetaBadges && (
                <>
                  {rarity && <span className={`builder-badge rarity-${rarityClass}`} title={messages?.cardPreviewRarity || 'Rarity'}>{rarity}</span>}
                  {setCode && <span className="builder-badge builder-badge-set" title={messages?.cardPreviewSet || 'Set'}>{setCode}</span>}
                  {collectorNumber && <span className="builder-badge" title={messages?.cardPreviewCollector || 'Collector number'}>{collectorNumber}</span>}
                </>
              )}
            </div>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', justifyContent: 'flex-end', marginLeft: 'auto' }}>
            {hasMultipleFaces && (
              <button
                type="button"
                onClick={e => {
                  e.stopPropagation()
                  setFaceIndex(prev => (prev + 1) % faces.length)
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
                {messages?.cardPreviewFlip || 'Flip'} ({(faceIndex % faces.length) + 1}/{faces.length})
              </button>
            )}
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
            <div style={{ minHeight: 0, overflow: 'auto', display: 'grid', gap: 10 }}>
              <div style={{ fontWeight: 700, fontSize: '.96rem', lineHeight: 1.15 }}>{preview.name}</div>
              {hasMultipleFaces && shownFaceName && shownFaceName !== preview.name && (
                <div style={{ color: 'var(--muted)', fontSize: '.76rem', marginBottom: 6 }}>
                  {messages?.cardPreviewFace || 'Face'}: {shownFaceName}
                </div>
              )}
              {(preview.mana_cost || preview.type_line) && (
                <div style={{ color: 'var(--muted)', fontSize: '.78rem', marginBottom: 8 }}>
                  {[preview.mana_cost, preview.type_line].filter(Boolean).join(' · ')}
                </div>
              )}
              {shownImage && (
                <img
                  key={`${shownImage}-${faceIndex}`}
                  src={shownImage}
                  alt={shownFaceName || preview.name}
                  loading="lazy"
                  style={{
                    width: '100%',
                    borderRadius: 12,
                    border: '1px solid var(--border)',
                    marginBottom: 0,
                    maxHeight: isTouchMode ? '46vh' : 420,
                    objectFit: 'contain',
                    background: 'rgba(0,0,0,.14)',
                    transition: 'transform .18s ease, opacity .18s ease',
                    transform: 'rotateY(0deg)',
                    opacity: 1,
                  }}
                />
              )}
              <div style={{ whiteSpace: 'pre-line', fontSize: '.84rem', lineHeight: 1.45, color: 'var(--text)' }}>
                {preview.oracle_text || (messages?.cardPreviewNoRules || 'No oracle text available')}
              </div>
            </div>
          )}
        </div>
        </>
      )}
    </span>
  )
}

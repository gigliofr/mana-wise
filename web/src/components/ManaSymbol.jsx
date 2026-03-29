import { useMemo, useState } from 'react'

const VALID_CODES = new Set(['W', 'U', 'B', 'R', 'G', 'C'])

export function isManaColorCode(value) {
  return VALID_CODES.has(String(value || '').trim().toUpperCase())
}

function normalizeCode(value) {
  return String(value || '').trim().toUpperCase()
}

function symbolURL(code) {
  return `https://svgs.scryfall.io/card-symbols/${code}.svg`
}

export function ManaSymbol({ code, size = 20 }) {
  const normalized = useMemo(() => normalizeCode(code), [code])
  const [failed, setFailed] = useState(false)

  if (!isManaColorCode(normalized) || failed) {
    return (
      <span
        aria-label={`mana-${normalized || 'unknown'}`}
        title={normalized || ''}
        style={{
          width: size,
          height: size,
          minWidth: size,
          borderRadius: '50%',
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: Math.max(10, Math.round(size * 0.5)),
          fontWeight: 700,
          border: '1px solid var(--border)',
          color: 'var(--text)',
          background: 'rgba(255,255,255,0.04)',
          lineHeight: 1,
        }}
      >
        {normalized || '?'}
      </span>
    )
  }

  return (
    <img
      src={symbolURL(normalized)}
      alt={normalized}
      title={normalized}
      loading="lazy"
      onError={() => setFailed(true)}
      style={{ width: size, height: size, minWidth: size, display: 'inline-block', verticalAlign: 'middle' }}
    />
  )
}

export function ManaSymbolGroup({ colors, size = 20, gap = 4 }) {
  const normalized = Array.isArray(colors)
    ? colors.map(c => normalizeCode(c)).filter(isManaColorCode)
    : []

  if (!normalized.length) return null

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap }}>
      {normalized.map((c, idx) => (
        <ManaSymbol key={`${c}-${idx}`} code={c} size={size} />
      ))}
    </span>
  )
}

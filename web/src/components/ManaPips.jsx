// Pallini di colore e costi di mana leggeri, senza immagini esterne.
const ORDER = ['W', 'U', 'B', 'R', 'G']

export function ColorDots({ colors = [], label, size = 'md' }) {
  const list = ORDER.filter(c => colors.includes(c))
  if (!list.length) return <span className={`mw-dots mw-dots--${size}`} aria-label={label}><span className="mw-dot mw-dot--C" /></span>
  return (
    <span className={`mw-dots mw-dots--${size}`} aria-label={label}>
      {list.map(c => <span key={c} className={`mw-dot mw-dot--${c}`} />)}
    </span>
  )
}

// "{2}{R}{R}" -> pallini; i numeri diventano un pallino neutro con la cifra.
export function ManaCost({ cost, label }) {
  const parts = String(cost || '').match(/\{[^}]+\}/g) || []
  if (!parts.length) return null
  return (
    <span className="mw-cost" aria-label={label || cost}>
      {parts.map((p, i) => {
        const v = p.slice(1, -1).toUpperCase()
        if (/^\d+$/.test(v) || v === 'X') return <span key={i} className="mw-pip mw-pip--n">{v}</span>
        const color = v.split('/')[0]
        return <span key={i} className={`mw-pip mw-pip--${ORDER.includes(color) ? color : 'C'}`} />
      })}
    </span>
  )
}

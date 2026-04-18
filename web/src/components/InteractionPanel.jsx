const CATEGORY_ICONS = {
  removal:    '🗡️',
  counter:    '🛡️',
  draw:       '📖',
  ramp:       '💎',
  protection: '✨',
  discard:    '🌀',
}

function barColor(count, ideal) {
  if (ideal <= 0) return 'var(--muted)'
  const ratio = ideal > 0 ? count / ideal : 0
  if (ratio >= 0.8) return 'var(--green)'
  if (ratio >= 0.5) return 'var(--orange)'
  return 'var(--red)'
}

function rowStatus(count, ideal, messages) {
  if (ideal <= 0) {
    return { label: messages?.rowNotRequired || 'Not required', color: 'var(--muted)', width: 100, value: messages?.naLabel || 'n/a' }
  }
  const ratio = count / ideal
  if (ratio >= 1) {
    return { label: messages?.rowGood || 'Good', color: 'var(--green)', width: 100, value: `${count}/${ideal}` }
  }
  if (ratio >= 0.5) {
    return { label: messages?.rowPartial || 'Partial', color: 'var(--orange)', width: Math.min(ratio * 100, 100), value: `${count}/${ideal}` }
  }
  return { label: messages?.rowLow || 'Low', color: 'var(--red)', width: Math.min(ratio * 100, 100), value: `${count}/${ideal}` }
}

function localizeSuggestion(s, messages, format, commanderScore) {
  if (!s) return s
  const m = s.match(/^Your\s+(\w+)\s+package\s+\((\d+)\s+cards\)\s+is\s+(?:well\s+)?below\s+ideal(?:\s+for\s+a\s+(\w+)\s+deck)?\s*\((\d+)\s+expected\)\.\s+Consider\s+adding\s+more\.$/i)
  if (!m) return s

  const category = (m[1] || '').toLowerCase()
  const count = Number(m[2] || 0)
  const archetype = (m[3] || '').toLowerCase()
  const ideal = Number(m[4] || 0)
  const idealForDisplay = adjustedIdeal(ideal, format, commanderScore)

  if (!messages?.translateSuggestion) {
    return s.replace(/\(\d+\s+expected\)/i, `(${idealForDisplay} expected)`)
  }

  return messages.translateSuggestion(category, count, archetype, idealForDisplay)
}

function commanderBracketForScore(score) {
  if (score >= 8.5) return 5
  if (score >= 6.5) return 4
  if (score >= 4.5) return 3
  if (score >= 2.5) return 2
  return 1
}

function commanderIdealMultiplier(bracket) {
  if (bracket >= 5) return 1.4
  if (bracket === 4) return 1.25
  if (bracket === 3) return 1.1
  if (bracket === 2) return 1.0
  return 0.85
}

function adjustedIdeal(ideal, format, commanderScore) {
  if (ideal <= 0) return 0
  if (format !== 'commander' || typeof commanderScore !== 'number') return ideal
  const bracket = commanderBracketForScore(commanderScore)
  const scaled = Math.round(ideal * commanderIdealMultiplier(bracket))
  return Math.max(1, scaled)
}

export default function InteractionPanel({ data }) {
  const messages = data.messages
  const labels = messages?.categoryLabels || {}

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <span style={{ fontSize: '.85rem', color: 'var(--muted)' }}>{messages?.overallScore || 'Overall score'}</span>
        <span style={{
          fontSize: '1.6rem',
          fontWeight: 700,
          color: data.total_score >= 70 ? 'var(--green)' : data.total_score >= 40 ? 'var(--yellow)' : 'var(--red)',
        }}>
          {data.total_score}
        </span>
        <span style={{ fontSize: '.8rem', color: 'var(--muted)' }}>/100</span>
      </div>
      <p style={{ fontSize: '.78rem', color: 'var(--muted)', marginBottom: 12 }}>
        {messages?.interactionReadHint || 'Each row compares current count against the archetype-adjusted target for this category.'}
      </p>

      {data.breakdowns?.map(bd => {
        const ideal = adjustedIdeal(bd.ideal, data.format, data.commanderScore)
        const status = rowStatus(bd.count, ideal, messages)
        return (
          <div key={bd.category} className="score-row">
            <span className="score-label">
              {CATEGORY_ICONS[bd.category] || '◆'} {labels[bd.category] || bd.category}
            </span>
            <div className="score-bar-track">
              <div
                className="score-bar-fill"
                style={{
                  width: `${status.width}%`,
                  background: status.color,
                  opacity: ideal <= 0 ? 0.45 : 1,
                }}
              />
            </div>
            <span className="score-count" style={{ color: status.color }}>
              {status.value}
            </span>
            <span className="score-meta" style={{ color: status.color }}>
              {status.label}
            </span>
          </div>
        )
      })}

      {data.suggestions?.length > 0 && (
        <>
          <p style={{ fontSize: '.85rem', color: 'var(--muted)', margin: '16px 0 8px' }}>{messages?.suggestions || 'Suggestions'}</p>
          <ul className="suggestion-list">
            {data.suggestions.map((s, i) => <li key={i}>{localizeSuggestion(s, messages, data.format, data.commanderScore)}</li>)}
          </ul>
        </>
      )}
    </div>
  )
}

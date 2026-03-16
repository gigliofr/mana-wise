const CMC_LABELS = ['0', '1', '2', '3', '4', '5', '6+']
const CMC_COLORS = [
  '#6c757d', '#3498db', '#2ecc71', '#e67e22', '#e74c3c', '#9b59b6', '#1abc9c',
]

export default function ManaCurveChart({ distribution, maxCount, messages }) {
  const midCount = Math.max(1, Math.ceil(maxCount / 2))

  return (
    <div className="curve-chart">
      <div className="curve-chart-head">
        <p className="curve-title">{messages?.curveTitle || 'Mana Curve Distribution'}</p>
        <p className="curve-subtitle">{messages?.curveSubtitle || 'Non-land spells by mana value. Lands are counted separately.'}</p>
        <p className="curve-subtitle">{messages?.curveReadHint || 'Top number = cards in that mana value bucket. Bottom % = share of non-land spells. Left axis = absolute card count.'}</p>
      </div>

      <div className="curve-layout">
        <div className="curve-y-axis" aria-hidden="true">
          <span>{maxCount}</span>
          <span>{midCount}</span>
          <span>0</span>
        </div>

        <div className="curve-plot">
          <div className="curve-grid-line curve-grid-top" />
          <div className="curve-grid-line curve-grid-mid" />
          <div className="curve-grid-line curve-grid-bottom" />

          <div className="curve-bars">
            {distribution.map((bucket, i) => {
              const heightPct = maxCount > 0 ? (bucket.count / maxCount) * 100 : 0
              const sharePct = distribution.length > 0
                ? Math.round((bucket.count / Math.max(distribution.reduce((sum, item) => sum + item.count, 0), 1)) * 100)
                : 0

              return (
                <div key={bucket.cmc} className="curve-bar-wrap">
                  <span className="curve-bar-count">{bucket.count}</span>
                  <div className="curve-bar-slot">
                    <div
                      className="curve-bar"
                      style={{
                        height: `${heightPct}%`,
                        background: CMC_COLORS[i % CMC_COLORS.length],
                        minHeight: bucket.count > 0 ? '10px' : '0',
                      }}
                      title={`CMC ${CMC_LABELS[i]}: ${bucket.count} cards (${sharePct}%)`}
                    />
                  </div>
                  <span className="curve-bar-label">{CMC_LABELS[i]}</span>
                  <span className="curve-bar-share">{bucket.count > 0 ? `${sharePct}%` : ''}</span>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}

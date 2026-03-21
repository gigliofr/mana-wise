const PAYPAL_DONATE_URL = import.meta.env.VITE_PAYPAL_DONATE_URL || 'https://paypal.me/gigliofr'

export default function PlansSupport({ messages }) {
  const planRows = [
    {
      label: messages.planFeatureAnalyses,
      free: messages.planFeatureAnalysesFree,
      pro: messages.planFeatureAnalysesPro,
    },
    {
      label: messages.planFeatureDeckSlots,
      free: messages.planFeatureDeckSlotsFree,
      pro: messages.planFeatureDeckSlotsPro,
    },
    {
      label: messages.planFeatureTools,
      free: messages.planFeatureToolsFree,
      pro: messages.planFeatureToolsPro,
    },
  ]

  function donateCoffee() {
    window.open(PAYPAL_DONATE_URL, '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="card plan-support-card">
      <h2>{messages.planSectionTitle}</h2>
      <p className="plan-support-subtitle">{messages.planSectionSubtitle}</p>

      <div className="plan-grid">
        <div className="plan-box free">
          <div className="plan-box-head">
            <strong>{messages.planFreeTitle}</strong>
            <span>{messages.planFreeBadge}</span>
          </div>
          <ul className="plan-list">
            {planRows.map(item => <li key={`free-${item.label}`}><strong>{item.label}:</strong> {item.free}</li>)}
          </ul>
        </div>

        <div className="plan-box pro">
          <div className="plan-box-head">
            <strong>{messages.planProTitle}</strong>
            <span>{messages.planProBadge}</span>
          </div>
          <ul className="plan-list">
            {planRows.map(item => <li key={`pro-${item.label}`}><strong>{item.label}:</strong> {item.pro}</li>)}
          </ul>
        </div>
      </div>

      <div className="donate-box">
        <div>
          <strong>{messages.donateTitle}</strong>
          <p>{messages.donateBody}</p>
        </div>
        <button type="button" className="btn-primary" onClick={donateCoffee}>
          {messages.donateButton}
        </button>
      </div>
    </div>
  )
}

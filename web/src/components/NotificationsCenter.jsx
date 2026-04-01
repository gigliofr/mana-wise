import { useEffect, useMemo, useState } from 'react'

const API = '/api/v1'

export default function NotificationsCenter({ token, locale, messages }) {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [reloadTick, setReloadTick] = useState(0)

  const dateFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale === 'en' ? 'en-GB' : 'it-IT', {
      dateStyle: 'short',
      timeStyle: 'short',
    }),
    [locale],
  )

  useEffect(() => {
    let cancelled = false

    async function loadNotifications() {
      setLoading(true)
      setError('')
      try {
        const res = await fetch(`${API}/users/me/notifications`, {
          headers: { Authorization: `Bearer ${token}` },
        })
        const data = await res.json()
        if (!res.ok) throw new Error(data?.error || messages.notificationsLoadFailed || 'Failed to load notifications')
        if (!cancelled) {
          setItems(Array.isArray(data?.items) ? data.items : [])
        }
      } catch (err) {
        if (!cancelled) {
          setError(err?.message || messages.notificationsLoadFailed || 'Failed to load notifications')
          setItems([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    if (token) loadNotifications()
    return () => {
      cancelled = true
    }
  }, [token, reloadTick, messages.notificationsLoadFailed])

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 12 }}>
        <h2 style={{ margin: 0 }}>🔔 {messages.notificationsTitle || 'Notifiche'}</h2>
        <button
          type="button"
          className="btn-ghost"
          onClick={() => setReloadTick(v => v + 1)}
          disabled={loading}
        >
          {messages.refreshNotifications || 'Aggiorna'}
        </button>
      </div>

      {loading && <div style={{ color: 'var(--muted)', fontSize: '.9rem' }}>{messages.loading}</div>}
      {!loading && error && <div className="banner banner-error">{error}</div>}

      {!loading && !error && items.length === 0 && (
        <div style={{ color: 'var(--muted)', fontSize: '.9rem' }}>
          {messages.noNotificationsYet || 'Nessuna notifica al momento.'}
        </div>
      )}

      {!loading && !error && items.length > 0 && (
        <div style={{ display: 'grid', gap: 10 }}>
          {items.map((it, idx) => {
            const created = it?.created_at ? new Date(it.created_at) : null
            const createdLabel = created && !Number.isNaN(created.getTime())
              ? dateFormatter.format(created)
              : ''
            return (
              <div
                key={`${it?.deck_id || 'global'}-${it?.card || 'card'}-${idx}`}
                style={{
                  border: '1px solid var(--border)',
                  borderRadius: 10,
                  padding: '10px 12px',
                  background: 'rgba(255,255,255,0.02)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'center' }}>
                  <strong>{it?.card || '-'}</strong>
                  <span style={{ fontSize: '.78rem', color: 'var(--muted)' }}>
                    {createdLabel}
                  </span>
                </div>
                <div style={{ marginTop: 6, fontSize: '.92rem' }}>{it?.message || '-'}</div>
                {it?.replacement_suggestion && (
                  <div style={{ marginTop: 6, fontSize: '.86rem', color: 'var(--muted)' }}>
                    {it.replacement_suggestion}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

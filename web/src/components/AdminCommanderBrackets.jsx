import { useEffect, useState } from 'react'

const SECRET_KEY = 'manawise_admin_secret'

export default function AdminCommanderBrackets({ token, user, messages }) {
  const [secret, setSecret] = useState(() => localStorage.getItem(SECRET_KEY) || '')
  const [rawConfig, setRawConfig] = useState('')
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function loadConfig() {
    setError('')
    setStatus('')
    setLoading(true)
    try {
      const res = await fetch('/api/v1/admin/commander-brackets', {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
          'X-Admin-Secret': secret.trim(),
        },
      })
      const data = await res.json().catch(() => null)
      if (!res.ok) {
        throw new Error(data?.error || 'Could not load commander bracket config')
      }
      const next = JSON.stringify(data?.config || {}, null, 2)
      setRawConfig(next)
      setStatus('Configuration loaded')
      localStorage.setItem(SECRET_KEY, secret.trim())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function saveConfig() {
    setError('')
    setStatus('')
    setLoading(true)
    try {
      const config = JSON.parse(rawConfig)
      const res = await fetch('/api/v1/admin/commander-brackets', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
          'X-Admin-Secret': secret.trim(),
        },
        body: JSON.stringify({ config }),
      })
      const data = await res.json().catch(() => null)
      if (!res.ok) {
        throw new Error(data?.error || 'Could not save commander bracket config')
      }
      setRawConfig(JSON.stringify(data?.config || config, null, 2))
      setStatus('Configuration saved')
      localStorage.setItem(SECRET_KEY, secret.trim())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (secret.trim()) {
      void loadConfig()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const isAdminUser = String(user?.email || '').toLowerCase().includes('admin')

  if (!isAdminUser) {
    return (
      <div className="card">
        <h2>{messages.adminHiddenTitle || 'Admin Console'}</h2>
        <p style={{ color: 'var(--muted)' }}>{messages.adminHiddenBody || 'This area is reserved for administrative users.'}</p>
      </div>
    )
  }

  return (
    <div className="card">
      <h2>{messages.adminCommanderTitle || 'Commander Brackets Admin'}</h2>
      <p style={{ color: 'var(--muted)', marginTop: 0 }}>
        {messages.adminCommanderBody || 'Update bracket rules, decisive cards and keyword lists without redeploying.'}
      </p>

      <div className="form-row">
        <label>{messages.adminSecretLabel || 'Admin secret'}</label>
        <input
          type="password"
          value={secret}
          onChange={e => setSecret(e.target.value)}
          placeholder={messages.adminSecretPlaceholder || 'X-Admin-Secret value'}
        />
      </div>

      <div className="form-row">
        <label>{messages.adminCommanderJsonLabel || 'Commander bracket JSON'}</label>
        <textarea
          value={rawConfig}
          onChange={e => setRawConfig(e.target.value)}
          spellCheck={false}
          style={{ minHeight: 320, fontFamily: 'monospace' }}
          placeholder={messages.adminCommanderJsonPlaceholder || '{ "config": { ... } }'}
        />
      </div>

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <button type="button" className="btn-ghost" onClick={loadConfig} disabled={loading || !secret.trim()}>
          {messages.adminLoadConfig || 'Load config'}
        </button>
        <button type="button" className="btn-primary" onClick={saveConfig} disabled={loading || !secret.trim()}>
          {messages.adminSaveConfig || 'Save config'}
        </button>
      </div>

      {status && <div className="banner banner-info" style={{ marginTop: 16 }}>{status}</div>}
      {error && <div className="banner banner-error" style={{ marginTop: 16 }}>{error}</div>}
    </div>
  )
}
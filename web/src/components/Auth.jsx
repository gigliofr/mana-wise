import { useState } from 'react'
import { LOCALES } from '../i18n'

const API = '/api/v1'

export default function Auth({ onLogin, locale, messages, onLocaleChange }) {
  const [mode, setMode] = useState('login') // 'login' | 'register'
  const [form, setForm] = useState({ email: '', password: '', name: '' })
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function set(field) {
    return e => setForm(f => ({ ...f, [field]: e.target.value }))
  }

  async function submit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const endpoint = mode === 'login' ? `${API}/auth/login` : `${API}/auth/register`
      const body = mode === 'login'
        ? { email: form.email, password: form.password }
        : { email: form.email, password: form.password, name: form.name }

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Request failed')
      onLogin(data.token, data.user)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <main>
      <div className="container">
        <div className="auth-wrap">
          <h1>🔮 ManaWise AI</h1>
          <p>{messages.appTagline}</p>

          <div className="locale-switch auth-locale-switch" aria-label="Language switcher">
            {LOCALES.map(item => (
              <button
                key={item.code}
                type="button"
                className={`locale-btn${locale === item.code ? ' active' : ''}`}
                onClick={() => onLocaleChange(item.code)}
                title={item.label}
              >
                <span>{item.flag}</span>
                <span>{item.label}</span>
              </button>
            ))}
          </div>

          <div className="card">
            <h2>{mode === 'login' ? messages.signIn : messages.createAccount}</h2>

            {error && <div className="banner banner-error">{error}</div>}

            <form onSubmit={submit}>
              {mode === 'register' && (
                <div className="form-row">
                  <label>{messages.name}</label>
                  <input value={form.name} onChange={set('name')} placeholder={messages.yourName} required />
                </div>
              )}
              <div className="form-row">
                <label>{messages.email}</label>
                <input type="email" value={form.email} onChange={set('email')} placeholder="you@example.com" required />
              </div>
              <div className="form-row">
                <label>{messages.password}</label>
                <div className="password-input-wrap">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={form.password}
                    onChange={set('password')}
                    placeholder="••••••••"
                    required
                    minLength={8}
                  />
                  <button
                    type="button"
                    className="btn-ghost password-toggle-btn"
                    onClick={() => setShowPassword(v => !v)}
                    aria-label={showPassword ? messages.hidePassword : messages.showPassword}
                  >
                    {showPassword ? messages.hidePassword : messages.showPassword}
                  </button>
                </div>
              </div>
              <button className="btn-primary" type="submit" disabled={loading} style={{ width: '100%' }}>
                {loading ? messages.loading : mode === 'login' ? messages.signIn : messages.createAccount}
              </button>
            </form>

            <div className="auth-toggle">
              {mode === 'login'
                ? <>{messages.noAccount} <button onClick={() => setMode('register')}>{messages.signUpFree}</button></>
                : <>{messages.haveAccount} <button onClick={() => setMode('login')}>{messages.signIn}</button></>}
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}

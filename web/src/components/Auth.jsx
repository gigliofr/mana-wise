import { useState } from 'react'
import { LOCALES } from '../i18n'
import { apiRequest, throwIfNotOK } from '../lib/apiClient'

const PASSWORD_RULES = [
  { key: 'length', test: password => password.length >= 8, labelKey: 'passwordStrengthLength' },
  { key: 'lowercase', test: password => /[a-z]/.test(password), labelKey: 'passwordStrengthLowercase' },
  { key: 'uppercase', test: password => /[A-Z]/.test(password), labelKey: 'passwordStrengthUppercase' },
  { key: 'number', test: password => /\d/.test(password), labelKey: 'passwordStrengthNumber' },
  { key: 'symbol', test: password => /[^A-Za-z0-9]/.test(password), labelKey: 'passwordStrengthSymbol' },
]

function getPasswordStrength(password) {
  const checks = PASSWORD_RULES.reduce((acc, rule) => {
    acc[rule.key] = rule.test(password)
    return acc
  }, {})
  const score = Object.values(checks).filter(Boolean).length
  const level = score >= 5 ? 'strong' : score >= 3 ? 'medium' : 'weak'
  return { checks, level, score }
}

export default function Auth({ onLogin, locale, messages, onLocaleChange }) {
  const [mode, setMode] = useState(() => {
    const token = new URLSearchParams(window.location.search).get('token')
    return token ? 'reset' : 'login'
  }) // 'login' | 'register' | 'forgot' | 'reset'
  const [form, setForm] = useState({ email: '', password: '', name: '' })
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  function set(field) {
    return e => setForm(f => ({ ...f, [field]: e.target.value }))
  }

  const passwordStrength = mode === 'register' ? getPasswordStrength(form.password) : null
  const passwordStrengthWidth = passwordStrength
    ? (passwordStrength.score === 0 ? 0 : Math.max(20, (passwordStrength.score / PASSWORD_RULES.length) * 100))
    : 0

  async function submit(e) {
    e.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)
    try {
      let endpoint = '/auth/login'
      let body = { email: form.email, password: form.password }

      if (mode === 'register') {
        if (form.password !== confirmPassword) {
          throw new Error(messages.passwordMismatch || 'Passwords do not match')
        }
        endpoint = '/auth/register'
        body = { email: form.email, password: form.password, name: form.name }
      }

      if (mode === 'forgot') {
        endpoint = '/auth/forgot-password'
        body = { email: form.email }
      }

      if (mode === 'reset') {
        if (form.password !== confirmPassword) {
          throw new Error(messages.passwordMismatch || 'Passwords do not match')
        }
        const token = new URLSearchParams(window.location.search).get('token') || ''
        if (!token) {
          throw new Error(messages.resetTokenMissing || 'Reset token is missing')
        }
        endpoint = '/auth/reset-password'
        body = { token, new_password: form.password }
      }

      const { res, data } = await apiRequest(endpoint, {
        method: 'POST',
        body,
      })
      throwIfNotOK(res, data, 'Request failed')

      if (mode === 'login' || mode === 'register') {
        if (!data?.token || !data?.user) {
          throw new Error('Request failed')
        }
        onLogin(data.token, data.user)
        return
      }

      if (mode === 'forgot') {
        setSuccess(data?.message || messages.resetLinkSent)
        return
      }

      if (mode === 'reset') {
        setSuccess(messages.resetPasswordSuccess)
        setMode('login')
        setForm(f => ({ ...f, password: '' }))
        setConfirmPassword('')
      }
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
            <h2>
              {mode === 'login' && messages.signIn}
              {mode === 'register' && messages.createAccount}
              {mode === 'forgot' && messages.forgotPasswordTitle}
              {mode === 'reset' && messages.resetPasswordTitle}
            </h2>

            {error && <div className="banner banner-error">{error}</div>}
            {success && <div className="banner banner-info">{success}</div>}

            <form onSubmit={submit}>
              {mode === 'register' && (
                <>
                  <div className="form-row">
                    <label>{messages.name}</label>
                    <input value={form.name} onChange={set('name')} placeholder={messages.yourName} required />
                  </div>
                  <p style={{ fontSize: '.82rem', color: 'var(--muted)', marginBottom: 12 }}>{messages.proActivationFromPlansNote}</p>
                </>
              )}
              {(mode === 'login' || mode === 'register' || mode === 'forgot') && (
                <div className="form-row">
                  <label>{messages.email}</label>
                  <input type="email" value={form.email} onChange={set('email')} placeholder="you@example.com" required />
                </div>
              )}

              {(mode === 'login' || mode === 'register' || mode === 'reset') && (
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
                  {mode === 'register' && (
                    <div className={`password-strength password-strength-${passwordStrength.level}`} aria-live="polite">
                      <div className="password-strength-head">
                        <span>{messages.passwordStrengthLabel || 'Password strength'}</span>
                        <strong>{messages[`passwordStrength${passwordStrength.level.charAt(0).toUpperCase()}${passwordStrength.level.slice(1)}`] || passwordStrength.level}</strong>
                      </div>
                      <div className="password-strength-bar" aria-hidden="true">
                        <span style={{ width: `${passwordStrengthWidth}%` }} />
                      </div>
                      <ul className="password-strength-list">
                        {PASSWORD_RULES.map(rule => (
                          <li key={rule.key} className={passwordStrength.checks[rule.key] ? 'met' : ''}>
                            <span aria-hidden="true">{passwordStrength.checks[rule.key] ? '✓' : '•'}</span>
                            <span>{messages[rule.labelKey] || rule.labelKey}</span>
                          </li>
                        ))}
                      </ul>
                      <p className="password-strength-hint">{messages.passwordStrengthHint || 'Use a mix of uppercase and lowercase letters, numbers and symbols.'}</p>
                    </div>
                  )}
                </div>
              )}

              {(mode === 'register' || mode === 'reset') && (
                <div className="form-row">
                  <label>{messages.confirmPassword}</label>
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={confirmPassword}
                    onChange={e => setConfirmPassword(e.target.value)}
                    placeholder="••••••••"
                    required
                    minLength={8}
                  />
                </div>
              )}

              <button className="btn-primary" type="submit" disabled={loading} style={{ width: '100%' }}>
                {loading ? messages.loading : (
                  mode === 'login' ? messages.signIn :
                  mode === 'register' ? messages.createAccount :
                  mode === 'forgot' ? messages.sendResetLink :
                  messages.resetPasswordAction
                )}
              </button>

              {mode === 'login' && (
                <button
                  type="button"
                  className="btn-ghost"
                  onClick={() => {
                    setMode('forgot')
                    setError('')
                    setSuccess('')
                  }}
                  style={{ width: '100%', marginTop: 8 }}
                >
                  {messages.forgotPasswordCta}
                </button>
              )}

              {(mode === 'forgot' || mode === 'reset') && (
                <button
                  type="button"
                  className="btn-ghost"
                  onClick={() => {
                    setMode('login')
                    setError('')
                    setSuccess('')
                  }}
                  style={{ width: '100%', marginTop: 8 }}
                >
                  {messages.backToSignIn}
                </button>
              )}
            </form>

            {(mode === 'login' || mode === 'register') && (
              <div className="auth-toggle">
                {mode === 'login'
                  ? <>{messages.noAccount} <button onClick={() => setMode('register')}>{messages.signUpFree}</button></>
                  : <>{messages.haveAccount} <button onClick={() => setMode('login')}>{messages.signIn}</button></>}
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  )
}

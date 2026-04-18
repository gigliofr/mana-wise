const API_BASE = '/api/v1'

const sessionHandlers = {
  getToken: null,
  getRefreshToken: null,
  onSessionUpdate: null,
  onUnauthorized: null,
}

export function configureApiAuthSession(handlers = {}) {
  sessionHandlers.getToken = typeof handlers.getToken === 'function' ? handlers.getToken : null
  sessionHandlers.getRefreshToken = typeof handlers.getRefreshToken === 'function' ? handlers.getRefreshToken : null
  sessionHandlers.onSessionUpdate = typeof handlers.onSessionUpdate === 'function' ? handlers.onSessionUpdate : null
  sessionHandlers.onUnauthorized = typeof handlers.onUnauthorized === 'function' ? handlers.onUnauthorized : null
}

export function __resetApiClientAuthSessionForTests() {
  configureApiAuthSession()
}

function parseMaybeJSON(res) {
  const contentType = String(res.headers.get('content-type') || '').toLowerCase()
  if (!contentType.includes('application/json')) {
    return null
  }
  return res.json().catch(() => null)
}

export async function apiRequest(path, options = {}) {
  const {
    token,
    method = 'GET',
    headers = {},
    body,
    skipAuthRefresh = false,
  } = options

  async function performRequest(authToken) {
    const finalHeaders = { ...headers }
    if (authToken) {
      finalHeaders.Authorization = `Bearer ${authToken}`
    }

    let requestBody = body
    if (body !== undefined && body !== null && typeof body === 'object' && !(body instanceof FormData)) {
      if (!finalHeaders['Content-Type'] && !finalHeaders['content-type']) {
        finalHeaders['Content-Type'] = 'application/json'
      }
      requestBody = JSON.stringify(body)
    }

    const res = await fetch(`${API_BASE}${path}`, {
      method,
      headers: finalHeaders,
      body: requestBody,
    })
    const data = await parseMaybeJSON(res)
    return { res, data }
  }

  const initialToken = token || sessionHandlers.getToken?.() || ''
  const initialRefreshToken = sessionHandlers.getRefreshToken?.() || ''
  let result = await performRequest(initialToken)

  const refreshablePath = !path.startsWith('/auth/login')
    && !path.startsWith('/auth/register')
    && !path.startsWith('/auth/forgot-password')
    && !path.startsWith('/auth/reset-password')
    && !path.startsWith('/auth/refresh')

  if (!skipAuthRefresh && initialToken && initialRefreshToken && refreshablePath && result.res.status === 401) {
    const refreshRes = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: initialRefreshToken }),
    })
    const refreshData = await parseMaybeJSON(refreshRes)

    if (refreshRes.ok && refreshData?.token && refreshData?.refresh_token) {
      const hasSessionTTL = Number(refreshData?.session_ttl_minutes || 0) > 0
      if (hasSessionTTL) {
        sessionHandlers.onSessionUpdate?.(
          refreshData.token,
          refreshData.refresh_token,
          refreshData.user,
          {
            session_ttl_minutes: Number(refreshData.session_ttl_minutes),
            session_expires_at: String(refreshData?.session_expires_at || ''),
          },
        )
      } else {
        sessionHandlers.onSessionUpdate?.(refreshData.token, refreshData.refresh_token, refreshData.user)
      }
      result = await performRequest(refreshData.token)
    } else {
      sessionHandlers.onUnauthorized?.()
    }
  }

  return result
}

export function throwIfNotOK(res, data, fallbackMessage = 'Request failed') {
  if (res.ok) return
  const message = data?.error || fallbackMessage
  throw new Error(message)
}

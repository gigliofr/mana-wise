const API_BASE = '/api/v1'

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
  } = options

  const finalHeaders = { ...headers }
  if (token) {
    finalHeaders.Authorization = `Bearer ${token}`
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

export function throwIfNotOK(res, data, fallbackMessage = 'Request failed') {
  if (res.ok) return
  const message = data?.error || fallbackMessage
  throw new Error(message)
}

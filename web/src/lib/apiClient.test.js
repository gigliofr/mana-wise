import { afterEach, describe, expect, it, vi } from 'vitest'
import { __resetApiClientAuthSessionForTests, apiRequest, configureApiAuthSession, logoutSession, throwIfNotOK } from './apiClient'

describe('apiClient', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    __resetApiClientAuthSessionForTests()
  })

  it('serializes JSON bodies and attaches auth headers', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({ ok: true }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const { res, data } = await apiRequest('/cards', {
      token: 'token-123',
      method: 'POST',
      body: { name: 'Lightning Bolt' },
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/cards', {
      method: 'POST',
      headers: {
        Authorization: 'Bearer token-123',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: 'Lightning Bolt' }),
    })
    expect(res.ok).toBe(true)
    expect(data).toEqual({ ok: true })
  })

  it('returns null data for non-json responses', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'text/plain' },
      json: vi.fn(),
    })
    vi.stubGlobal('fetch', fetchMock)

    const { data } = await apiRequest('/status')

    expect(data).toBeNull()
  })

  it('throws the API error message when a response is not ok', () => {
    expect(() => throwIfNotOK({ ok: false }, { error: 'Bad request' })).toThrow('Bad request')
    expect(() => throwIfNotOK({ ok: false }, null, 'Fallback message')).toThrow('Fallback message')
  })

  it('migrates a legacy localStorage refresh token on 401', async () => {
    const onSessionUpdate = vi.fn()
    configureApiAuthSession({
      getToken: () => 'expired-token',
      getRefreshToken: () => 'refresh-token-1',
      onSessionUpdate,
      onUnauthorized: vi.fn(),
    })

    const fetchMock = vi.fn()
      .mockResolvedValueOnce({
        status: 401,
        ok: false,
        headers: { get: () => 'application/json' },
        json: vi.fn().mockResolvedValue({ error: 'invalid or expired token' }),
      })
      .mockResolvedValueOnce({
        status: 200,
        ok: true,
        headers: { get: () => 'application/json' },
        json: vi.fn().mockResolvedValue({ token: 'new-token', user: { id: 'u1' } }),
      })
      .mockResolvedValueOnce({
        status: 200,
        ok: true,
        headers: { get: () => 'application/json' },
        json: vi.fn().mockResolvedValue({ ok: true }),
      })
    vi.stubGlobal('fetch', fetchMock)

    const { res, data } = await apiRequest('/decks')

    expect(res.ok).toBe(true)
    expect(data).toEqual({ ok: true })
    expect(onSessionUpdate).toHaveBeenCalledWith('new-token', '', { id: 'u1' })
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/auth/refresh', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: 'refresh-token-1' }),
    })
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/v1/decks', {
      method: 'GET',
      headers: {
        Authorization: 'Bearer new-token',
      },
      body: undefined,
    })
  })

  it('refreshes through the HttpOnly cookie without sending any token', async () => {
    const onSessionUpdate = vi.fn()
    configureApiAuthSession({ getToken: () => 'expired-token', getRefreshToken: () => '', onSessionUpdate, onUnauthorized: vi.fn() })
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ status: 401, ok: false, headers: { get: () => 'application/json' }, json: vi.fn().mockResolvedValue({}) })
      .mockResolvedValueOnce({ status: 200, ok: true, headers: { get: () => 'application/json' }, json: vi.fn().mockResolvedValue({ token: 't2', user: { id: 'u1' } }) })
      .mockResolvedValueOnce({ status: 200, ok: true, headers: { get: () => 'application/json' }, json: vi.fn().mockResolvedValue({ ok: true }) })
    vi.stubGlobal('fetch', fetchMock)

    const { res } = await apiRequest('/decks')

    expect(res.ok).toBe(true)
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/auth/refresh', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    })
    expect(onSessionUpdate).toHaveBeenCalledWith('t2', '', { id: 'u1' })
  })

  it('ends the session as expired when the refresh fails', async () => {
    const onUnauthorized = vi.fn()
    configureApiAuthSession({ getToken: () => '', getRefreshToken: () => '', onUnauthorized })
    const fetchMock = vi.fn().mockResolvedValue({
      status: 401,
      ok: false,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({ error: 'invalid or expired token' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const { res } = await apiRequest('/decks', { token: 'stale-token', method: 'POST', body: {} })

    expect(res.status).toBe(401)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/auth/refresh')
    expect(onUnauthorized).toHaveBeenCalledWith('expired')
  })

  it('logoutSession posts to the logout endpoint and never throws', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('offline'))
    vi.stubGlobal('fetch', fetchMock)
    await expect(logoutSession()).resolves.toBeNull()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', { method: 'POST', credentials: 'same-origin' })
  })

  it('does not end the session for a 401 on the login endpoint', async () => {
    const onUnauthorized = vi.fn()
    configureApiAuthSession({ getToken: () => 'old', getRefreshToken: () => '', onUnauthorized })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ status: 401, ok: false, headers: { get: () => '' } }))

    await apiRequest('/auth/login', { method: 'POST', body: {} })

    expect(onUnauthorized).not.toHaveBeenCalled()
  })
})

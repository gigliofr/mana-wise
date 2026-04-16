import { afterEach, describe, expect, it, vi } from 'vitest'
import { __resetApiClientAuthSessionForTests, apiRequest, configureApiAuthSession, throwIfNotOK } from './apiClient'

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

  it('refreshes session and retries once on 401', async () => {
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
        json: vi.fn().mockResolvedValue({ token: 'new-token', refresh_token: 'refresh-token-2', user: { id: 'u1' } }),
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
    expect(onSessionUpdate).toHaveBeenCalledWith('new-token', 'refresh-token-2', { id: 'u1' })
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/auth/refresh', {
      method: 'POST',
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
})
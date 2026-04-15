import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiRequest, throwIfNotOK } from './apiClient'

describe('apiClient', () => {
  afterEach(() => {
    vi.restoreAllMocks()
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
})
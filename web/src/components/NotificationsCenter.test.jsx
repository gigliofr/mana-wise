import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import NotificationsCenter from './NotificationsCenter'

const messages = {
  notificationsTitle: 'Notifications',
  refreshNotifications: 'Refresh',
  notificationsLoadFailed: 'Failed to load notifications',
  noNotificationsYet: 'No notifications yet',
  loading: 'Loading...',
}

describe('NotificationsCenter', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads and renders notifications', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({
        items: [
          {
            card: 'Sol Ring',
            message: 'Price changed',
            replacement_suggestion: 'Try Arcane Signet',
            created_at: '2026-04-15T09:00:00.000Z',
            deck_id: 'deck-1',
          },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<NotificationsCenter token="token-123" locale="en" messages={messages} />)

    await waitFor(() => {
      expect(screen.getByText('Sol Ring')).toBeInTheDocument()
      expect(screen.getByText('Price changed')).toBeInTheDocument()
      expect(screen.getByText('Try Arcane Signet')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/me/notifications', {
      method: 'GET',
      headers: {
        Authorization: 'Bearer token-123',
      },
      body: undefined,
    })
  })

  it('shows empty state when API returns no items', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({ items: [] }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<NotificationsCenter token="token-123" locale="en" messages={messages} />)

    await waitFor(() => {
      expect(screen.getByText('No notifications yet')).toBeInTheDocument()
    })
  })

  it('shows API error and can reload', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: false,
        headers: { get: () => 'application/json' },
        json: vi.fn().mockResolvedValue({ error: 'Backend down' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        headers: { get: () => 'application/json' },
        json: vi.fn().mockResolvedValue({
          items: [{ card: 'Island', message: 'Recovered', created_at: '2026-04-15T10:00:00.000Z' }],
        }),
      })
    vi.stubGlobal('fetch', fetchMock)

    render(<NotificationsCenter token="token-123" locale="en" messages={messages} />)

    await waitFor(() => {
      expect(screen.getByText('Backend down')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => {
      expect(screen.getByText('Island')).toBeInTheDocument()
      expect(screen.getByText('Recovered')).toBeInTheDocument()
    })
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import MulliganAssistant from './MulliganAssistant'

const messages = {
  mulliganTitle: 'Mulligan Assistant',
  decklist: 'Decklist',
  decklistHint: '4 Lightning Bolt',
  selectADeck: 'Select a deck',
  loading: 'Loading...',
  loadSavedDeck: 'Load saved deck',
  format: 'Format',
  archetype: 'Archetype',
  optional: 'optional',
  archetypeLabel: a => a,
  autoDetect: 'Auto detect',
  iterations: 'Iterations',
  onPlay: 'On play',
  simulating: 'Simulating',
  runMulligan: 'Run mulligan',
  mulliganFailed: 'Mulligan failed',
  recommendation: 'Recommendation',
  handSizeSummaries: 'Hand size summaries',
  handSize: 'Hand size',
  keepRate: 'Keep rate',
  avgLands: 'Avg lands',
  avgEarlyPlays: 'Avg early plays',
}

function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'application/json' },
    json: vi.fn().mockResolvedValue(body),
  }
}

describe('MulliganAssistant', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('runs simulation and renders recommendation', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/mulligan/simulate' && options.method === 'POST') {
        return jsonResponse({
          recommendation: 'Keep 7 on the play',
          iterations: 1000,
          summaries: [
            { hand_size: 7, keep_rate: 0.64, avg_lands: 2.5, avg_early_plays: 1.8 },
          ],
        })
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MulliganAssistant
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /run mulligan/i }))

    await waitFor(() => {
      expect(screen.getByText(/Recommendation:/)).toBeInTheDocument()
      expect(screen.getByText('Keep 7 on the play')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/mulligan/simulate', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('shows backend error message on failure', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/mulligan/simulate' && options.method === 'POST') {
        return jsonResponse({ error: 'Simulation not available' }, 500)
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MulliganAssistant
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /run mulligan/i }))

    await waitFor(() => {
      expect(screen.getByText('Simulation not available')).toBeInTheDocument()
    })
  })

  it('handles selecting saved deck with missing cards array', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([
          {
            id: 'deck-1',
            user_id: 'user-1',
            name: 'Deck without cards',
            format: 'pioneer',
          },
        ])
      }
      if (url === '/api/v1/mulligan/simulate' && options.method === 'POST') {
        return jsonResponse({ recommendation: 'Keep', iterations: 1000, summaries: [] })
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MulliganAssistant
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('Deck without cards (pioneer)')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue('Load saved deck'), { target: { value: 'deck-1' } })

    await waitFor(() => {
      expect(screen.getByDisplayValue('Pioneer')).toBeInTheDocument()
    })
  })
})

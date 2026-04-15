import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import MatchupSimulator from './MatchupSimulator'

function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'application/json' },
    json: vi.fn().mockResolvedValue(body),
  }
}

const messages = {
  matchupTitle: 'Matchup Simulator',
  decklist: 'Decklist',
  decklistHint: '1 Card Name',
  loadSavedDeck: 'Load saved deck',
  selectADeck: 'Select a deck',
  sideboardDecklist: 'Sideboard decklist',
  optional: 'optional',
  sideboardHint: 'Optional sideboard',
  format: 'Format',
  opponents: 'Opponents',
  archetypeLabel: a => a,
  onPlay: 'On play',
  simulating: 'Simulating',
  runSimulation: 'Run simulation',
  matchupFailed: 'Matchup failed',
}

describe('MatchupSimulator', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads and selects owned saved deck', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([
          {
            id: 'deck-1',
            user_id: 'user-1',
            name: 'Owned Deck',
            format: 'pioneer',
            cards: [{ card_name: 'Fable of the Mirror-Breaker', quantity: 4 }],
          },
          {
            id: 'deck-2',
            user_id: 'user-2',
            name: 'Other Deck',
            format: 'modern',
            cards: [{ card_name: 'Island', quantity: 10 }],
          },
        ])
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MatchupSimulator
        token="token-123"
        user={{ id: 'user-1' }}
        decklist=""
        format="standard"
        messages={messages}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('Owned Deck (pioneer)')).toBeInTheDocument()
      expect(screen.queryByText('Other Deck (modern)')).not.toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue('Select a deck'), { target: { value: 'deck-1' } })

    await waitFor(() => {
      expect(screen.getByDisplayValue('4 Fable of the Mirror-Breaker')).toBeInTheDocument()
      expect(screen.getByDisplayValue('Pioneer')).toBeInTheDocument()
    })
  })

  it('submits simulation payload and renders backend error', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/matchup/simulate' && options.method === 'POST') {
        return jsonResponse({ error: 'Simulation unavailable' }, 500)
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MatchupSimulator
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /run simulation/i }))

    await waitFor(() => {
      expect(screen.getByText('Simulation unavailable')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/matchup/simulate', expect.objectContaining({ method: 'POST' }))
  })

  it('handles selecting saved deck with missing cards array', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([
          {
            id: 'deck-1',
            user_id: 'user-1',
            name: 'Deck without cards',
            format: 'modern',
          },
        ])
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <MatchupSimulator
        token="token-123"
        user={{ id: 'user-1' }}
        decklist=""
        format="standard"
        messages={messages}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('Deck without cards (modern)')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue('Select a deck'), { target: { value: 'deck-1' } })

    await waitFor(() => {
      expect(screen.getByDisplayValue('Modern')).toBeInTheDocument()
    })
  })
})

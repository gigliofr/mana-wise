import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import SideboardCoach from './SideboardCoach'

const messages = {
  sideboardTitle: 'Sideboard Coach',
  mainDecklist: 'Main decklist',
  decklistHint: '4 Lightning Bolt',
  selectADeck: 'Select a deck',
  loading: 'Loading...',
  loadSavedDeck: 'Load saved deck',
  sideboardDecklist: 'Sideboard decklist',
  sideboardHint: '2 Abrade',
  format: 'Format',
  opponentArchetype: 'Opponent archetype',
  archetypeLabel: a => a,
  planning: 'Planning',
  runPlan: 'Run plan',
  sideboardFailed: 'Sideboard failed',
  matchupLabel: 'Matchup',
  sideboardIns: 'Ins',
  sideboardOuts: 'Outs',
  sideboardNotes: 'Notes',
  qty: 'Qty',
  card: 'Card',
  reason: 'Reason',
}

function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'application/json' },
    json: vi.fn().mockResolvedValue(body),
  }
}

describe('SideboardCoach', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('runs sideboard plan and renders swaps', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/sideboard/plan' && options.method === 'POST') {
        return jsonResponse({
          matchup: 'aggro',
          ins: [{ qty: 2, card: 'Abrade', reason: 'Remove cheap threats' }],
          outs: [{ qty: 2, card: 'Negate', reason: 'Too reactive here' }],
          notes: ['Prioritize cheap interaction.'],
        })
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <SideboardCoach
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    fireEvent.change(screen.getByPlaceholderText('2 Abrade'), { target: { value: '2 Abrade' } })
    fireEvent.click(screen.getByRole('button', { name: /run plan/i }))

    await waitFor(() => {
      expect(screen.getByText('Abrade')).toBeInTheDocument()
      expect(screen.getByText('Negate')).toBeInTheDocument()
      expect(screen.getByText('Prioritize cheap interaction.')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/sideboard/plan', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('shows backend error message on failure', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/sideboard/plan' && options.method === 'POST') {
        return jsonResponse({ error: 'Planning failed' }, 500)
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <SideboardCoach
        token="token-123"
        user={{ id: 'user-1' }}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
        messages={messages}
      />,
    )

    fireEvent.change(screen.getByPlaceholderText('2 Abrade'), { target: { value: '2 Abrade' } })
    fireEvent.click(screen.getByRole('button', { name: /run plan/i }))

    await waitFor(() => {
      expect(screen.getByText('Planning failed')).toBeInTheDocument()
    })
  })
})

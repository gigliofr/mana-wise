import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import Analyzer from './Analyzer'

function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'application/json' },
    json: vi.fn().mockResolvedValue(body),
  }
}

const messages = {
  deckAnalyzer: 'Deck Analyzer',
  decklist: 'Decklist',
  decklistHint: '1 Card Name',
  loadSavedDeck: 'Load saved deck',
  selectADeck: 'Select a deck',
  format: 'Format',
  analyzing: 'Analyzing',
  analyzeDeck: 'Analyze deck',
  analysisFailed: 'Analysis failed',
}

describe('Analyzer', () => {
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
            format: 'modern',
            cards: [{ card_name: 'Lightning Bolt', quantity: 4 }],
          },
          {
            id: 'deck-2',
            user_id: 'user-2',
            name: 'Other Deck',
            format: 'standard',
            cards: [{ card_name: 'Island', quantity: 10 }],
          },
        ])
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <Analyzer
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        locale="en"
        messages={messages}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('Owned Deck (modern)')).toBeInTheDocument()
      expect(screen.queryByText('Other Deck (standard)')).not.toBeInTheDocument()
    })

    fireEvent.change(screen.getByDisplayValue('Select a deck'), { target: { value: 'deck-1' } })

    await waitFor(() => {
      expect(screen.getByDisplayValue('4 Lightning Bolt')).toBeInTheDocument()
      expect(screen.getByDisplayValue('Modern')).toBeInTheDocument()
    })
  })

  it('shows backend error when analyze fails', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/analyze' && options.method === 'POST') {
        return jsonResponse({ error: 'No credits left' }, 429)
      }
      if (url === '/api/v1/deck/classify' && options.method === 'POST') {
        return jsonResponse({ archetype: 'aggro' })
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <Analyzer
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        locale="en"
        messages={messages}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /analyze deck/i }))

    await waitFor(() => {
      expect(screen.getByText('No credits left')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/analyze', expect.objectContaining({ method: 'POST' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/deck/classify', expect.objectContaining({ method: 'POST' }))
  })

  it('falls back to generic message when analyze error response is non-json', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([])
      }
      if (url === '/api/v1/analyze' && options.method === 'POST') {
        return {
          ok: false,
          status: 500,
          headers: { get: () => 'text/plain' },
          text: vi.fn().mockResolvedValue('internal error'),
        }
      }
      if (url === '/api/v1/deck/classify' && options.method === 'POST') {
        return jsonResponse({ archetype: 'aggro' })
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <Analyzer
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        locale="en"
        messages={messages}
        decklist={'4 Lightning Bolt\n20 Mountain'}
        format="modern"
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /analyze deck/i }))

    await waitFor(() => {
      expect(screen.getByText('Analysis failed')).toBeInTheDocument()
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
            format: 'modern',
          },
        ])
      }
      throw new Error(`Unhandled request: ${String(url)} ${String(options.method)}`)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <Analyzer
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        locale="en"
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

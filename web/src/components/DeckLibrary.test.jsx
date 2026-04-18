import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import DeckLibrary from './DeckLibrary'

const messages = {
  deckLibraryTitle: 'Deck Library',
  currentDeck: 'Current deck',
  cardsCount: n => `${n} cards`,
  planLabel: 'Plan',
  planPro: 'Pro',
  planFree: 'Free',
  deckSlots: 'Deck slots',
  unlimited: 'Unlimited',
  saveDeck: 'Save deck',
  deckLimitReached: 'Deck limit reached',
  loading: 'Loading...',
  noSavedDecks: 'No saved decks',
  deckLoadFailed: 'Deck load failed',
  deckDeleteFailed: 'Deck delete failed',
  legalityUnavailableShort: 'N/D',
  legalityLegalLabel: 'Legal',
  legalityIllegalLabel: 'Illegal',
  unknownLabel: 'N/A',
  useDeck: 'Use deck',
  deleteDeck: 'Delete deck',
  expandDeckList: 'Expand deck list',
  collapseDeckList: 'Collapse deck list',
  changeDeckFormat: 'Change deck format',
}

function jsonResponse(body, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'application/json' },
    json: vi.fn().mockResolvedValue(body),
  }
}

function noContentResponse(status = 204) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => 'text/plain' },
    json: vi.fn(),
  }
}

describe('DeckLibrary', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads only user-owned decks and renders summary value', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks?page=1&limit=3' && options.method === 'GET') {
        return jsonResponse({
          data: [
            {
              id: 'deck-1',
              user_id: 'user-1',
              name: 'Mono Black',
              format: 'standard',
              cards: [{ card_name: 'Swamp', quantity: 20, is_sideboard: false }],
            },
            {
              id: 'deck-2',
              user_id: 'user-2',
              name: 'Other User Deck',
              format: 'standard',
              cards: [{ card_name: 'Island', quantity: 20, is_sideboard: false }],
            },
          ],
          total: 2,
          page: 1,
          limit: 3,
        })
      }
      if (url === '/api/v1/decks/deck-1/summary' && options.method === 'GET') {
        return jsonResponse({
          deck_id: 'deck-1',
          estimated_usd: 123.45,
          legality: {
            standard: {
              is_legal: true,
              illegal_cards: [],
            },
          },
        })
      }
      if (url === '/api/v1/cards/metadata/batch' && options.method === 'POST') {
        return jsonResponse({
          items: [{ name: 'Swamp', rarity: 'common', set_code: 'm10' }],
        })
      }
      throw new Error(`Unhandled request in test: ${String(url)} ${String(options.method)}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    render(
      <DeckLibrary
        token="token-123"
        user={{ id: 'user-1', plan: 'free' }}
        messages={messages}
        currentDecklist={'20 Swamp'}
        currentFormat="standard"
      />,
    )

    await waitFor(() => {
      expect(screen.getAllByText('Mono Black').length).toBeGreaterThan(0)
      expect(screen.queryByText('Other User Deck')).not.toBeInTheDocument()
      expect(screen.getByText('~$123.45')).toBeInTheDocument()
      expect(screen.getByText('Legal')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks?page=1&limit=3', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-1/summary', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/cards/metadata/batch', expect.objectContaining({ method: 'POST' }))
  })

  it('falls back to legality endpoint when summary fails and supports delete', async () => {
    let pageDecks = [
      {
        id: 'deck-3',
        user_id: 'user-1',
        name: 'Fallback Deck',
        format: 'modern',
        cards: [{ card_name: 'Lightning Bolt', quantity: 4, is_sideboard: false }],
      },
    ]

    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks?page=1&limit=3' && options.method === 'GET') {
        return jsonResponse({
          data: pageDecks,
          total: pageDecks.length,
          page: 1,
          limit: 3,
        })
      }
      if (url === '/api/v1/decks/deck-3/summary' && options.method === 'GET') {
        return jsonResponse({ error: 'summary unavailable' }, 500)
      }
      if (url === '/api/v1/decks/deck-3/legality' && options.method === 'GET') {
        return jsonResponse({
          formats: {
            modern: {
              is_legal: false,
              illegal_cards: [{ card_name: 'Lightning Bolt' }],
            },
          },
        })
      }
      if (url === '/api/v1/cards/metadata/batch' && options.method === 'POST') {
        return jsonResponse({ items: [] })
      }
      if (url === '/api/v1/decks/deck-3' && options.method === 'DELETE') {
        pageDecks = []
        return noContentResponse(204)
      }
      throw new Error(`Unhandled request in test: ${String(url)} ${String(options.method)}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    render(
      <DeckLibrary
        token="token-123"
        user={{ id: 'user-1', plan: 'free' }}
        messages={messages}
        currentDecklist={'4 Lightning Bolt'}
        currentFormat="modern"
      />,
    )

    await waitFor(() => {
      expect(screen.getAllByText('Fallback Deck').length).toBeGreaterThan(0)
      expect(screen.getByText('Illegal')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Delete deck' }))

    await waitFor(() => {
      expect(screen.getByText('No saved decks')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-3/legality', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-3', expect.objectContaining({ method: 'DELETE' }))
  })

  it('filters tabs by search and allows pin-based ordering', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks?page=1&limit=3' && options.method === 'GET') {
        return jsonResponse({
          data: [
            {
              id: 'deck-a',
              user_id: 'user-1',
              name: 'Aggro Rush',
              format: 'standard',
              cards: [{ card_name: 'Mountain', quantity: 20, is_sideboard: false }],
            },
            {
              id: 'deck-b',
              user_id: 'user-1',
              name: 'Control Shell',
              format: 'modern',
              cards: [{ card_name: 'Island', quantity: 20, is_sideboard: false }],
            },
          ],
          total: 2,
          page: 1,
          limit: 3,
        })
      }
      if (url === '/api/v1/decks/deck-a/summary' && options.method === 'GET') {
        return jsonResponse({ deck_id: 'deck-a', legality: { standard: { is_legal: true, illegal_cards: [] } } })
      }
      if (url === '/api/v1/decks/deck-b/summary' && options.method === 'GET') {
        return jsonResponse({ deck_id: 'deck-b', legality: { modern: { is_legal: true, illegal_cards: [] } } })
      }
      if (url === '/api/v1/cards/metadata/batch' && options.method === 'POST') {
        return jsonResponse({ items: [] })
      }
      throw new Error(`Unhandled request in test: ${String(url)} ${String(options.method)}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    render(
      <DeckLibrary
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        messages={messages}
        currentDecklist={'20 Mountain'}
        currentFormat="standard"
      />,
    )

    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs).toHaveLength(2)
    })

    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'control' } })
    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs).toHaveLength(1)
      expect(tabs[0]).toHaveTextContent('Control Shell')
    })

    fireEvent.change(screen.getByRole('searchbox'), { target: { value: '' } })
    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs).toHaveLength(2)
    })

    fireEvent.click(screen.getAllByText('Pin')[1])

    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs[0]).toHaveTextContent('Control Shell')
      expect(screen.getByText('Pinned')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByLabelText('Pin deck'))

    fireEvent.click(screen.getByLabelText('Move deck right Control Shell'))

    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs[0]).toHaveTextContent('Aggro Rush')
      expect(tabs[1]).toHaveTextContent('Control Shell')
    })
  })

  it('updates deck format inline and toggles compressed list', async () => {
    let deckData = {
      id: 'deck-fmt',
      user_id: 'user-1',
      name: 'Format Shift',
      format: 'standard',
      is_public: false,
      cards: [{ card_name: 'Plains', quantity: 20, is_sideboard: false }],
    }

    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks?page=1&limit=3' && options.method === 'GET') {
        return jsonResponse({
          data: [deckData],
          total: 1,
          page: 1,
          limit: 3,
        })
      }
      if (url === '/api/v1/decks/deck-fmt/summary' && options.method === 'GET') {
        return jsonResponse({ deck_id: 'deck-fmt', legality: { standard: { is_legal: true, illegal_cards: [] } } })
      }
      if (url === '/api/v1/cards/metadata/batch' && options.method === 'POST') {
        return jsonResponse({ items: [] })
      }
      if (url === '/api/v1/decks/deck-fmt' && options.method === 'PUT') {
        const parsedBody = JSON.parse(options.body)
        deckData = {
          ...deckData,
          ...parsedBody,
          id: 'deck-fmt',
          user_id: 'user-1',
        }
        return jsonResponse(deckData)
      }
      throw new Error(`Unhandled request in test: ${String(url)} ${String(options.method)}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    render(
      <DeckLibrary
        token="token-123"
        user={{ id: 'user-1', plan: 'pro' }}
        messages={messages}
        currentDecklist={'20 Plains'}
        currentFormat="standard"
      />,
    )

    await waitFor(() => {
      expect(screen.getAllByText('Format Shift').length).toBeGreaterThan(0)
      expect(screen.getByRole('button', { name: 'Expand deck list' })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Expand deck list' }))

    await waitFor(() => {
      expect(screen.getByText('Qty')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Collapse deck list' })).toBeInTheDocument()
    })

    fireEvent.change(screen.getByRole('combobox', { name: 'Change deck format' }), { target: { value: 'modern' } })

    await waitFor(() => {
      const tabs = screen.getAllByRole('tab')
      expect(tabs[0]).toHaveTextContent('modern')
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-fmt', expect.objectContaining({ method: 'PUT' }))
  })
})

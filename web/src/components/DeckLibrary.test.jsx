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
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([
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
        ])
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
      expect(screen.getByText('Mono Black')).toBeInTheDocument()
      expect(screen.queryByText('Other User Deck')).not.toBeInTheDocument()
      expect(screen.getByText('~$123.45')).toBeInTheDocument()
      expect(screen.getByText('Legal')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-1/summary', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/cards/metadata/batch', expect.objectContaining({ method: 'POST' }))
  })

  it('falls back to legality endpoint when summary fails and supports delete', async () => {
    const fetchMock = vi.fn(async (url, options = {}) => {
      if (url === '/api/v1/decks' && options.method === 'GET') {
        return jsonResponse([
          {
            id: 'deck-3',
            user_id: 'user-1',
            name: 'Fallback Deck',
            format: 'modern',
            cards: [{ card_name: 'Lightning Bolt', quantity: 4, is_sideboard: false }],
          },
        ])
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
      expect(screen.getByText('Fallback Deck')).toBeInTheDocument()
      expect(screen.getByText('Illegal')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: 'Delete deck' }))

    await waitFor(() => {
      expect(screen.getByText('No saved decks')).toBeInTheDocument()
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-3/legality', expect.objectContaining({ method: 'GET' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/decks/deck-3', expect.objectContaining({ method: 'DELETE' }))
  })
})

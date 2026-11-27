import { describe, it, expect } from 'vitest'
import { parseDecklist, sum, guessFormat, deckToText, apiCardsFromParsed, parsedFromApiDeck, SAMPLE_DECK } from './deck'

describe('parseDecklist', () => {
  it('legge il mazzo di esempio con la sideboard', () => {
    const p = parseDecklist(SAMPLE_DECK)
    expect(sum(p.main)).toBe(60)
    expect(sum(p.side)).toBe(15)
    expect(guessFormat(p)).toBe('modern')
  })

  it('accetta il formato di Arena e la sideboard dopo una riga vuota', () => {
    const p = parseDecklist('4 Lightning Bolt (M11) 149\n20 Mountain\n\n2 Skullcrack')
    expect(p.main).toEqual([{ name: 'Lightning Bolt', qty: 4 }, { name: 'Mountain', qty: 20 }])
    expect(p.side).toEqual([{ name: 'Skullcrack', qty: 2 }])
  })

  it('riconosce il comandante e il formato Commander', () => {
    const lines = ['1 Sisay, Weatherlight Captain [Commander]', ...Array.from({ length: 99 }, (_, i) => `1 Card ${i}`)]
    const p = parseDecklist(lines.join('\n'))
    expect(p.commander).toHaveLength(1)
    expect(guessFormat(p)).toBe('commander')
  })

  it('somma le righe duplicate e gestisce SB:', () => {
    const p = parseDecklist('2 Lava Spike\n2 Lava Spike\nSB: 1 Path to Exile')
    expect(p.main).toEqual([{ name: 'Lava Spike', qty: 4 }])
    expect(p.side).toEqual([{ name: 'Path to Exile', qty: 1 }])
  })

  it('conserva le sezioni nel passaggio da e verso le API', () => {
    const p = parseDecklist(SAMPLE_DECK)
    const back = parsedFromApiDeck({ cards: apiCardsFromParsed(p) })
    expect(deckToText(back)).toBe(deckToText(p))
  })
})

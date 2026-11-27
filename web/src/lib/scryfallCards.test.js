import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fetchCardInfos, clearCardInfoCache, toCardInfo } from './scryfallCards'

const card = (name, extra = {}) => ({ name, type_line: 'Instant', cmc: 2, colors: ['U'], prices: { eur: '0.50' }, image_uris: { normal: `n/${name}`, large: `l/${name}` }, ...extra })

describe('scryfallCards', () => {
  beforeEach(() => clearCardInfoCache())

  it('usa la prima faccia per le carte bifronti', () => {
    const info = toCardInfo({ name: 'A // B', card_faces: [{ type_line: 'Creature', mana_cost: '{G}', image_uris: { normal: 'front' } }, {}] })
    expect(info.image).toBe('front')
    expect(info.typeLine).toBe('Creature')
    expect(info.priceEur).toBeNull()
  })

  it('divide le richieste in blocchi da 75, segna i non trovati e usa la cache', async () => {
    const names = Array.from({ length: 80 }, (_, i) => `Card ${i}`).concat(['Fire', 'Nope'])
    const fetchImpl = vi.fn(async (_url, opts) => {
      const ids = JSON.parse(opts.body).identifiers.map(x => x.name)
      return {
        ok: true,
        json: async () => ({
          data: ids.filter(n => n !== 'Nope').map(n => (n === 'Fire' ? card('Fire // Ice') : card(n))),
          not_found: ids.filter(n => n === 'Nope').map(name => ({ name })),
        }),
      }
    })
    const out = await fetchCardInfos(names, { fetchImpl })
    expect(fetchImpl).toHaveBeenCalledTimes(2)
    expect(JSON.parse(fetchImpl.mock.calls[0][1].body).identifiers).toHaveLength(75)
    expect(out['card 3'].priceEur).toBe(0.5)
    expect(out.fire.name).toBe('Fire // Ice')
    expect(out.nope).toBeNull()

    await fetchCardInfos(['Card 3', 'Fire'], { fetchImpl })
    expect(fetchImpl).toHaveBeenCalledTimes(2)
  })

  it('propaga gli errori HTTP', async () => {
    const fetchImpl = async () => ({ ok: false, status: 429 })
    await expect(fetchCardInfos(['X'], { fetchImpl })).rejects.toThrow('429')
  })
})

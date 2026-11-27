import { describe, it, expect } from 'vitest'
import { typeGroup, costGroup, groupEntries, totalPrice } from './deckView'

const infos = {
  'Sol Ring': { typeLine: 'Artifact', cmc: 1, colors: [], priceEur: 1.5 },
  'Forest': { typeLine: 'Basic Land — Forest', cmc: 0, colors: [] },
  'Llanowar Elves': { typeLine: 'Creature — Elf Druid', cmc: 1, colors: ['G'], priceEur: 0.3 },
  'Dryad Arbor': { typeLine: 'Land Creature — Forest Dryad', cmc: 0, colors: ['G'] },
  'The First Sliver': { typeLine: 'Legendary Creature — Sliver', cmc: 5, colors: ['W', 'U', 'B', 'R', 'G'], priceEur: 4 },
}
const infoOf = e => infos[e.name]

describe('deckView', () => {
  it('classifica le terre prima delle creature e le artefatto-creature come creature', () => {
    expect(typeGroup('Land Creature — Forest Dryad')).toBe('land')
    expect(typeGroup('Artifact Creature — Golem')).toBe('creature')
    expect(typeGroup('')).toBe('other')
    expect(costGroup({ typeLine: 'Sorcery', cmc: 9 })).toBe('7')
  })

  it('mette il comandante in cima e conta le copie, non le righe', () => {
    const entries = [
      { name: 'Forest', qty: 30, zone: 'main' },
      { name: 'Sol Ring', qty: 1, zone: 'main' },
      { name: 'Llanowar Elves', qty: 1, zone: 'main' },
      { name: 'The First Sliver', qty: 1, zone: 'commander' },
    ]
    const groups = groupEntries(entries, infoOf, 'type', 'name')
    expect(groups.map(g => g.key)).toEqual(['commander', 'creature', 'artifact', 'land'])
    expect(groups.find(g => g.key === 'land').count).toBe(30)
  })

  it('ordina per prezzo con le carte senza prezzo in fondo e somma il totale', () => {
    const entries = [
      { name: 'Forest', qty: 10, zone: 'main' },
      { name: 'Llanowar Elves', qty: 2, zone: 'main' },
      { name: 'Sol Ring', qty: 1, zone: 'main' },
    ]
    const [g] = groupEntries(entries, infoOf, 'none', 'price')
    expect(g.entries.map(e => e.name)).toEqual(['Sol Ring', 'Llanowar Elves', 'Forest'])
    const { total, priced } = totalPrice(entries, infoOf)
    expect(total).toBeCloseTo(2.1)
    expect(priced).toBe(2)
  })
})

import { describe, it, expect } from 'vitest'
import { parseDecklist, cleanCardName, sum, guessFormat, deckToText, apiCardsFromParsed, parsedFromApiDeck, SAMPLE_DECK } from './deck'

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

// Export reale di Moxfield (comandante in prima riga, senza intestazioni).
const MOXFIELD_SLIVERS = `1 The First Sliver (J21) 691
1 Arcane Signet (TMC) 57
1 Arctic Treeline (KHM) 249
1 Bonescythe Sliver (M14) 9
1 Bring to Light (BFZ) 209
1 Canopy Vista (MKC) 252
1 Chromatic Lantern (RVR) 442
1 Cinder Glade (FIC) 380
1 Cleansing Nova (FIC) 235
1 Cleaving Sliver (J21) 430
1 Cloudshredder Sliver (SLD) 660 *F*
1 Command Tower (TMC) 63
1 Constricting Sliver (PLST) M15-7
1 Crypt Sliver (PLST) LGN-63
1 Crystalline Sliver (PLST) STH-127
1 Cultivate (TMC) 50
1 Decimate (FIC) 323
1 Descendants' Path (LCC) 238
1 Dregscape Sliver (MH1) 88
1 Eladamri's Call (MH1) 197
1 Essence Sliver (LGN) 13
1 Exotic Orchard (SOC) 369
1 Farseek (TSR) 363
1 Fellwar Stone (SOC) 347
1 Fiery Islet (HA7) 21
1 Flare of Fortitude (MH3) 26
14 Forest (M20) 279
1 Frenetic Sliver (PLC) 157
1 Galerider Sliver (M14) 57
1 Gemhide Sliver (PLST) TSP-196
1 Harmonic Sliver (PLST) TSP-240 *F*
1 Heart Sliver (TMP) 182
1 Hibernation Sliver (STH) 128
1 Highland Forest (KHM) 261
1 Horned Sliver (TMP) 234
1 Hunter Sliver (LGN) 102
1 Inevitable Betrayal (PRM) 91239
1 Inspiring Vantage (OTJ) 303
1 Lancer Sliver (MH1) 18
1 Lazotep Sliver (CMM) 764
1 Lightning Greaves (MRD) 199
1 Lotus Bloom (TSP) 259
1 Magma Sliver (LGN) 107
1 Manaweft Sliver (M14) 184
1 Megantic Sliver (M14) 185
1 Might Sliver (TSP) 205
1 Molten Tributary (SOC) 387
1 Morophon, the Boundless (MH1) 1
1 Nomad Outpost (FIC) 409
1 Opaline Sliver (TSP) 244
1 Path of Ancestry (FIC) 411
1 Path to Exile (FIC) 248
1 Patriarch's Bidding (MH2) 275
1 Pillar of Origins (XLN) 241
1 Plated Sliver (LGN) 19
1 Prairie Stream (FIC) 413
1 Predatory Sliver (M14) 189
1 Quick Sliver (LGN) 136
1 Rampant Growth (8ED) 274
1 Ready / Willing (DGM) 132
1 Realmwalker (PRM) 88334
1 Reflections of Littjara (PRM) 88254
1 Sacred Peaks (FIC) 418
1 Sentinel Sliver (PLST) M14-30
1 Shifting Sliver (LGN) 52
1 Sinew Sliver (TSR) 44
1 Sliver Gravemother (CMM) 777
1 Sliver Hivelord (M15) 211
1 Sliver Overlord (SCG) 139
1 Smoldering Marsh (FIC) 425
1 Sol Ring (TDC) 106
1 Spiteful Sliver (MH1) 148
1 Striking Sliver (M14) 157
1 Sunbaked Canyon (ACR) 111
1 Sunken Hollow (FIC) 429
1 Sunlit Marsh (FIC) 431
1 Swords to Plowshares (SOC) 179
1 Sylvan Scrying (MRD) 130
1 Syphon Sliver (M14) 117
1 Terramorphic Expanse (DRC) 179
1 Thrumming Hivepool (PEOE) 247p *F*
1 Toxin Sliver (PSAL) D25
1 Twisted Landscape (MH3) 232
1 Unclaimed Territory (C20) 320
1 Venom Sliver (PLST) M15-205
1 Winged Sliver (TMP) 106
1 Wooded Ridgeline (FIC) 441`

describe('import da Moxfield', () => {
  it('riconosce il comandante in prima riga e pulisce set, numeri e foil', () => {
    const p = parseDecklist(MOXFIELD_SLIVERS)
    expect(p.commander).toEqual([{ name: 'The First Sliver', qty: 1 }])
    expect(sum(p.main)).toBe(99)
    expect(guessFormat(p)).toBe('commander')
    for (const c of p.main) expect(c.name).not.toMatch(/[()*]|\s\d+$/)
    const names = p.main.map(c => c.name)
    expect(names).toContain('Cloudshredder Sliver')
    expect(names).toContain('Harmonic Sliver')
    expect(names).toContain('Thrumming Hivepool')
    expect(names).toContain('Constricting Sliver')
    expect(names).toContain('Toxin Sliver')
    expect(names).toContain('Ready // Willing')
    expect(apiCardsFromParsed(p)[0]).toMatchObject({ card_name: 'The First Sliver', is_commander: true })
  })

  it('pulisce i suffissi in qualsiasi ordine', () => {
    expect(cleanCardName('Cloudshredder Sliver (SLD) 660 *F*')).toBe('Cloudshredder Sliver')
    expect(cleanCardName('Constricting Sliver (PLST) M15-7')).toBe('Constricting Sliver')
    expect(cleanCardName('Thrumming Hivepool (PEOE) 247p *F*')).toBe('Thrumming Hivepool')
    expect(cleanCardName('Sol Ring (C21) 263 [Ramp]')).toBe('Sol Ring')
    expect(cleanCardName('Fire // Ice')).toBe('Fire // Ice')
  })

  it('non inventa un comandante se la prima carta è in ordine alfabetico o il mazzo non è Commander', () => {
    const sorted = ['1 Aaa Card', ...Array.from({ length: 99 }, (_, i) => `1 Card ${String(i).padStart(3, '0')}`)]
    expect(parseDecklist(sorted.join('\n')).commander).toEqual([])
    const sixty = ['4 Zzz', '4 Aaa', '4 Bbb', '48 Mountain']
    expect(parseDecklist(sixty.join('\n')).commander).toEqual([])
  })
})

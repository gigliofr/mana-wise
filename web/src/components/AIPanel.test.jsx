import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AIPanel } from './Analyzer'

const messages = {
  aiBadge: 'AI ASSISTANT',
  localSummaryBadge: 'INTERNAL RULES',
  commanderBracketLabel: 'Bracket commander',
  aiSourceUnknown: 'source unavailable',
  aiSourceInternal: 'internal rules',
  aiSourceUsed: source => `Suggestion source: ${source}`,
  aiStatusExternal: 'External AI',
  aiStatusInternal: 'Local rules',
  aiStatusFallback: 'Fallback active',
  aiFallbackActiveTitle: 'Local suggestions used after provider error',
  aiUnavailable: 'AI suggestions are unavailable.',
  aiFallbackNote: 'Showing a local summary based on deterministic analysis instead.',
  localSummaryTitle: 'Local summary',
  scoreBandGood: 'Good',
  scoreBandAverage: 'Average',
  scoreBandLow: 'Low',
  localSummaryLines: {
    archetype: archetype => `Deck archetype: ${archetype}`,
    colorSpeed: (archetype, colorCount, risk) => `Color speed: ${archetype} ${colorCount} ${risk}`,
    landRange: (lands, min, max) => `Land range: ${lands} ${min}-${max}`,
    lands: (lands, ideal) => `Lands: ${lands}/${ideal}`,
    consistency: (screw, flood, sweetSpot) => `Consistency: ${screw} ${flood} ${sweetSpot}`,
    cmc: avg => `Il CMC medio è ${avg.toFixed(2)}: descrive quanto il mazzo parte veloce o lenta.`,
    peak: peak => `Peak: ${peak}`,
    interaction: (score, band) => `Interaction: ${score} ${band}`,
    topGap: (label, deficit) => `Gap: ${label} ${deficit}`,
    topGapNone: () => 'No gaps',
    playtestingLoop: () => 'Playtest loop',
  },
}

describe('AIPanel', () => {
  it('shows fallback status and warning when internal source has provider error', () => {
    const { asFragment } = render(
      <AIPanel
        text={'1) Cut A, add B'}
        error={'LLM unavailable (falling back to internal rules): quota exceeded'}
        source={'internal_rules'}
        result={null}
        messages={messages}
      />,
    )

    expect(screen.getByText('Fallback active')).toBeInTheDocument()
    expect(screen.getByText('Suggestion source: internal_rules')).toBeInTheDocument()
    expect(screen.getByText('Local suggestions used after provider error')).toBeInTheDocument()
    expect(screen.getByText(/quota exceeded/i)).toBeInTheDocument()
    expect(asFragment()).toMatchSnapshot()
  })

  it('shows external status without warning when source is external and no error', () => {
    const { asFragment } = render(
      <AIPanel
        text={'1) CUT: X ADD: Y WHY: curve'}
        error={''}
        source={'openai:gpt-4o-mini'}
        result={null}
        messages={messages}
      />,
    )

    expect(screen.getByText('External AI')).toBeInTheDocument()
    expect(screen.getByText('Suggestion source: openai:gpt-4o-mini')).toBeInTheDocument()
    expect(screen.queryByText('Local suggestions used after provider error')).not.toBeInTheDocument()
    expect(asFragment()).toMatchSnapshot()
  })

  it('shows commander bracket in the local summary and keeps CMC text readable', () => {
    render(
      <AIPanel
        text={''}
        error={''}
        source={''}
        commanderScore={6.8}
        result={{
          deterministic: {
            mana: {
              land_count: 37,
              ideal_land_count: 36,
              total_cards: 100,
              average_cmc: 3.73,
              mana_screw_chance: 11,
              mana_flood_chance: 9.1,
              sweet_spot_chance: 79.8,
              distribution: [{ cmc: 3, count: 12 }],
              color_requirements: [],
            },
            interaction: {
              total_score: 100,
              archetype: 'ramp',
              categories: [],
            },
          },
        }}
        messages={messages}
      />,
    )

    expect(screen.getByText('Local summary')).toBeInTheDocument()
    expect(screen.getByText(/Bracket commander: 4 · Optimized · 6\.8\/10/)).toBeInTheDocument()
    expect(screen.getByText(/Il CMC medio è 3\.73/i)).toBeInTheDocument()
  })
})

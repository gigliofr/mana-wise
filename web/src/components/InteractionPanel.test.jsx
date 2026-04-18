import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import InteractionPanel from './InteractionPanel'

const messages = {
  overallScore: 'Overall score',
  interactionReadHint: 'Hint',
  categoryLabels: { removal: 'Removal' },
  rowGood: 'Good',
  rowPartial: 'Partial',
  rowLow: 'Low',
  rowNotRequired: 'Not required',
  naLabel: 'n/a',
  translateSuggestion: (category, count, archetype, ideal) => `Suggestion ${category} ${count} ${archetype} ${ideal}`,
}

describe('InteractionPanel', () => {
  it('keeps base ideals for non-commander formats', () => {
    render(
      <InteractionPanel
        data={{
          total_score: 65,
          format: 'modern',
          messages,
          breakdowns: [{ category: 'removal', count: 10, ideal: 10 }],
          suggestions: [],
        }}
      />,
    )

    expect(screen.getByText('10/10')).toBeInTheDocument()
    expect(screen.getByText('Good')).toBeInTheDocument()
  })

  it('scales ideals by commander bracket score', () => {
    render(
      <InteractionPanel
        data={{
          total_score: 65,
          format: 'commander',
          commanderScore: 8.8,
          messages,
          breakdowns: [{ category: 'removal', count: 10, ideal: 10 }],
          suggestions: [],
        }}
      />,
    )

    expect(screen.getByText('10/14')).toBeInTheDocument()
    expect(screen.getByText('Partial')).toBeInTheDocument()
  })

  it('scales expected value in suggestion text for commander', () => {
    render(
      <InteractionPanel
        data={{
          total_score: 65,
          format: 'commander',
          commanderScore: 8.8,
          messages,
          breakdowns: [{ category: 'removal', count: 10, ideal: 10 }],
          suggestions: ['Your removal package (10 cards) is below ideal for a midrange deck (10 expected). Consider adding more.'],
        }}
      />,
    )

    expect(screen.getByText('Suggestion removal 10 midrange 14')).toBeInTheDocument()
  })
})

import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AIPanel } from './Analyzer'

const messages = {
  aiBadge: 'AI ASSISTANT',
  localSummaryBadge: 'INTERNAL RULES',
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
}

describe('AIPanel', () => {
  it('shows fallback status and warning when internal source has provider error', () => {
    render(
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
  })

  it('shows external status without warning when source is external and no error', () => {
    render(
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
  })
})

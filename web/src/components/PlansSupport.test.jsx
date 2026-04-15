import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import PlansSupport from './PlansSupport'

const messages = {
  planFeatureAnalyses: 'Analyses',
  planFeatureAnalysesFree: '1/day',
  planFeatureAnalysesPro: 'Unlimited',
  planFeatureDeckSlots: 'Deck slots',
  planFeatureDeckSlotsFree: '3',
  planFeatureDeckSlotsPro: 'Unlimited',
  planFeatureTools: 'Tools',
  planFeatureToolsFree: 'Basic',
  planFeatureToolsPro: 'Full',
  planDowngradeConfirm: 'Confirm downgrade?',
  planSwitchError: 'Plan switch failed',
  planDonationReferenceRequired: 'Donation reference required',
  planSectionTitle: 'Plans',
  planSectionSubtitle: 'Manage access and support',
  planFreeTitle: 'Free',
  planFreeBadge: 'Current',
  planProTitle: 'Pro',
  planProBadge: 'Beta',
  planBetaNotice: 'Beta notice',
  planCurrent: 'Current plan',
  planSelect: 'Select',
  planDonateMonth: 'Donate monthly',
  planDonateYear: 'Donate yearly',
  planDonationReferenceLabel: 'Reference',
  planDonationReferencePlaceholder: 'Enter reference',
  planActivateMonth: 'Activate month',
  planActivateYear: 'Activate year',
  planActiveUntil: date => `Active until ${date}`,
  donateTitle: 'Support the project',
  donateBody: 'Donate here',
  donateButton: 'Donate',
  metricsSectionTitle: 'Funnel metrics',
  metricsSectionSubtitle: 'Admin snapshot',
  metricsSecretLabel: 'Secret',
  metricsSecretPlaceholder: 'Enter secret',
  metricsRefresh: 'Load metrics',
  metricsDownloadJson: 'Download JSON',
  metricsDownloadCsv: 'Download CSV',
  metricsLoadError: 'Unable to load metrics',
  metricsTotalEvents: 'Total events',
  metricsFallbacks: 'Fallbacks',
  metricsForwardingErrors: 'Forwarding errors',
  metricsEventCounts: 'Event counts',
  metricsByAISource: 'By source',
  metricsNoData: 'No data',
  metricsLastEvent: value => `Last event at ${value}`,
  metricsDeltaStable: 'stable',
  loading: 'Loading...',
}

describe('PlansSupport metrics panel', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({
        snapshot: {
          total_events: 12,
          analysis_fallbacks: 3,
          forwarding_errors: 1,
          last_event_at_unix_ms: 1700000000000,
          event_counts: { analysis_requested: 5 },
          analysis_by_ai_source: { fallback: 3 },
        },
      }),
    }))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads and renders the funnel metrics snapshot', async () => {
    render(
      <PlansSupport
        token="token-123"
        user={{ plan: 'free' }}
        messages={messages}
      />,
    )

    fireEvent.change(screen.getByPlaceholderText('Enter secret'), { target: { value: 'admin-secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Load metrics' }))

    await waitFor(() => {
      expect(screen.getByText('12')).toBeInTheDocument()
      expect(screen.getByText('analysis_requested: 5')).toBeInTheDocument()
      expect(screen.getByText('fallback: 3')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Download JSON' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Download CSV' })).toBeInTheDocument()
    })

    expect(fetch).toHaveBeenCalledWith('/api/v1/admin/metrics/funnel', expect.objectContaining({
      headers: expect.objectContaining({
        Authorization: 'Bearer token-123',
        'X-Admin-Secret': 'admin-secret',
      }),
    }))
  })
})
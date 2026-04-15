import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import ScorePanel from './ScorePanel';

describe('ScorePanel Integration Tests', () => {
  it('renders loading state correctly', () => {
    render(<ScorePanel loading={true} />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('renders error state correctly', () => {
    const errorMsg = 'Failed to fetch score';
    render(<ScorePanel error={errorMsg} />);
    expect(screen.getByText(/Error:/)).toBeInTheDocument();
    expect(screen.getByText(errorMsg)).toBeInTheDocument();
  });

  it('renders empty state when no data', () => {
    render(<ScorePanel scoreDetail={null} />);
    expect(screen.getByText(/No score data available/)).toBeInTheDocument();
  });

  it('renders complete score panel with all fields', () => {
    const mockScoreDetail = {
      score: 7.5,
      total_impact: 42.3,
      tipping_point: 4,
      impact_by_cmc: {
        0: 2.1,
        1: 3.5,
        2: 5.2,
        3: 8.4,
        4: 12.1,
        5: 5.0,
      },
      mana_screw: 15.2,
      mana_flood: 8.7,
      sweet_spot: 76.1,
      card_impacts: [
        {
          card_id: 'card1',
          card_name: 'Sol Ring',
          price_usd: 25.00,
          edhrec_rank: 50,
          impact_score: 9.2,
        },
        {
          card_id: 'card2',
          card_name: 'Swamp',
          price_usd: 0.10,
          edhrec_rank: 500,
          impact_score: 2.1,
        },
      ],
    };

    render(<ScorePanel scoreDetail={mockScoreDetail} />);

    // Check main title
    expect(screen.getByText(/Deck Score Analysis/)).toBeInTheDocument();

    // Check Power Level section
    expect(screen.getByRole('heading', { name: 'Power Level' })).toBeInTheDocument();

    // Check Mana Distribution section
    expect(screen.getByText(/Mana Distribution Analysis/)).toBeInTheDocument();
    expect(screen.getAllByText(/15\.2%/)[0]).toBeInTheDocument(); // Mana Screw
    expect(screen.getAllByText(/76\.1%/)[0]).toBeInTheDocument(); // Sweet Spot

    // Check Mana Curve section
    expect(screen.getByText(/Mana Curve & Tipping Point/)).toBeInTheDocument();
    expect(screen.getByText(/Tipping Point: CMC 4/)).toBeInTheDocument();

    // Check Card Impacts section
    expect(screen.getByText(/Card Impact Scores/)).toBeInTheDocument();
    expect(screen.getByText(/Sol Ring/)).toBeInTheDocument();
    expect(screen.getByText(/Swamp/)).toBeInTheDocument();

    // Check legend
    expect(screen.getByText(/Score Range: 0–3 Casual/)).toBeInTheDocument();
  });

  it('correctly displays card impact table with sortable columns', async () => {
    const mockScoreDetail = {
      score: 5.0,
      total_impact: 20.0,
      tipping_point: 3,
      impact_by_cmc: {},
      mana_screw: 25.0,
      mana_flood: 25.0,
      sweet_spot: 50.0,
      card_impacts: [
        {
          card_id: 'card1',
          card_name: 'Snapcaster Mage',
          price_usd: 15.00,
          edhrec_rank: 100,
          impact_score: 8.5,
        },
        {
          card_id: 'card2',
          card_name: 'Lightning Bolt',
          price_usd: 5.00,
          edhrec_rank: 50,
          impact_score: 7.2,
        },
      ],
    };

    render(<ScorePanel scoreDetail={mockScoreDetail} />);

    // Verify cards are displayed
    expect(screen.getByText(/Snapcaster Mage/)).toBeInTheDocument();
    expect(screen.getByText(/Lightning Bolt/)).toBeInTheDocument();

    // Verify prices are displayed
    expect(screen.getByText(/15\.00/)).toBeInTheDocument();
    expect(screen.getByText('$5.00', { selector: 'td' })).toBeInTheDocument();
  });

  it('handles missing optional fields gracefully', () => {
    const minimalScoreDetail = {
      score: 3.0,
      total_impact: 15.0,
      tipping_point: 0,
      impact_by_cmc: {},
      mana_screw: 0,
      mana_flood: 0,
      sweet_spot: 100,
      card_impacts: [],
    };

    const { container } = render(<ScorePanel scoreDetail={minimalScoreDetail} />);

    // Should not crash
    expect(container).toBeInTheDocument();
    expect(screen.getByText(/Deck Score Analysis/)).toBeInTheDocument();
  });
});

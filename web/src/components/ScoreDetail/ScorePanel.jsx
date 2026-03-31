import React from 'react';
import ScoreGauge from './ScoreGauge';
import CardImpactTable from './CardImpactTable';
import ManaChart from './ManaChart';
import CurveChart from './CurveChart';

/**
 * ScorePanel — Orchestrator di tutti i componenti ScoreDetail
 * Consuma un oggetto ScoreDetail dal backend POST /score
 */
function ScorePanel({ scoreDetail = null, loading = false, error = null, token, messages }) {
  if (loading) {
    return (
      <div className="flex justify-center items-center p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
        <strong>Error:</strong> {error}
      </div>
    );
  }

  if (!scoreDetail) {
    return (
      <div className="bg-gray-100 border border-gray-400 text-gray-700 px-4 py-3 rounded">
        No score data available
      </div>
    );
  }

  const {
    score = 0,
    total_impact = 0,
    tipping_point = 0,
    impact_by_cmc = {},
    mana_screw = 0,
    mana_flood = 0,
    sweet_spot = 0,
    card_impacts = [],
  } = scoreDetail;

  return (
    <div className="space-y-8 p-6 bg-white rounded-lg shadow">
      {/* Header */}
      <div className="border-b pb-4">
        <h2 className="text-2xl font-bold text-gray-900">Deck Score Analysis</h2>
        <p className="text-gray-600 mt-1">
          Comprehensive evaluation of mana curve, card impact, and statistical draw probabilities
        </p>
      </div>

      {/* 1. Power Level Gauge */}
      <div className="space-y-2">
        <h3 className="text-lg font-semibold text-gray-800">Power Level</h3>
        <div className="flex justify-center">
          <div className="w-64 h-64">
            <ScoreGauge score={score} />
          </div>
        </div>
        <div className="text-center text-gray-600 text-sm">
          <p>Total Impact: <strong>{total_impact.toFixed(2)}</strong></p>
        </div>
      </div>

      {/* 2. Mana Analysis */}
      <div className="space-y-2">
        <h3 className="text-lg font-semibold text-gray-800">Mana Distribution Analysis</h3>
        <p className="text-gray-600 text-sm">
          Probability of drawing optimal, insufficient, or excessive mana in your opening hand and first draws
        </p>
        <ManaChart 
          manaScrew={manaScrew}
          manaFlood={manaFlood}
          sweetSpot={sweetSpot}
        />
      </div>

      {/* 3. Mana Curve & Tipping Point */}
      <div className="space-y-2">
        <h3 className="text-lg font-semibold text-gray-800">Mana Curve & Tipping Point</h3>
        <p className="text-gray-600 text-sm">
          Total impact per casting cost. Tipping Point (red) is the CMC where your deck peaks in power.
        </p>
        <CurveChart 
          impactByCMC={impact_by_cmc}
          tippingPoint={tipping_point}
        />
      </div>

      {/* 4. Card Impacts */}
      <div className="space-y-2">
        <h3 className="text-lg font-semibold text-gray-800">Card Impact Scores</h3>
        <p className="text-gray-600 text-sm">
          Individual card ratings based on price, EDHREC popularity, and reprint frequency. Click columns to sort.
        </p>
        <CardImpactTable cardImpacts={card_impacts} token={token} messages={messages} />
      </div>

      {/* Footer */}
      <div className="text-xs text-gray-500 border-t pt-4">
        <p>Score Range: 0–3 Casual | 4–6 Mid-Power | 7–8 High-Power | 9–10 cEDH</p>
      </div>
    </div>
  );
}

export default ScorePanel;

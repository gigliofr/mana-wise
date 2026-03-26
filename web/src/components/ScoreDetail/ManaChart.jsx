import React from 'react';

/**
 * ManaChart — Visualizzazione Screw/Flood/SweetSpot
 */
function ManaChart({ manaScrew = 0, manaFlood = 0, sweetSpot = 0 }) {
  const total = manaScrew + manaFlood + sweetSpot;
  const screenWidth = Math.min(window.innerWidth - 40, 500);
  const chartHeight = 200;

  // Bar widths proporzionali al valore
  const screwWidth = total > 0 ? (manaScrew / total) * screenWidth : 0;
  const floodWidth = total > 0 ? (manaFlood / total) * screenWidth : 0;
  const sweetWidth = total > 0 ? (sweetSpot / total) * screenWidth : 0;

  return (
    <div className="space-y-4">
      <div className="flex gap-8">
        {/* Mana Screw */}
        <div className="flex-1">
          <div className="text-sm font-semibold text-red-600 mb-2">
            Mana Screw: {manaScrew.toFixed(1)}%
          </div>
          <div className="w-full bg-gray-200 rounded h-8 overflow-hidden relative">
            <div
              className="bg-red-500 h-full transition-all duration-500"
              style={{ width: `${manaScrew < 5 ? 5 : manaScrew}%` }}
            />
            <div className="absolute inset-0 flex items-center justify-center text-xs text-gray-700 font-bold">
              {manaScrew.toFixed(1)}%
            </div>
          </div>
        </div>

        {/* Mana Flood */}
        <div className="flex-1">
          <div className="text-sm font-semibold text-gray-600 mb-2">
            Mana Flood: {manaFlood.toFixed(1)}%
          </div>
          <div className="w-full bg-gray-200 rounded h-8 overflow-hidden relative">
            <div
              className="bg-gray-500 h-full transition-all duration-500"
              style={{ width: `${manaFlood < 5 ? 5 : manaFlood}%` }}
            />
            <div className="absolute inset-0 flex items-center justify-center text-xs text-white font-bold">
              {manaFlood.toFixed(1)}%
            </div>
          </div>
        </div>

        {/* Sweet Spot */}
        <div className="flex-1">
          <div className="text-sm font-semibold text-green-600 mb-2">
            Sweet Spot: {sweetSpot.toFixed(1)}%
          </div>
          <div className="w-full bg-gray-200 rounded h-8 overflow-hidden relative">
            <div
              className="bg-green-500 h-full transition-all duration-500"
              style={{ width: `${sweetSpot < 5 ? 5 : sweetSpot}%` }}
            />
            <div className="absolute inset-0 flex items-center justify-center text-xs text-white font-bold">
              {sweetSpot.toFixed(1)}%
            </div>
          </div>
        </div>
      </div>

      {/* Legend */}
      <div className="flex gap-6 text-xs mt-4">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-red-500 rounded" />
          <span>Screw: Draw more mana than needed</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-gray-500 rounded" />
          <span>Flood: Draw fewer lands than needed</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-green-500 rounded" />
          <span>Sweet Spot: Optimal mana distribution</span>
        </div>
      </div>
    </div>
  );
}

export default ManaChart;

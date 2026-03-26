import React from 'react';

/**
 * CurveChart — Grafico barrato dell'impact per CMC
 */
function CurveChart({ impactByCMC = {}, tippingPoint = 0 }) {
  // Assicurati che impactByCMC sia un oggetto valido
  const data = typeof impactByCMC === 'object' && impactByCMC !== null ? impactByCMC : {};
  
  // Estrai i CMC da 0 a 10+, e ordina
  const cmcValues = Object.keys(data)
    .map(k => parseInt(k))
    .filter(k => !isNaN(k))
    .sort((a, b) => a - b);

  if (cmcValues.length === 0) {
    return <div className="text-gray-500 text-sm">No mana curve data available</div>;
  }

  // Trova il max per la scala Y
  const maxImpact = Math.max(...cmcValues.map(cmc => data[cmc] || 0), 1);
  const chartHeight = 250;

  return (
    <div className="space-y-4">
      {/* Chart Area */}
      <div className="overflow-x-auto">
        <div className="flex items-end gap-2 p-4 border rounded bg-gray-50 min-w-max">
          {cmcValues.map((cmc) => {
            const impact = data[cmc] || 0;
            const barHeight = (impact / maxImpact) * chartHeight;
            const isTP = cmc === tippingPoint;

            return (
              <div key={cmc} className="flex flex-col items-center gap-2">
                {/* Bar */}
                <div
                  className={`transition-all duration-300 ${
                    isTP
                      ? 'bg-red-500 border-2 border-red-700'
                      : 'bg-blue-500'
                  } rounded hover:opacity-80`}
                  style={{
                    width: '40px',
                    height: `${barHeight}px`,
                    minHeight: '5px',
                  }}
                  title={`CMC ${cmc}: ${impact.toFixed(2)} impact`}
                />
                {/* Label */}
                <div className="text-xs font-semibold text-gray-700">
                  {cmc}
                </div>
                {/* Value */}
                <div className="text-xs text-gray-600">
                  {impact.toFixed(1)}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex gap-6 text-xs mt-4">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-blue-500 rounded" />
          <span>Impact per CMC</span>
        </div>
        {tippingPoint !== undefined && tippingPoint > 0 && (
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-red-500 border-2 border-red-700 rounded" />
            <span>Tipping Point: CMC {tippingPoint} (Mana Curve Peak)</span>
          </div>
        )}
      </div>
    </div>
  );
}

export default CurveChart;

import React from 'react';

/**
 * ScoreGauge — Gauge circolare 0-10 con colori
 * - Verde (0-3): casual
 * - Giallo (4-6): mid-power
 * - Arancione (7-8): high-power
 * - Rosso (9-10): cEDH competitive
 */
function ScoreGauge({ score = 5 }) {
  const getColor = (s) => {
    if (s <= 3) return '#10b981'; // verde
    if (s <= 6) return '#eab308'; // giallo
    if (s <= 8) return '#f97316'; // arancione
    return '#ef4444'; // rosso
  };

  const getLabel = (s) => {
    if (s <= 3) return 'Casual';
    if (s <= 6) return 'Mid-Power';
    if (s <= 8) return 'High-Power';
    return 'cEDH';
  };

  const color = getColor(score);
  const label = getLabel(score);
  const circumference = 2 * Math.PI * 45;
  const offset = circumference - (score / 10) * circumference;

  return (
    <div className="flex flex-col items-center justify-center gap-4">
      <div className="relative w-40 h-40">
        <svg
          viewBox="0 0 120 120"
          className="w-full h-full transform -rotate-90"
        >
          {/* Background arc */}
          <circle
            cx="60"
            cy="60"
            r="45"
            fill="none"
            stroke="#e5e7eb"
            strokeWidth="4"
          />
          {/* Progress arc */}
          <circle
            cx="60"
            cy="60"
            r="45"
            fill="none"
            stroke={color}
            strokeWidth="4"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            style={{ transition: 'stroke-dashoffset 0.5s ease' }}
          />
        </svg>
        {/* Center text */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <div className="text-4xl font-bold" style={{ color }}>
            {score.toFixed(1)}
          </div>
          <div className="text-sm text-gray-600">/10</div>
        </div>
      </div>
      <div className="text-center">
        <div className="text-lg font-semibold">{label}</div>
        <div className="text-xs text-gray-500">Power Level</div>
      </div>
    </div>
  );
}

export default ScoreGauge;

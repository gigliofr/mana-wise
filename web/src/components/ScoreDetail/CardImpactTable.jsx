import React, { useState } from 'react';
import CardHoverPreview from '../CardHoverPreview';

/**
 * CardImpactTable — Tabella ordinabile delle carte con impact score
 */
function CardImpactTable({ cardImpacts = [], token, messages }) {
  const [sortBy, setSortBy] = useState('impact');
  const [sortOrder, setSortOrder] = useState('desc');

  const sortedCards = [...cardImpacts].sort((a, b) => {
    let aVal = a[sortBy] || 0;
    let bVal = b[sortBy] || 0;

    if (sortBy === 'card_name') {
      aVal = a.card_name.toLowerCase();
      bVal = b.card_name.toLowerCase();
    }

    if (sortOrder === 'asc') {
      return aVal > bVal ? 1 : aVal < bVal ? -1 : 0;
    } else {
      return aVal < bVal ? 1 : aVal > bVal ? -1 : 0;
    }
  });

  const toggleSort = (field) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('desc');
    }
  };

  const SortIcon = ({ field }) => {
    if (sortBy !== field) return <span className="opacity-30">↕</span>;
    return sortOrder === 'desc' ? <span>↓</span> : <span>↑</span>;
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-100 border-b">
          <tr>
            <th className="px-4 py-2 text-left cursor-pointer hover:bg-gray-200"
                onClick={() => toggleSort('card_name')}>
              Card <SortIcon field="card_name" />
            </th>
            <th className="px-4 py-2 text-center cursor-pointer hover:bg-gray-200"
                onClick={() => toggleSort('price_usd')}>
              Price $ <SortIcon field="price_usd" />
            </th>
            <th className="px-4 py-2 text-center cursor-pointer hover:bg-gray-200"
                onClick={() => toggleSort('edhrec_rank')}>
              EDHREC <SortIcon field="edhrec_rank" />
            </th>
            <th className="px-4 py-2 text-center cursor-pointer hover:bg-gray-200"
                onClick={() => toggleSort('impact_score')}>
              Impact <SortIcon field="impact_score" />
            </th>
          </tr>
        </thead>
        <tbody>
          {sortedCards.map((card, i) => (
            <tr key={i} className={i % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
              <td className="px-4 py-2">
                <CardHoverPreview cardName={card.card_name} token={token} messages={messages}>
                  {card.card_name}
                </CardHoverPreview>
              </td>
              <td className="px-4 py-2 text-center">${card.price_usd?.toFixed(2) || '–'}</td>
              <td className="px-4 py-2 text-center">{card.edhrec_rank || '–'}</td>
              <td className="px-4 py-2 text-center">
                <span className="font-bold text-blue-600">
                  {card.impact_score?.toFixed(1)}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default CardImpactTable;

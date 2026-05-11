import React, { useEffect } from "react";
import { useParams } from "react-router-dom";

function fmtNumber(value, digits = 2) {
  const n = Number(value);
  if (!Number.isFinite(n)) return "-";
  return n.toFixed(digits);
}

function buildSummaryA4Canvas({ analysis, token, expiresAt }) {
  const width = 1240;
  const height = 1754;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;

  // Palette and base
  const bg = "#f7f3ec";
  const border = "#cdbfa9";
  const ink = "#1f2937";
  const dim = "#6b7280";
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, width, height);

  // Outer card
  const margin = 70;
  ctx.strokeStyle = border;
  ctx.lineWidth = 6;
  ctx.strokeRect(margin, margin + 40, width - margin * 2, height - margin * 2 - 40);

  // Header
  ctx.fillStyle = ink;
  ctx.font = "700 56px Georgia";
  ctx.fillText("ManaWise - Riepilogo Analisi", margin + 18, margin + 92);

  // Token & expiry (small)
  ctx.fillStyle = dim;
  ctx.font = "400 18px Georgia";
  ctx.fillText(`Token: ${token || "-"}`, margin + 24, margin + 132);
  ctx.fillText(`Valido fino: ${expiresAt ? new Date(expiresAt).toLocaleString() : "-"}`, margin + 24, margin + 156);

  // Two-column stats block
  const colLeft = margin + 40;
  const colRight = width / 2 + 20;
  let y = margin + 220;

  const statLabelFont = "500 26px Georgia";
  const statValueFont = "700 44px Georgia";

  function drawStat(label, value, xLabel, xValue) {
    ctx.fillStyle = dim;
    ctx.font = statLabelFont;
    ctx.fillText(label, xLabel, y);
    ctx.fillStyle = ink;
    ctx.font = statValueFont;
    ctx.fillText(value, xValue, y);
    y += 88;
  }

  drawStat("Formato:", String(analysis?.format || "-"), colLeft, colLeft + 240);
  drawStat("Carte totali:", String(analysis?.mana?.total_cards ?? "-"), colLeft, colLeft + 240);
  drawStat("CMC medio:", fmtNumber(analysis?.mana?.average_cmc, 2), colLeft, colLeft + 240);
  drawStat("Terre:", String(analysis?.mana?.land_count ?? "-"), colRight, colRight + 240);
  drawStat("Interazione:", String(analysis?.interaction?.total_score ?? "-"), colRight, colRight + 240);
  drawStat("Score:", String(analysis?.score_detail?.score ?? "-"), colRight, colRight + 240);

  // Horizontal rule
  ctx.strokeStyle = border;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(margin + 24, y + 8);
  ctx.lineTo(width - margin - 24, y + 8);
  ctx.stroke();

  // Notes area
  y += 40;
  ctx.fillStyle = dim;
  ctx.font = "500 20px Georgia";
  const notes = [
    `Archetipo: ${analysis?.interaction?.archetype || "-"}`,
    `Mulligan keep%: ${fmtNumber(analysis?.mulligan?.keep_probability, 1)}%`,
    `Mana screw: ${fmtNumber(analysis?.mana?.mana_screw_probability, 1)}%`,
    `Mana flood: ${fmtNumber(analysis?.mana?.mana_flood_probability, 1)}%`,
  ];
  for (const n of notes) {
    ctx.fillText(n, margin + 28, y);
    y += 38;
  }

  // Footer
  ctx.fillStyle = dim;
  ctx.font = "500 18px Georgia";
  ctx.fillText("Creato con ManaWise - formato A4", margin + 24, height - margin - 18);

  return canvas;
}

async function createSummaryA4File(payload) {
  const canvas = buildSummaryA4Canvas(payload);
  if (!canvas) return null;
  const blob = await new Promise((resolve) => {
    canvas.toBlob((b) => resolve(b), "image/png");
  });
  if (!blob) return null;
  return new File([blob], `manawise-share-${payload?.token || "analysis"}.png`, { type: "image/png" });
}

async function shareSummaryA4Image(payload) {
  const file = await createSummaryA4File(payload);
  if (!file) return false;
  if (!navigator.share || !navigator.canShare || !navigator.canShare({ files: [file] })) {
    return false;
  }
  await navigator.share({
    title: "ManaWise - Riepilogo Analisi",
    text: "Riepilogo analisi deck",
    files: [file],
  });
  return true;
}

async function downloadSummaryA4Image(payload) {
  const file = await createSummaryA4File(payload);
  if (!file) return;
  const url = URL.createObjectURL(file);

  const link = document.createElement("a");
  link.href = url;
  link.download = file.name;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

async function downloadSharedAnalysisPdf(token) {
  const response = await fetch(`/api/v1/analysis/share/${token}/pdf`);
  if (!response.ok) {
    const maybeJSON = await response.json().catch(() => null);
    throw new Error(maybeJSON?.error || "Impossibile scaricare il PDF");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `mana-wise-share-${token}.pdf`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export default function SharedAnalysisPage() {
  const { token } = useParams();

  useEffect(() => {
    if (!token) return;
    window.location.replace(`/api/v1/analysis/share/${token}/pdf`);
  }, [token]);

  return (
    <div
      className="shared-analysis-page"
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "#0f1116",
        color: "#e5e7eb",
        fontFamily: "system-ui, sans-serif",
        padding: 16,
      }}
    >
      Preparazione del PDF...
    </div>
  );
}

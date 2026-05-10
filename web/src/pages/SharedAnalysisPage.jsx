import React, { useEffect, useState } from "react";
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [data, setData] = useState(null);
  const [copied, setCopied] = useState(false);
  const [shareImageBusy, setShareImageBusy] = useState(false);
  const [shareImageError, setShareImageError] = useState("");
  const [sharePdfBusy, setSharePdfBusy] = useState(false);
  const [sharePdfError, setSharePdfError] = useState("");

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    setError("");
    fetch(`/api/v1/analysis/share/${token}`)
      .then(async (res) => {
        if (!res.ok) {
          const maybeJSON = await res.json().catch(() => null);
          const apiError = maybeJSON?.error;
          throw new Error(apiError || "Errore nel caricamento della condivisione");
        }
        return res.json();
      })
      .then(setData)
      .catch((e) => setError(e.message || "Errore di caricamento"))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <div className="shared-analysis-page">Caricamento…</div>;
  if (error) return <div className="shared-analysis-page error">Errore: {error}</div>;
  if (!data) return null;

  const { analysis, expires_at, shared_by } = data;
  const shareUrl = window.location.origin + "/share/" + token;
  const shareText = encodeURIComponent(`Guarda questa analisi di un mazzo su ManaWise!\n${shareUrl}`);
  return (
    <div className="shared-analysis-page" style={{
      maxWidth: 380,
      margin: "0 auto",
      padding: 8,
      fontFamily: "system-ui, sans-serif",
      background: "#111115",
      borderRadius: 10,
      boxShadow: "0 2px 12px #0002"
    }}>
      <div style={{display:"flex",alignItems:"center",gap:8,marginBottom:8}}>
        <span style={{fontSize:22,lineHeight:1}}>🔮</span>
        <span style={{fontWeight:700,fontSize:18,color:"#9b7fe0",letterSpacing:-0.5}}>Analisi condivisa</span>
      </div>
      <div style={{fontSize:12, color:"#aaa", marginBottom:6}}>
        {expires_at && (
          <>Valida fino al: {new Date(expires_at).toLocaleString()}</>
        )}
      </div>
      <div style={{display:"flex",gap:6,marginBottom:10,alignItems:"center"}}>
        <a aria-label="Condividi su WhatsApp" href={`https://wa.me/?text=${shareText}`} target="_blank" rel="noopener noreferrer" style={{background:"#25D366",color:"#fff",padding:"7px 10px",borderRadius:5,textDecoration:"none",fontWeight:600,fontSize:13}}>WhatsApp</a>
        <a aria-label="Condividi su Telegram" href={`https://t.me/share/url?url=${encodeURIComponent(shareUrl)}&text=${shareText}`} target="_blank" rel="noopener noreferrer" style={{background:"#229ED9",color:"#fff",padding:"7px 10px",borderRadius:5,textDecoration:"none",fontWeight:600,fontSize:13}}>Telegram</a>
        <button
          aria-label="Condividi immagine A4"
          disabled={shareImageBusy}
          onClick={async () => {
            setShareImageError("");
            setShareImageBusy(true);
            try {
              const shared = await shareSummaryA4Image({ analysis, token, expiresAt: expires_at });
              if (!shared) {
                await downloadSummaryA4Image({ analysis, token, expiresAt: expires_at });
              }
            } catch (e) {
              setShareImageError("Impossibile condividere l'immagine su questo dispositivo.");
            } finally {
              setShareImageBusy(false);
            }
          }}
          style={{background:"#0f766e",color:"#fff",padding:"6px 10px",borderRadius:5,border:"none",fontSize:13,fontWeight:600}}
        >
          {shareImageBusy ? "..." : "Condividi A4"}
        </button>
        <button
          aria-label="Scarica riepilogo A4"
          onClick={() => downloadSummaryA4Image({ analysis, token, expiresAt: expires_at })}
          style={{background:"#7c3aed",color:"#fff",padding:"6px 10px",borderRadius:5,border:"none",fontSize:13,fontWeight:600}}
        >
          Scarica A4
        </button>
        <button
          aria-label="Scarica PDF"
          disabled={sharePdfBusy}
          onClick={async () => {
            setSharePdfError("");
            setSharePdfBusy(true);
            try {
              await downloadSharedAnalysisPdf(token);
            } catch (e) {
              setSharePdfError(e.message || "Impossibile scaricare il PDF");
            } finally {
              setSharePdfBusy(false);
            }
          }}
          style={{background:"#0f172a",color:"#fff",padding:"6px 10px",borderRadius:5,border:"none",fontSize:13,fontWeight:600}}
        >
          {sharePdfBusy ? "..." : "Scarica PDF"}
        </button>
        <button aria-label="Copia link" onClick={async () => {
          try {
            await navigator.clipboard.writeText(shareUrl)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          } catch (e) {
            // fallback
            const ta = document.createElement('textarea')
            ta.value = shareUrl
            document.body.appendChild(ta)
            ta.select()
            try { document.execCommand('copy') } catch {}
            document.body.removeChild(ta)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          }
        }} style={{marginLeft:'auto',background:'#444',color:'#fff',padding:'6px 10px',borderRadius:5,border:'none',fontSize:13}}>Copia link</button>
      </div>
      {shareImageError && (
        <div className="banner banner-error" style={{ marginBottom: 8 }}>{shareImageError}</div>
      )}
      {sharePdfError && (
        <div className="banner banner-error" style={{ marginBottom: 8 }}>{sharePdfError}</div>
      )}
      <div style={{background:"#191922",borderRadius:7,padding:"10px 12px",marginBottom:10,fontSize:14,lineHeight:1.6}}>
        <div><span style={{color:"#aaa"}}>Formato:</span> <b>{analysis?.format}</b></div>
        <div><span style={{color:"#aaa"}}>Carte:</span> <b>{analysis?.mana?.total_cards}</b></div>
        <div><span style={{color:"#aaa"}}>CMC:</span> <b>{analysis?.mana?.average_cmc?.toFixed(2)}</b></div>
        <div><span style={{color:"#aaa"}}>Score:</span> <b>{analysis?.score_detail?.score ?? "-"}</b></div>
      </div>
      <details style={{marginBottom:10}}>
        <summary style={{fontSize:13,color:"#9b7fe0",cursor:"pointer"}}>Dettagli analisi</summary>
        <pre style={{background:"#222",color:"#eee",padding:6,borderRadius:4,overflowX:"auto",fontSize:11,marginTop:6}}>
          {JSON.stringify(analysis, null, 2)}
        </pre>
      </details>
      <div style={{fontSize:11, color:"#888",textAlign:"center",marginTop:8}}>
        Powered by <a href="https://mana-wise.app" style={{color:"#9b7fe0",textDecoration:"none"}}>ManaWise</a>
      </div>
      {copied && (
        <div role="status" aria-live="polite" style={{position:'fixed',left:'50%',transform:'translateX(-50%)',bottom:24,background:'#2b2b2b',color:'#fff',padding:'10px 14px',borderRadius:8,boxShadow:'0 6px 20px rgba(0,0,0,0.4)'}}>Link copiato!</div>
      )}
    </div>
  );
}

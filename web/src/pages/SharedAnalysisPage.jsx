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

  ctx.fillStyle = "#f5f2ea";
  ctx.fillRect(0, 0, width, height);

  ctx.fillStyle = "#1f2937";
  ctx.font = "700 56px Georgia";
  ctx.fillText("ManaWise - Riepilogo Analisi", 80, 120);

  ctx.strokeStyle = "#c8b89d";
  ctx.lineWidth = 4;
  ctx.strokeRect(70, 160, width - 140, height - 250);

  ctx.fillStyle = "#374151";
  ctx.font = "500 28px Georgia";
  ctx.fillText(`Token: ${token || "-"}`, 100, 230);
  ctx.fillText(`Valido fino: ${expiresAt ? new Date(expiresAt).toLocaleString() : "-"}`, 100, 280);

  const rows = [
    ["Formato", String(analysis?.format || "-")],
    ["Carte totali", String(analysis?.mana?.total_cards ?? "-")],
    ["CMC medio", fmtNumber(analysis?.mana?.average_cmc, 2)],
    ["Terre", String(analysis?.mana?.land_count ?? "-")],
    ["Interazione", String(analysis?.interaction?.total_score ?? "-")],
    ["Score", String(analysis?.score_detail?.score ?? "-")],
  ];

  let y = 380;
  for (const [label, value] of rows) {
    ctx.fillStyle = "#6b7280";
    ctx.font = "500 30px Georgia";
    ctx.fillText(`${label}:`, 120, y);

    ctx.fillStyle = "#111827";
    ctx.font = "700 34px Georgia";
    ctx.fillText(value, 430, y);
    y += 90;
  }

  const notes = [
    `Archetipo: ${analysis?.interaction?.archetype || "-"}`,
    `Mulligan keep%: ${fmtNumber(analysis?.mulligan?.keep_probability, 1)}%`,
    `Mana screw: ${fmtNumber(analysis?.mana?.mana_screw_probability, 1)}%`,
    `Mana flood: ${fmtNumber(analysis?.mana?.mana_flood_probability, 1)}%`,
  ];

  ctx.fillStyle = "#374151";
  ctx.font = "500 26px Georgia";
  y += 20;
  for (const row of notes) {
    ctx.fillText(row, 120, y);
    y += 56;
  }

  ctx.fillStyle = "#9ca3af";
  ctx.font = "500 22px Georgia";
  ctx.fillText("Creato con ManaWise - formato A4", 120, height - 110);

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

export default function SharedAnalysisPage() {
  const { token } = useParams();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [data, setData] = useState(null);
  const [copied, setCopied] = useState(false);
  const [shareImageBusy, setShareImageBusy] = useState(false);
  const [shareImageError, setShareImageError] = useState("");

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

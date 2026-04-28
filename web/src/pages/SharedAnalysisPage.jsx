import React, { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

export default function SharedAnalysisPage() {
  const { token } = useParams();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [data, setData] = useState(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    setError("");
    fetch(`/share/${token}`)
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
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

// Lista e builder: cerca carte, modifica le quantità, salva o esporta.
import { useEffect, useMemo, useRef, useState } from 'react'
import { apiRequest } from '../lib/apiClient'
import { parseDecklist, deckToText, sum, apiCardsFromParsed, apiMaybeboardFromParsed, isBasicLand } from '../lib/deck'
import { ManaCost } from './ManaPips'
import VisualDeckBuilder from './VisualDeckBuilder'

import { TYPE_ORDER, typeGroup } from '../lib/deckView'

const metaCache = new Map()

async function scryfallSearch(query, signal) {
  const url = `https://api.scryfall.com/cards/search?order=edhrec&unique=cards&q=${encodeURIComponent(query)}`
  const res = await fetch(url, { signal, headers: { Accept: 'application/json' } })
  if (res.status === 404) return []
  if (!res.ok) throw new Error(`Scryfall ${res.status}`)
  const data = await res.json()
  return (data.data || []).slice(0, 30).map(c => ({
    name: c.name,
    printedName: c.printed_name || '',
    manaCost: c.mana_cost || c.card_faces?.[0]?.mana_cost || '',
    typeLine: c.type_line || '',
    cmc: c.cmc || 0,
    oracle: (c.oracle_text || c.card_faces?.[0]?.oracle_text || '').split('\n')[0],
  }))
}

export default function DeckListEditor({ token, user, deck, messages: m, locale = 'it', colors = [], onChange, onSaved, onReanalyze }) {
  const parsed = useMemo(() => parseDecklist(deck.text), [deck.text])
  const [section, setSection] = useState('main')
  const [query, setQuery] = useState('')
  const [filters, setFilters] = useState({ colors: true, cheap: false, instant: false, legal: true })
  const [results, setResults] = useState([])
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState('')
  const [asText, setAsText] = useState(false)
  const [visual, setVisual] = useState(false)
  const [draftText, setDraftText] = useState(deck.text)
  const [meta, setMeta] = useState({})
  const [status, setStatus] = useState('')
  const [saving, setSaving] = useState(false)
  const abortRef = useRef(null)
  const hasPro = user?.plan === 'pro' && (!user?.pro_until || new Date(user.pro_until) > new Date())

  useEffect(() => { setDraftText(deck.text) }, [deck.text])

  // Tipi delle carte per raggruppare la lista (dal catalogo del server).
  useEffect(() => {
    const names = [...parsed.commander, ...parsed.main, ...parsed.side, ...parsed.maybe].map(c => c.name).filter(n => !metaCache.has(n.toLowerCase()))
    if (!names.length || !token) {
      setMeta(Object.fromEntries(metaCache))
      return
    }
    let cancelled = false
    apiRequest('/cards/metadata/batch', { token, method: 'POST', body: { names: names.slice(0, 200) } })
      .then(({ res, data }) => {
        if (!res.ok) return
        for (const it of data?.items || []) metaCache.set(String(it.name).toLowerCase(), it)
        if (!cancelled) setMeta(Object.fromEntries(metaCache))
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [deck.text, token])

  // Ricerca con attesa breve tra una battuta e l'altra.
  useEffect(() => {
    const q = query.trim()
    if (q.length < 2) {
      setResults([])
      setSearchError('')
      return undefined
    }
    const parts = [q]
    if (filters.colors && colors.length) parts.push(`id<=${colors.join('').toLowerCase()}`)
    if (filters.cheap) parts.push('cmc<=2')
    if (filters.instant) parts.push('t:instant')
    if (filters.legal && deck.format) parts.push(`legal:${deck.format}`)
    const timer = setTimeout(async () => {
      abortRef.current?.abort()
      const ctrl = new AbortController()
      abortRef.current = ctrl
      setSearching(true)
      setSearchError('')
      try {
        const found = await scryfallSearch(parts.join(' '), ctrl.signal)
        for (const c of found) metaCache.set(c.name.toLowerCase(), { name: c.name, type_line: c.typeLine, mana_cost: c.manaCost, cmc: c.cmc })
        setResults(found)
      } catch (err) {
        if (err.name !== 'AbortError') setSearchError(m.wSearchError)
      } finally {
        setSearching(false)
      }
    }, 350)
    return () => clearTimeout(timer)
  }, [query, filters, colors.join(''), deck.format])

  function commit(next) {
    onChange(deckToText(next))
    setStatus('')
  }

  function changeQty(target, name, delta) {
    const next = { main: [...parsed.main], side: [...parsed.side], commander: [...parsed.commander], maybe: [...parsed.maybe] }
    const list = next[target]
    const idx = list.findIndex(c => c.name.toLowerCase() === name.toLowerCase())
    if (idx === -1) {
      if (delta > 0) list.push({ name, qty: delta })
    } else {
      const qty = list[idx].qty + delta
      if (qty <= 0) list.splice(idx, 1)
      else list[idx] = { ...list[idx], qty }
    }
    commit(next)
  }

  function addCard(name) {
    const target = section === 'side' || section === 'maybe' ? section : 'main'
    const current = parsed[target].find(c => c.name.toLowerCase() === name.toLowerCase())
    const limit = deck.format === 'commander' ? 1 : 4
    if (current && current.qty >= limit && !isBasicLand(name)) {
      setStatus(m.wCopyLimit(limit))
      return
    }
    changeQty(target, name, 1)
    setStatus(m.wAdded(name))
  }

  async function save() {
    setSaving(true)
    setStatus('')
    try {
      const body = { name: deck.name, format: deck.format, cards: apiCardsFromParsed(parsed), maybeboard: apiMaybeboardFromParsed(parsed), is_public: false, note: 'builder' }
      const { res, data } = deck.id
        ? await apiRequest(`/decks/${deck.id}`, { token, method: 'PUT', body })
        : await apiRequest('/decks', { token, method: 'POST', body })
      if (!res.ok) throw new Error(data?.error || m.deckSaveFailed || 'Error')
      onSaved?.(data?.id || deck.id)
      setStatus(m.wSaved)
    } catch (err) {
      setStatus(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function exportArena() {
    // Arena non conosce le carte "forse": restano fuori dall'esportazione.
    const text = deckToText({ ...parsed, maybe: [] })
    try {
      await navigator.clipboard.writeText(text)
      setStatus(m.wCopied)
    } catch {
      setAsText(true)
    }
  }

  const list = section === 'side' ? parsed.side
    : section === 'maybe' ? parsed.maybe
      : [...parsed.commander.map(c => ({ ...c, commander: true })), ...parsed.main]
  const groups = {}
  for (const c of list) {
    const g = c.commander ? 'commander' : typeGroup(meta[c.name.toLowerCase()]?.type_line)
    ;(groups[g] ||= []).push(c)
  }
  const groupOrder = ['commander', ...TYPE_ORDER].filter(g => groups[g]?.length)
  const mainCount = sum(parsed.main) + sum(parsed.commander)
  const sideCount = sum(parsed.side)
  const maybeCount = sum(parsed.maybe)
  const showVisual = visual && hasPro && !asText

  return (
    <div className={`mw-builder${showVisual ? ' mw-builder--visual' : ''}`}>
      <section className="mw-panel mw-search" aria-labelledby="mwSearchTitle">
        <h2 className="mw-panel__title" id="mwSearchTitle">{m.wAddCards}</h2>
        <label htmlFor="mwCardQuery" className="mw-sr">{m.wSearchCards}</label>
        <input id="mwCardQuery" type="search" value={query} onChange={e => setQuery(e.target.value)} placeholder={m.wSearchPh} autoComplete="off" />
        <div className="mw-pills mw-pills--sm" role="group" aria-label={m.wSearchFilters}>
          {colors.length > 0 && <button type="button" className={`mw-pill${filters.colors ? ' is-active' : ''}`} aria-pressed={filters.colors} onClick={() => setFilters(f => ({ ...f, colors: !f.colors }))}>{m.wFilterColors}</button>}
          <button type="button" className={`mw-pill${filters.cheap ? ' is-active' : ''}`} aria-pressed={filters.cheap} onClick={() => setFilters(f => ({ ...f, cheap: !f.cheap }))}>{m.wFilterCheap}</button>
          <button type="button" className={`mw-pill${filters.instant ? ' is-active' : ''}`} aria-pressed={filters.instant} onClick={() => setFilters(f => ({ ...f, instant: !f.instant }))}>{m.wFilterInstant}</button>
          <button type="button" className={`mw-pill${filters.legal ? ' is-active' : ''}`} aria-pressed={filters.legal} onClick={() => setFilters(f => ({ ...f, legal: !f.legal }))}>{m.wFilterLegal}</button>
        </div>
        <div aria-live="polite" className="mw-search__results">
          {searching && <p className="mw-muted mw-small">{m.wSearching}</p>}
          {searchError && <p className="mw-error">{searchError}</p>}
          {!searching && !searchError && query.trim().length >= 2 && results.length === 0 && <p className="mw-muted mw-small">{m.wNoCards}</p>}
          {query.trim().length < 2 && <p className="mw-muted mw-small">{m.wSearchHint}</p>}
          <ul className="mw-results">
            {results.map(c => (
              <li key={c.name}>
                <div className="mw-results__body">
                  <p className="mw-results__name">{c.name}</p>
                  <p className="mw-muted mw-small">{c.typeLine}{c.oracle ? ` · ${c.oracle}` : ''}</p>
                </div>
                <ManaCost cost={c.manaCost} />
                <button type="button" className="mw-icon-btn" onClick={() => addCard(c.name)} aria-label={m.wAddCard(c.name)}>+</button>
              </li>
            ))}
          </ul>
        </div>
        <p className="mw-muted mw-small mw-credit">{m.wScryfallCredit}</p>
      </section>

      <section className="mw-panel mw-list" aria-labelledby="mwListTitle">
        <div className="mw-panel__head">
          <h2 className="mw-panel__title" id="mwListTitle">{m.wListTitle(mainCount)}</h2>
          {!showVisual && (
            <div className="mw-seg" role="tablist" aria-label={m.wSection}>
              <button type="button" role="tab" aria-selected={section === 'main'} className={section === 'main' ? 'is-active' : ''} onClick={() => setSection('main')}>{m.builderMainboard} {mainCount}</button>
              <button type="button" role="tab" aria-selected={section === 'side'} className={section === 'side' ? 'is-active' : ''} onClick={() => setSection('side')}>{m.builderSideboard} {sideCount}</button>
              <button type="button" role="tab" aria-selected={section === 'maybe'} className={section === 'maybe' ? 'is-active' : ''} onClick={() => setSection('maybe')}>{m.builderMaybeboard} {maybeCount}</button>
            </div>
          )}
        </div>

        {status && <p className="mw-status" role="status">{status}</p>}

        {asText ? (
          <div className="mw-astext">
            <label htmlFor="mwAsText" className="mw-sr">{m.wDeckList}</label>
            <textarea id="mwAsText" rows={18} value={draftText} onChange={e => setDraftText(e.target.value)} spellCheck={false} />
            <div className="mw-actions">
              <button type="button" className="mw-btn mw-btn--primary" onClick={() => { onChange(deckToText(parseDecklist(draftText))); setAsText(false) }}>{m.wApplyText}</button>
              <button type="button" className="mw-btn mw-btn--ghost" onClick={() => { setDraftText(deck.text); setAsText(false) }}>{m.wCancel}</button>
            </div>
          </div>
        ) : showVisual ? (
          <VisualDeckBuilder
            messages={m}
            parsed={parsed}
            locale={locale}
            onChange={next => commit(next)}
          />
        ) : list.length === 0 ? (
          <p className="mw-muted">{section === 'side' ? m.wEmptySide : section === 'maybe' ? m.vbEmptyBoard : m.wEmptyMain}</p>
        ) : (
          <div className="mw-groups">
            {groupOrder.map(g => (
              <div key={g} className="mw-group">
                <h3>{m.wType?.[g] || g} · {sum(groups[g])}</h3>
                <ul>
                  {groups[g].map(c => {
                    const info = meta[c.name.toLowerCase()]
                    const target = c.commander ? 'commander' : (section === 'side' || section === 'maybe' ? section : 'main')
                    return (
                      <li key={c.name}>
                        <span className="mw-group__qty">{c.qty}</span>
                        <span className="mw-group__name" title={info?.type_line || undefined}>{c.name}</span>
                        {info?.mana_cost ? <ManaCost cost={info.mana_cost} /> : null}
                        <span className="mw-group__btns">
                          <button type="button" className="mw-icon-btn mw-icon-btn--sm" onClick={() => changeQty(target, c.name, -1)} aria-label={m.wRemoveOne(c.name)}>−</button>
                          <button type="button" className="mw-icon-btn mw-icon-btn--sm" onClick={() => changeQty(target, c.name, 1)} aria-label={m.wAddOne(c.name)}>+</button>
                        </span>
                      </li>
                    )
                  })}
                </ul>
              </div>
            ))}
          </div>
        )}

        <div className="mw-actions mw-actions--foot">
          <button type="button" className="mw-btn mw-btn--primary" onClick={onReanalyze}>{m.wReanalyze}</button>
          <button type="button" className="mw-btn mw-btn--ghost" onClick={save} disabled={saving}>{saving ? m.wSaving : (deck.id ? m.wSaveChanges : m.wSaveDeck)}</button>
          <button type="button" className="mw-btn mw-btn--ghost" onClick={exportArena}>{m.wExportArena}</button>
          {!asText && <button type="button" className="mw-btn mw-btn--ghost" onClick={() => setAsText(true)}>{m.wEditAsText}</button>}
          {hasPro && !asText && (
            <button type="button" className="mw-btn mw-btn--ghost" aria-pressed={visual} onClick={() => setVisual(v => !v)}>{visual ? m.wListView : m.wVisualView}</button>
          )}
        </div>
      </section>
    </div>
  )
}


// Vista visuale del mazzo (Pro): immagini delle carte in pile o in griglia,
// raggruppate e ordinate a scelta, con trascinamento tra Main, Sideboard e Forse.
import { useEffect, useMemo, useState } from 'react'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import { ManaCost } from './ManaPips'
import { fetchCardInfos } from '../lib/scryfallCards'
import { groupEntries, totalPrice, formatEur } from '../lib/deckView'

const BOARDS = ['main', 'side', 'maybe']
const PREFS_KEY = 'manawise_visual_prefs'

function loadPrefs() {
  try {
    return { view: 'stacks', groupBy: 'type', sortBy: 'name', ...JSON.parse(localStorage.getItem(PREFS_KEY) || '{}') }
  } catch {
    return { view: 'stacks', groupBy: 'type', sortBy: 'name' }
  }
}

function savePrefs(prefs) {
  try { localStorage.setItem(PREFS_KEY, JSON.stringify(prefs)) } catch { /* ignore */ }
}

const keyOf = name => String(name || '').trim().toLowerCase()

// zoneOfBoard: nel main il comandante ha una zona sua.
function entriesOf(parsed, board) {
  if (board === 'main') {
    return [
      ...(parsed.commander || []).map(c => ({ ...c, zone: 'commander' })),
      ...(parsed.main || []).map(c => ({ ...c, zone: 'main' })),
    ]
  }
  return (parsed[board] || []).map(c => ({ ...c, zone: board }))
}

const sumQty = list => list.reduce((acc, c) => acc + (Number(c.qty) || 0), 0)

function CardFace({ entry, info, m }) {
  if (info?.image) {
    return (
      <img
        className="vb-card__img"
        src={info.image}
        alt={entry.name}
        loading="lazy"
        decoding="async"
        draggable={false}
      />
    )
  }
  // Senza immagine: una carta testuale leggibile, con costo e tipo.
  return (
    <div className="vb-card__text">
      <span className="vb-card__text-name">{entry.name}</span>
      {info?.manaCost ? <ManaCost cost={info.manaCost} /> : null}
      {info?.typeLine ? <span className="vb-card__text-type">{info.typeLine}</span> : null}
      {info === null ? <span className="vb-card__text-type">{m.vbNotFound?.(1) || 'Not found'}</span> : null}
    </div>
  )
}

function DraggableCard({ entry, info, m, onQty }) {
  const id = `${entry.zone}|${entry.name}`
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id, data: { entry } })
  const price = info?.priceEur
  return (
    <div
      ref={setNodeRef}
      className={`vb-card${isDragging ? ' is-dragging' : ''}`}
      title={entry.name}
      {...attributes}
      {...listeners}
    >
      <CardFace entry={entry} info={info} m={m} />
      {entry.qty > 1 && <span className="vb-card__qty" aria-label={m.vbCopies?.(entry.qty)}>{entry.qty}×</span>}
      <div className="vb-card__bar">
        {price != null && <span className="vb-card__price">{formatEur(price)}</span>}
        <span className="vb-card__btns">
          <button
            type="button"
            className="vb-card__btn"
            onPointerDown={e => e.stopPropagation()}
            onKeyDown={e => e.stopPropagation()}
            onClick={() => onQty(entry, -1)}
            aria-label={m.vbRemoveOne?.(entry.name)}
          >−</button>
          <button
            type="button"
            className="vb-card__btn"
            onPointerDown={e => e.stopPropagation()}
            onKeyDown={e => e.stopPropagation()}
            onClick={() => onQty(entry, 1)}
            aria-label={m.vbAddOne?.(entry.name)}
          >+</button>
        </span>
      </div>
    </div>
  )
}

function BoardTab({ board, label, count, active, onSelect, dragging, hint }) {
  const { setNodeRef, isOver } = useDroppable({ id: `board-${board}`, data: { board } })
  return (
    <button
      ref={setNodeRef}
      type="button"
      role="tab"
      aria-selected={active}
      className={`vb-tab${active ? ' is-active' : ''}${isOver ? ' is-over' : ''}${dragging ? ' is-target' : ''}`}
      onClick={() => onSelect(board)}
      title={dragging ? hint : undefined}
    >
      <span>{label}</span>
      <strong>{count}</strong>
    </button>
  )
}

function groupLabel(key, groupBy, m) {
  if (key === 'commander') return m.wType?.commander || 'Commander'
  if (groupBy === 'cost') return key === 'land' ? m.vbLands : m.vbCost?.(Number(key))
  if (groupBy === 'color') return m.vbColors?.[key] || key
  if (groupBy === 'none') return m.vbAllInGroup
  return m.wType?.[key] || key
}

export default function VisualDeckBuilder({ messages: m, parsed, locale = 'it', onChange }) {
  const [prefs, setPrefs] = useState(loadPrefs)
  const [board, setBoard] = useState('main')
  const [query, setQuery] = useState('')
  const [rarities, setRarities] = useState([])
  const [sets, setSets] = useState([])
  const [infos, setInfos] = useState({})
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [dragged, setDragged] = useState(null)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor),
  )

  useEffect(() => { savePrefs(prefs) }, [prefs])

  const allNames = useMemo(() => {
    const names = ['commander', 'main', 'side', 'maybe'].flatMap(z => (parsed[z] || []).map(c => c.name))
    return Array.from(new Set(names))
  }, [parsed])

  useEffect(() => {
    if (!allNames.length) return undefined
    const ctrl = new AbortController()
    setLoading(true)
    setLoadError('')
    fetchCardInfos(allNames, { signal: ctrl.signal })
      .then(map => setInfos(prev => ({ ...prev, ...map })))
      .catch(err => { if (err?.name !== 'AbortError') setLoadError(m.vbLoadError) })
      .finally(() => { if (!ctrl.signal.aborted) setLoading(false) })
    return () => ctrl.abort()
  }, [allNames.join('\n')])

  const infoOf = entry => infos[keyOf(entry.name)]
  const notFound = allNames.filter(n => infos[keyOf(n)] === null).length

  const counts = {
    main: sumQty(entriesOf(parsed, 'main')),
    side: sumQty(entriesOf(parsed, 'side')),
    maybe: sumQty(entriesOf(parsed, 'maybe')),
  }

  const boardEntries = entriesOf(parsed, board)
  const rarityOptions = Array.from(new Set(boardEntries.map(e => infoOf(e)?.rarity).filter(Boolean))).sort()
  const setOptions = Array.from(new Set(boardEntries.map(e => infoOf(e)?.set).filter(Boolean))).sort()

  const q = query.trim().toLowerCase()
  const visible = boardEntries.filter(e => {
    const info = infoOf(e)
    if (q && !e.name.toLowerCase().includes(q) && !String(info?.typeLine || '').toLowerCase().includes(q)) return false
    if (rarities.length && !rarities.includes(info?.rarity)) return false
    if (sets.length && !sets.includes(info?.set)) return false
    return true
  })
  const groups = groupEntries(visible, infoOf, prefs.groupBy, prefs.sortBy)
  const price = totalPrice(boardEntries, infoOf)

  function commit(next) {
    onChange?.(next)
  }

  function cloneParsed() {
    return {
      commander: [...(parsed.commander || [])],
      main: [...(parsed.main || [])],
      side: [...(parsed.side || [])],
      maybe: [...(parsed.maybe || [])],
    }
  }

  function changeQty(entry, delta) {
    const next = cloneParsed()
    const list = next[entry.zone]
    const idx = list.findIndex(c => keyOf(c.name) === keyOf(entry.name))
    if (idx === -1) return
    const qty = list[idx].qty + delta
    if (qty <= 0) list.splice(idx, 1)
    else list[idx] = { ...list[idx], qty }
    commit(next)
  }

  function moveEntry(entry, toBoard) {
    const target = toBoard === 'main' ? 'main' : toBoard
    if (entry.zone === target || (entry.zone === 'commander' && toBoard === 'main')) return
    const next = cloneParsed()
    next[entry.zone] = next[entry.zone].filter(c => keyOf(c.name) !== keyOf(entry.name))
    const existing = next[target].find(c => keyOf(c.name) === keyOf(entry.name))
    if (existing) existing.qty += entry.qty
    else next[target].push({ name: entry.name, qty: entry.qty })
    commit(next)
  }

  function handleDragEnd(event) {
    setDragged(null)
    const toBoard = event.over?.data?.current?.board
    const entry = event.active?.data?.current?.entry
    if (toBoard && entry) moveEntry(entry, toBoard)
  }

  const setPref = (key, value) => setPrefs(p => ({ ...p, [key]: value }))
  const toggle = (setter, value) => setter(prev => (prev.includes(value) ? prev.filter(v => v !== value) : [...prev, value]))
  const boardLabel = b => (b === 'main' ? m.builderMainboard : b === 'side' ? m.builderSideboard : m.builderMaybeboard)
  const draggedInfo = dragged ? infoOf(dragged) : null

  return (
    <div className="vb">
      <DndContext
        sensors={sensors}
        onDragStart={e => setDragged(e.active?.data?.current?.entry || null)}
        onDragCancel={() => setDragged(null)}
        onDragEnd={handleDragEnd}
      >
        <div className="vb-tabs" role="tablist" aria-label={m.wSection}>
          {BOARDS.map(b => (
            <BoardTab
              key={b}
              board={b}
              label={boardLabel(b)}
              count={counts[b]}
              active={board === b}
              onSelect={setBoard}
              dragging={!!dragged}
              hint={m.vbMoveHint}
            />
          ))}
          {price.priced > 0 && <span className="vb-price">{m.vbTotalPrice?.(formatEur(price.total, locale))}</span>}
        </div>

        <div className="vb-toolbar">
          <div className="vb-field">
            <span className="vb-label">{m.vbView}</span>
            <div className="mw-seg" role="group" aria-label={m.vbView}>
              {['stacks', 'grid'].map(v => (
                <button key={v} type="button" className={prefs.view === v ? 'is-active' : ''} aria-pressed={prefs.view === v} onClick={() => setPref('view', v)}>
                  {v === 'stacks' ? m.vbViewStacks : m.vbViewGrid}
                </button>
              ))}
            </div>
          </div>
          <label className="vb-field">
            <span className="vb-label">{m.vbGroupBy}</span>
            <select value={prefs.groupBy} onChange={e => setPref('groupBy', e.target.value)}>
              <option value="type">{m.vbGroupType}</option>
              <option value="cost">{m.vbGroupCost}</option>
              <option value="color">{m.vbGroupColor}</option>
              <option value="none">{m.vbGroupNone}</option>
            </select>
          </label>
          <label className="vb-field">
            <span className="vb-label">{m.vbSortBy}</span>
            <select value={prefs.sortBy} onChange={e => setPref('sortBy', e.target.value)}>
              <option value="name">{m.vbSortName}</option>
              <option value="cost">{m.vbSortCost}</option>
              <option value="price">{m.vbSortPrice}</option>
            </select>
          </label>
          <label className="vb-field vb-field--grow">
            <span className="vb-label">{m.vbSearch}</span>
            <input type="search" value={query} onChange={e => setQuery(e.target.value)} placeholder={m.vbSearchPh} />
          </label>
        </div>

        {(rarityOptions.length > 1 || setOptions.length > 1) && (
          <details className="vb-filters">
            <summary>{m.vbFilters}{(rarities.length || sets.length) ? ` · ${rarities.length + sets.length}` : ''}</summary>
            {rarityOptions.length > 1 && (
              <div className="mw-pills mw-pills--sm" role="group" aria-label={m.builderFilterRarity}>
                <button type="button" className={`mw-pill${rarities.length === 0 ? ' is-active' : ''}`} onClick={() => setRarities([])}>{m.vbAllCards}</button>
                {rarityOptions.map(r => (
                  <button key={r} type="button" className={`mw-pill${rarities.includes(r) ? ' is-active' : ''}`} aria-pressed={rarities.includes(r)} onClick={() => toggle(setRarities, r)}>{r}</button>
                ))}
              </div>
            )}
            {setOptions.length > 1 && (
              <div className="mw-pills mw-pills--sm vb-filters__sets" role="group" aria-label={m.builderFilterSet}>
                <button type="button" className={`mw-pill${sets.length === 0 ? ' is-active' : ''}`} onClick={() => setSets([])}>{m.vbAllCards}</button>
                {setOptions.map(s => (
                  <button key={s} type="button" className={`mw-pill${sets.includes(s) ? ' is-active' : ''}`} aria-pressed={sets.includes(s)} onClick={() => toggle(setSets, s)}>{s.toUpperCase()}</button>
                ))}
              </div>
            )}
          </details>
        )}

        <p className="mw-muted mw-small vb-status" aria-live="polite">
          {loading ? m.vbLoading : loadError || (notFound ? m.vbNotFound?.(notFound) : m.builderHint)}
        </p>

        {boardEntries.length === 0 ? (
          <p className="mw-muted vb-empty">{m.vbEmptyBoard}</p>
        ) : groups.length === 0 ? (
          <p className="mw-muted vb-empty">{m.vbNoMatch}</p>
        ) : (
          <div className={prefs.view === 'grid' ? 'vb-grid-groups' : 'vb-stacks'}>
            {groups.map(g => (
              <section key={g.key} className="vb-group" aria-label={`${groupLabel(g.key, prefs.groupBy, m)} · ${g.count}`}>
                <h3 className="vb-group__title">
                  <span>{groupLabel(g.key, prefs.groupBy, m)}</span>
                  <span className="vb-group__count">{g.count}</span>
                </h3>
                <div className={prefs.view === 'grid' ? 'vb-grid' : 'vb-stack'}>
                  {g.entries.map(e => (
                    <DraggableCard key={`${e.zone}|${e.name}`} entry={e} info={infoOf(e)} m={m} onQty={changeQty} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}

        <DragOverlay>
          {dragged ? (
            <div className="vb-card vb-card--overlay">
              <CardFace entry={dragged} info={draggedInfo} m={m} />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  )
}

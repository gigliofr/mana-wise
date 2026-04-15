import { useEffect, useMemo, useRef, useState } from 'react'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDroppable,
  useSensor,
  useSensors,
  closestCenter,
} from '@dnd-kit/core'
import {
  SortableContext,
  useSortable,
  rectSortingStrategy,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import CardHoverPreview from './CardHoverPreview'
import { apiRequest, throwIfNotOK } from '../lib/apiClient'

const BOARD_MAIN = 'main'
const BOARD_SIDE = 'side'
const BOARD_MAYBE = 'maybe'
const BOARD_ORDER = [BOARD_MAIN, BOARD_SIDE, BOARD_MAYBE]

function boardTitle(board, messages) {
  if (board === BOARD_MAIN) return messages?.builderMainboard || 'Mainboard'
  if (board === BOARD_SIDE) return messages?.builderSideboard || 'Sideboard'
  return messages?.builderMaybeboard || 'Maybeboard'
}

function boardMarker(board) {
  if (board === BOARD_SIDE) return 'Sideboard'
  if (board === BOARD_MAYBE) return 'Maybeboard'
  return ''
}

function badgeClassName(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
}

function parseDecklist(decklist) {
  const rows = []
  const lines = String(decklist || '').split(/\r?\n/)
  let activeBoard = BOARD_MAIN

  for (const raw of lines) {
    const line = raw.trim()
    if (!line || line.startsWith('//')) continue
    const low = line.toLowerCase()

    if (low === 'sideboard' || low === 'sideboard:' || low === 'sb:' || low === 'sb') {
      activeBoard = BOARD_SIDE
      continue
    }
    if (low === 'maybeboard' || low === 'maybeboard:' || low === 'mb:' || low === 'maybe:') {
      activeBoard = BOARD_MAYBE
      continue
    }

    const m = line.match(/^(\d+)x?\s+(.+)$/i)
    if (!m) continue

    rows.push({
      quantity: Math.max(1, Number.parseInt(m[1], 10) || 1),
      name: m[2].trim(),
      board: activeBoard,
    })
  }

  return rows
}

function serializeDecklist(cards) {
  const byBoard = {
    [BOARD_MAIN]: cards.filter(c => c.board === BOARD_MAIN),
    [BOARD_SIDE]: cards.filter(c => c.board === BOARD_SIDE),
    [BOARD_MAYBE]: cards.filter(c => c.board === BOARD_MAYBE),
  }

  const lines = []
  for (const board of BOARD_ORDER) {
    const entries = byBoard[board]
    if (!entries.length) continue
    const marker = boardMarker(board)
    if (marker) {
      if (lines.length) lines.push('')
      lines.push(marker)
    }
    for (const c of entries) {
      lines.push(`${c.quantity} ${c.name}`)
    }
  }

  return lines.join('\n')
}

function SortableCard({ item, token, messages }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id, data: { board: item.board } })
  const meta = item.meta || {}
  const rarity = String(meta.rarity || '').trim().toUpperCase()
  const setCode = String(meta.set_code || '').trim().toUpperCase()
  const rarityClass = badgeClassName(rarity)

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div ref={setNodeRef} style={style} className="builder-card" {...attributes} {...listeners}>
      <div className="builder-card-qty">{item.quantity}x</div>
      <div className="builder-card-name">
        <CardHoverPreview cardName={item.name} token={token} messages={messages} metadata={meta}>
          {item.name}
        </CardHoverPreview>
      </div>
      {(rarity || setCode) && (
        <div className="builder-card-badges">
          {rarity && <span className={`builder-badge rarity-${rarityClass}`}>{rarity}</span>}
          {setCode && <span className="builder-badge builder-badge-set">{setCode}</span>}
        </div>
      )}
    </div>
  )
}

function DroppableBoard({ board, children }) {
  const { setNodeRef, isOver } = useDroppable({
    id: `drop-${board}`,
    data: { board },
  })

  return (
    <div
      ref={setNodeRef}
      className="builder-dropzone"
      style={isOver ? { borderColor: 'var(--primary-h)', background: 'rgba(124,92,191,.08)' } : undefined}
    >
      {children}
    </div>
  )
}

export default function VisualDeckBuilder({ token, messages, decklist, onDeckChange }) {
  const uidRef = useRef(1)
  const [cards, setCards] = useState([])
  const [activeId, setActiveId] = useState(null)
  const [metaByName, setMetaByName] = useState({})
  const [metaLoading, setMetaLoading] = useState(false)
  const [metaError, setMetaError] = useState('')
  const [rarityFilters, setRarityFilters] = useState([])
  const [setFilters, setSetFilters] = useState([])

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }))

  useEffect(() => {
    const parsed = parseDecklist(decklist)
    const next = parsed.map(item => ({
      id: `c-${uidRef.current++}`,
      ...item,
    }))
    setCards(next)
  }, [decklist])

  useEffect(() => {
    let cancelled = false
    const names = Array.from(new Set(cards.map(c => String(c.name || '').trim()).filter(Boolean)))
    if (!token || names.length === 0) {
      setMetaByName({})
      setMetaError('')
      setMetaLoading(false)
      return
    }

    async function loadMetadata() {
      setMetaLoading(true)
      setMetaError('')
      try {
        const { res, data } = await apiRequest('/cards/metadata/batch', {
          token,
          method: 'POST',
          body: { names },
        })
        throwIfNotOK(res, data, messages?.builderMetadataLoadFailed || 'Metadata load failed')
        const nextMap = {}
        for (const item of (data?.items || [])) {
          const key = String(item?.name || '').trim().toLowerCase()
          if (!key) continue
          nextMap[key] = item
        }
        if (!cancelled) {
          setMetaByName(nextMap)
        }
      } catch (err) {
        if (!cancelled) {
          setMetaError(err?.message || messages?.builderMetadataLoadFailed || 'Metadata load failed')
          setMetaByName({})
        }
      } finally {
        if (!cancelled) setMetaLoading(false)
      }
    }

    loadMetadata()
    return () => {
      cancelled = true
    }
  }, [cards, token, messages?.builderMetadataLoadFailed])

  const boardItems = useMemo(() => ({
    [BOARD_MAIN]: cards.filter(c => c.board === BOARD_MAIN),
    [BOARD_SIDE]: cards.filter(c => c.board === BOARD_SIDE),
    [BOARD_MAYBE]: cards.filter(c => c.board === BOARD_MAYBE),
  }), [cards])

  const rarityOptions = useMemo(() => {
    const values = new Set()
    for (const key of Object.keys(metaByName)) {
      const r = String(metaByName[key]?.rarity || '').trim().toLowerCase()
      if (r) values.add(r)
    }
    return Array.from(values).sort()
  }, [metaByName])

  const setOptions = useMemo(() => {
    const values = new Set()
    for (const key of Object.keys(metaByName)) {
      const s = String(metaByName[key]?.set_code || '').trim().toLowerCase()
      if (s) values.add(s)
    }
    return Array.from(values).sort()
  }, [metaByName])

  const matchesFilters = (card) => {
    const key = String(card?.name || '').trim().toLowerCase()
    const meta = metaByName[key]
    if (rarityFilters.length > 0) {
      const rarity = String(meta?.rarity || '').trim().toLowerCase()
      if (!rarityFilters.includes(rarity)) return false
    }
    if (setFilters.length > 0) {
      const setCode = String(meta?.set_code || '').trim().toLowerCase()
      if (!setFilters.includes(setCode)) return false
    }
    return true
  }

  const visibleBoardItems = useMemo(() => ({
    [BOARD_MAIN]: boardItems[BOARD_MAIN].filter(matchesFilters),
    [BOARD_SIDE]: boardItems[BOARD_SIDE].filter(matchesFilters),
    [BOARD_MAYBE]: boardItems[BOARD_MAYBE].filter(matchesFilters),
  }), [boardItems, rarityFilters, setFilters, metaByName])

  function toggleValue(setter, value) {
    setter(prev => prev.includes(value) ? prev.filter(item => item !== value) : [...prev, value])
  }

  const stats = useMemo(() => {
    const count = b => boardItems[b].reduce((sum, c) => sum + c.quantity, 0)
    return {
      main: count(BOARD_MAIN),
      side: count(BOARD_SIDE),
      maybe: count(BOARD_MAYBE),
    }
  }, [boardItems])

  function findBoardByItemId(id) {
    if (!id) return null
    const found = cards.find(c => c.id === id)
    return found?.board || null
  }

  function commit(nextCards) {
    setCards(nextCards)
    onDeckChange?.(serializeDecklist(nextCards))
  }

  function handleDragStart(event) {
    setActiveId(event.active.id)
  }

  function handleDragEnd(event) {
    const { active, over } = event
    setActiveId(null)
    if (!over || !active) return

    const activeIdLocal = active.id
    const overId = over.id

    const fromBoard = findBoardByItemId(activeIdLocal)
    const overCard = cards.find(c => c.id === overId)
    const fromOverData = over?.data?.current?.board
    const toBoard = fromOverData || overCard?.board || fromBoard
    if (!fromBoard || !toBoard) return

    if (fromBoard === toBoard) {
      const list = boardItems[fromBoard]
      const oldIndex = list.findIndex(c => c.id === activeIdLocal)
      const newIndex = list.findIndex(c => c.id === overId)
      if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return

      const moved = arrayMove(list, oldIndex, newIndex)
      const others = cards.filter(c => c.board !== fromBoard)
      commit([...others, ...moved])
      return
    }

    const movedCards = cards.map(c => {
      if (c.id === activeIdLocal) return { ...c, board: toBoard }
      return c
    })

    const targetList = movedCards.filter(c => c.board === toBoard)
    const fromList = movedCards.filter(c => c.board === fromBoard)
    const targetIndex = targetList.findIndex(c => c.id === overId)
    const movedIndex = targetList.findIndex(c => c.id === activeIdLocal)

    let targetReordered = targetList
    if (targetIndex >= 0 && movedIndex >= 0) {
      targetReordered = arrayMove(targetList, movedIndex, targetIndex)
    }

    const untouched = movedCards.filter(c => c.board !== fromBoard && c.board !== toBoard)
    commit([...untouched, ...fromList, ...targetReordered])
  }

  const activeCard = cards.find(c => c.id === activeId)
  const activeMeta = activeCard?.meta || {}
  const activeRarity = String(activeMeta.rarity || '').trim().toUpperCase()
  const activeSetCode = String(activeMeta.set_code || '').trim().toUpperCase()
  const activeRarityClass = badgeClassName(activeRarity)

  return (
    <div className="card">
      <h2>🧩 {messages?.builderTitle || 'Visual Deck Builder'}</h2>
      <p style={{ color: 'var(--muted)', marginTop: -8, marginBottom: 14, fontSize: '.88rem' }}>
        {messages?.builderHint || 'Drag cards between Mainboard, Sideboard and Maybeboard. Changes sync in real time with deck text.'}
      </p>

      <div className="builder-stats-grid">
        <div className="builder-stat"><strong>{stats.main}</strong><span>{boardTitle(BOARD_MAIN, messages)}</span></div>
        <div className="builder-stat"><strong>{stats.side}</strong><span>{boardTitle(BOARD_SIDE, messages)}</span></div>
        <div className="builder-stat"><strong>{stats.maybe}</strong><span>{boardTitle(BOARD_MAYBE, messages)}</span></div>
      </div>

      <div className="form-row-inline" style={{ gap: 8, marginBottom: 12 }}>
        <div className="form-row" style={{ flex: 1, marginBottom: 0 }}>
          <label>{messages?.builderFilterRarity || 'Rarity'}</label>
          <div className="builder-chip-row">
            <button type="button" className={`builder-chip${rarityFilters.length === 0 ? ' active' : ''}`} onClick={() => setRarityFilters([])}>{messages?.builderFilterAll || 'All'}</button>
            {rarityOptions.map(v => (
              <button
                type="button"
                key={v}
                className={`builder-chip${rarityFilters.includes(v) ? ' active' : ''}`}
                onClick={() => toggleValue(setRarityFilters, v)}
              >
                {v}
              </button>
            ))}
          </div>
        </div>
        <div className="form-row" style={{ flex: 1, marginBottom: 0 }}>
          <label>{messages?.builderFilterSet || 'Edition (set)'}</label>
          <div className="builder-chip-row builder-chip-row-scroll">
            <button type="button" className={`builder-chip${setFilters.length === 0 ? ' active' : ''}`} onClick={() => setSetFilters([])}>{messages?.builderFilterAll || 'All'}</button>
            {setOptions.map(v => (
              <button
                type="button"
                key={v}
                className={`builder-chip${setFilters.includes(v) ? ' active' : ''}`}
                onClick={() => toggleValue(setSetFilters, v)}
              >
                {v.toUpperCase()}
              </button>
            ))}
          </div>
        </div>
      </div>
      {(metaLoading || metaError) && (
        <div style={{ color: 'var(--muted)', fontSize: '.82rem', marginBottom: 10 }}>
          {metaLoading ? (messages?.builderMetadataLoading || 'Loading card metadata...') : metaError}
        </div>
      )}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="builder-columns">
          {BOARD_ORDER.map(board => {
            const items = visibleBoardItems[board]
            return (
              <div key={board} className="builder-column" id={board}>
                <div className="builder-column-header">
                  <span>{boardTitle(board, messages)}</span>
                  <span>{items.length}{(boardItems[board].length !== items.length) ? ` / ${boardItems[board].length}` : ''}</span>
                </div>
                <SortableContext items={items.map(i => i.id)} strategy={rectSortingStrategy}>
                  <DroppableBoard board={board}>
                    {items.length === 0 && (
                      <div className="builder-empty">{messages?.builderEmptyColumn || 'Drop cards here'}</div>
                    )}
                    {items.map(item => (
                      <SortableCard
                        key={item.id}
                        item={item}
                        token={token}
                        messages={messages}
                      />
                    ))}
                  </DroppableBoard>
                </SortableContext>
              </div>
            )
          })}
        </div>

        <DragOverlay>
          {activeCard ? (
            <div className="builder-card builder-card-overlay">
              <div className="builder-card-qty">{activeCard.quantity}x</div>
              <div className="builder-card-name">{activeCard.name}</div>
              {(activeRarity || activeSetCode) && (
                <div className="builder-card-badges">
                  {activeRarity && <span className={`builder-badge rarity-${activeRarityClass}`}>{activeRarity}</span>}
                  {activeSetCode && <span className="builder-badge builder-badge-set">{activeSetCode}</span>}
                </div>
              )}
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  )
}

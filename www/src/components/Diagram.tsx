import React, { useId } from "react"

// A small diagram renderer for the "how data models relate" and "how a flow runs" pictures of the docs. The upstream pages use images for
// them; here they are drawn from a description (www/diagrams/*.mjs): boxes placed on a grid, dashed groups, arrows that are routed with
// right angles, and sequence diagrams. Colours come from the docs theme, so the pictures follow the light and dark mode.

type Kind = "box" | "pill" | "decision" | "io" | "text" | "join"
export type DiagramNode = {
  id: string
  at: [number, number]
  kind?: Kind
  title?: string
  lines?: string[]
  // title style of the box: the name of a model is bold, the text of a flow step is not
  plain?: boolean
  // a small annotation above ("note") or below ("noteBelow") the box
  note?: string
  noteBelow?: string
}
type Side = "t" | "b" | "l" | "r"
export type DiagramEdge = {
  from: string
  to: string
  dir?: "->" | "<-" | "<->" | "--"
  label?: string
  fromSide?: Side
  toSide?: Side
  dashed?: boolean
}
export type DiagramGroup = { label: string; nodes: string[]; dashed?: boolean }
export type GridDiagram = {
  alt?: string
  title?: string
  nodes: DiagramNode[]
  edges?: DiagramEdge[]
  groups?: DiagramGroup[]
  gap?: [number, number]
}
export type SequenceDiagram = {
  alt?: string
  title?: string
  type: "sequence"
  actors: string[]
  // from/to are indexes of the actors; a step without `to` is a note over `from`
  steps: { from: number; to: number; text: string; dashed?: boolean }[]
  // frames around a range of steps: [first step, last step]
  frames?: { label: string; steps: [number, number]; actors?: [number, number] }[]
  numbered?: boolean
}
export type DiagramSpec = GridDiagram | SequenceDiagram

const FS = 15
const LH = 22
const cw = (s: string, fs = FS) => s.length * fs * 0.56 + 2

const nodeSize = (n: DiagramNode) => {
  const rows = [...(n.title ? [n.title] : []), ...(n.lines ?? [])]
  const textW = Math.max(0, ...rows.map((r) => cw(r)))
  const gapRows = n.title && n.lines?.length ? 0.55 : 0
  const textH = (rows.length + gapRows) * LH
  switch (n.kind) {
    case "join":
      return { w: 22, h: 22 }
    case "decision":
      return { w: textW + 84, h: Math.max(textH + 64, 90) }
    case "io":
      return { w: textW + 72, h: textH + 28 }
    case "pill":
      return { w: textW + 44, h: Math.max(textH + 20, 40) }
    case "text":
      return { w: textW, h: textH }
    default:
      return { w: textW + 44, h: Math.max(textH + 28, 44) }
  }
}

type Pt = [number, number]
const anchor = (r: { x: number; y: number; w: number; h: number }, side: Side): Pt =>
  side === "l" ? [r.x, r.y + r.h / 2] : side === "r" ? [r.x + r.w, r.y + r.h / 2] : side === "t" ? [r.x + r.w / 2, r.y] : [r.x + r.w / 2, r.y + r.h]

const route = (a: Rect, b: Rect, e: DiagramEdge): Pt[] => {
  const dx = b.x + b.w / 2 - (a.x + a.w / 2)
  const dy = b.y + b.h / 2 - (a.y + a.h / 2)
  let fs: Side
  let ts: Side
  if (e.fromSide && e.toSide) {
    fs = e.fromSide
    ts = e.toSide
  } else if (Math.abs(dy) < 2 || (Math.abs(dx) > Math.abs(dy) * 2.5 && !e.fromSide && !e.toSide && Math.abs(dy) < Math.max(a.h, b.h) / 2)) {
    fs = e.fromSide ?? (dx > 0 ? "r" : "l")
    ts = e.toSide ?? (dx > 0 ? "l" : "r")
  } else if (Math.abs(dx) < 2) {
    fs = e.fromSide ?? (dy > 0 ? "b" : "t")
    ts = e.toSide ?? (dy > 0 ? "t" : "b")
  } else {
    fs = e.fromSide ?? (dx > 0 ? "r" : "l")
    ts = e.toSide ?? (dy > 0 ? "t" : "b")
  }
  const s = anchor(a, fs)
  const t = anchor(b, ts)
  const sh = fs === "l" || fs === "r"
  const th = ts === "l" || ts === "r"
  if (sh && th) {
    if (Math.abs(s[1] - t[1]) < 16) {
      const y = (s[1] + t[1]) / 2
      return [
        [s[0], y],
        [t[0], y],
      ]
    }
    const mx = (s[0] + t[0]) / 2
    return [s, [mx, s[1]], [mx, t[1]], t]
  }
  if (!sh && !th) {
    if (Math.abs(s[0] - t[0]) < 16) {
      const x = (s[0] + t[0]) / 2
      return [
        [x, s[1]],
        [x, t[1]],
      ]
    }
    const my = (s[1] + t[1]) / 2
    return [s, [s[0], my], [t[0], my], t]
  }
  if (sh) return [s, [t[0], s[1]], t]
  return [s, [s[0], t[1]], t]
}

const roundedPath = (pts: Pt[], r = 8) => {
  let d = `M${pts[0][0]},${pts[0][1]}`
  for (let i = 1; i < pts.length - 1; i++) {
    const [px, py] = pts[i - 1]
    const [cx, cy] = pts[i]
    const [nx, ny] = pts[i + 1]
    const l1 = Math.hypot(cx - px, cy - py)
    const l2 = Math.hypot(nx - cx, ny - cy)
    const rr = Math.min(r, l1 / 2, l2 / 2)
    const ax = cx - ((cx - px) / l1) * rr
    const ay = cy - ((cy - py) / l1) * rr
    const bx = cx + ((nx - cx) / l2) * rr
    const by = cy + ((ny - cy) / l2) * rr
    d += ` L${ax},${ay} Q${cx},${cy} ${bx},${by}`
  }
  const last = pts[pts.length - 1]
  return d + ` L${last[0]},${last[1]}`
}

type Rect = { x: number; y: number; w: number; h: number }

const Defs = ({ id }: { id: string }) => (
  <defs>
    <pattern id={`${id}-dots`} width="22" height="22" patternUnits="userSpaceOnUse">
      <circle cx="2" cy="2" r="1" fill="var(--docs-border-strong)" />
    </pattern>
    <marker id={`${id}-arrow`} viewBox="0 0 10 10" refX="9" refY="5" markerWidth="9" markerHeight="9" orient="auto-start-reverse">
      <path d="M0,1 L9,5 L0,9 Z" fill="var(--docs-fg-muted)" />
    </marker>
  </defs>
)

const Frame = ({ id, w, h, title, alt, children }: { id: string; w: number; h: number; title?: string; alt?: string; children: React.ReactNode }) => (
  <figure className="my-docs_1 mx-0 overflow-hidden rounded-docs_DEFAULT border border-medusa-border-base bg-medusa-bg-component">
    <svg
      viewBox={`0 0 ${w} ${h}`}
      role="img"
      aria-label={alt}
      className="block h-auto w-full text-medusa-fg-base"
      style={{ maxWidth: w, margin: "0 auto", fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif", fontSize: FS }}
    >
      <Defs id={id} />
      <rect width={w} height={h} fill={`url(#${id}-dots)`} opacity="0.7" />
      {title && (
        <text x={w / 2} y={30} textAnchor="middle" fill="var(--docs-fg-subtle)">
          {title}
        </text>
      )}
      {children}
    </svg>
  </figure>
)

const NodeShape = ({ n, r }: { n: DiagramNode; r: Rect }) => {
  const stroke = "var(--docs-border-strong)"
  const fill = "var(--docs-bg-base)"
  const rows = [...(n.title ? [{ t: n.title, main: true }] : []), ...(n.lines ?? []).map((t) => ({ t, main: !!n.plain }))]
  const gapRows = n.title && n.lines?.length ? 0.55 : 0
  const top = r.y + r.h / 2 - ((rows.length + gapRows) * LH) / 2
  let shape: React.ReactNode = null
  if (n.kind === "join")
    return (
      <g>
        <circle cx={r.x + 11} cy={r.y + 11} r={10} fill={fill} stroke={stroke} />
        <path d={`M${r.x + 7},${r.y + 7} L${r.x + 15},${r.y + 15} M${r.x + 15},${r.y + 7} L${r.x + 7},${r.y + 15}`} stroke="var(--docs-fg-muted)" strokeWidth={1.3} />
      </g>
    )
  if (n.kind === "pill") shape = <rect x={r.x} y={r.y} width={r.w} height={r.h} rx={r.h / 2} fill={fill} stroke={stroke} />
  else if (n.kind === "decision")
    shape = <polygon points={`${r.x + r.w / 2},${r.y} ${r.x + r.w},${r.y + r.h / 2} ${r.x + r.w / 2},${r.y + r.h} ${r.x},${r.y + r.h / 2}`} fill={fill} stroke={stroke} />
  else if (n.kind === "io") {
    const k = 18
    shape = <polygon points={`${r.x + k},${r.y} ${r.x + r.w},${r.y} ${r.x + r.w - k},${r.y + r.h} ${r.x},${r.y + r.h}`} fill={fill} stroke={stroke} />
  } else if (n.kind !== "text") shape = <rect x={r.x} y={r.y} width={r.w} height={r.h} rx={10} fill={fill} stroke={stroke} />
  return (
    <g>
      {shape}
      {rows.map((row, i) => (
        <text
          key={i}
          x={r.x + r.w / 2}
          y={top + (i + (i > 0 ? gapRows : 0)) * LH + LH * 0.72}
          textAnchor="middle"
          fill={row.main ? "var(--docs-fg-base)" : "var(--docs-fg-subtle)"}
          style={row.main && !n.plain ? { fontWeight: 500 } : undefined}
        >
          {row.t}
        </text>
      ))}
    </g>
  )
}

export const GridDiagramView = ({ spec }: { spec: GridDiagram }) => {
  const id = useId().replace(/:/g, "")
  const [gx, gy] = spec.gap ?? [64, 52]
  const sizes = new Map(spec.nodes.map((n) => [n.id, nodeSize(n)]))
  const cols = new Map<number, number>()
  const rows = new Map<number, number>()
  for (const n of spec.nodes) {
    const c = Math.floor(n.at[0])
    const r = Math.floor(n.at[1])
    const s = sizes.get(n.id)!
    // a node that sits between two grid lines (fractional) does not widen a column
    if (Number.isInteger(n.at[0])) cols.set(c, Math.max(cols.get(c) ?? 0, s.w))
    if (Number.isInteger(n.at[1])) rows.set(r, Math.max(rows.get(r) ?? 0, s.h))
  }
  const maxC = Math.max(...spec.nodes.map((n) => Math.ceil(n.at[0])))
  const maxR = Math.max(...spec.nodes.map((n) => Math.ceil(n.at[1])))
  const colX: number[] = []
  const rowY: number[] = []
  let x = 0
  for (let c = 0; c <= maxC + 1; c++) {
    colX[c] = x
    x += (cols.get(c) ?? 0) + gx
  }
  let y = 0
  for (let r = 0; r <= maxR + 1; r++) {
    rowY[r] = y
    y += (rows.get(r) ?? 0) + gy
  }
  const place = (v: number, start: number[], size: Map<number, number>, gap: number) => {
    const i = Math.floor(v)
    const f = v - i
    return start[i] + f * ((size.get(i) ?? 0) + gap)
  }
  const rects = new Map<string, Rect>()
  for (const n of spec.nodes) {
    const s = sizes.get(n.id)!
    const cellW = cols.get(Math.floor(n.at[0])) ?? s.w
    const cellH = rows.get(Math.floor(n.at[1])) ?? s.h
    const frac0 = n.at[0] - Math.floor(n.at[0])
    const frac1 = n.at[1] - Math.floor(n.at[1])
    const px = place(n.at[0], colX, cols, gx) + (frac0 ? 0 : (cellW - s.w) / 2)
    const py = place(n.at[1], rowY, rows, gy) + (frac1 ? 0 : (cellH - s.h) / 2)
    rects.set(n.id, { x: px, y: py, w: s.w, h: s.h })
  }
  const GP = 20
  const GT = 40
  const groups = (spec.groups ?? []).map((g) => {
    const rs = g.nodes.map((nid) => rects.get(nid)!).filter(Boolean)
    let x0 = Math.min(...rs.map((r) => r.x)) - GP
    const y0 = Math.min(...rs.map((r) => r.y)) - GT
    let x1 = Math.max(...rs.map((r) => r.x + r.w)) + GP
    const y1 = Math.max(...rs.map((r) => r.y + r.h)) + GP
    // a group is at least as wide as its label
    const need = cw(g.label) + 36
    if (x1 - x0 < need) {
      const c = (x0 + x1) / 2
      x0 = c - need / 2
      x1 = c + need / 2
    }
    return { ...g, r: { x: x0, y: y0, w: x1 - x0, h: y1 - y0 } }
  })
  const noteRects: Rect[] = []
  for (const n of spec.nodes) {
    const r = rects.get(n.id)!
    if (n.note) noteRects.push({ x: r.x, y: r.y - 28, w: r.w, h: 20 })
    if (n.noteBelow) noteRects.push({ x: r.x, y: r.y + r.h + 8, w: r.w, h: 20 })
  }
  const all = [...rects.values(), ...groups.map((g) => g.r), ...noteRects]
  const minX = Math.min(...all.map((r) => r.x))
  const minY = Math.min(...all.map((r) => r.y))
  const maxX = Math.max(...all.map((r) => r.x + r.w))
  const maxY = Math.max(...all.map((r) => r.y + r.h))
  const PAD = 36
  const titleH = spec.title ? 34 : 0
  const ox = PAD - minX
  const oy = PAD + titleH - minY
  const W = Math.max(maxX - minX + PAD * 2, spec.title ? cw(spec.title) + PAD * 2 : 0)
  const H = maxY - minY + PAD * 2 + titleH
  const shift = (r: Rect): Rect => ({ x: r.x + ox + (W - (maxX - minX + PAD * 2)) / 2, y: r.y + oy, w: r.w, h: r.h })
  const R = (nid: string) => shift(rects.get(nid)!)
  return (
    <Frame id={id} w={W} h={H} title={spec.title} alt={spec.alt}>
      {groups.map((g, i) => {
        const r = shift(g.r)
        return <rect key={i} x={r.x} y={r.y} width={r.w} height={r.h} rx={16} fill="none" stroke="var(--docs-border-strong)" strokeDasharray={g.dashed === false ? undefined : "5 5"} />
      })}
      {(spec.edges ?? []).map((e, i) => {
        const a = R(e.from)
        const b = R(e.to)
        const pts = route(a, b, e)
        const dir = e.dir ?? "->"
        let best = 0
        let bestLen = -1
        for (let k = 0; k < pts.length - 1; k++) {
          const len = Math.abs(pts[k + 1][0] - pts[k][0]) + Math.abs(pts[k + 1][1] - pts[k][1])
          if (len > bestLen) {
            bestLen = len
            best = k
          }
        }
        const p0 = pts[best]
        const p1 = pts[best + 1]
        const horizontal = Math.abs(p1[1] - p0[1]) < 1
        const lw = e.label ? cw(e.label, 13) : 0
        const lx = horizontal ? (p0[0] + p1[0]) / 2 : p0[0] + 10 + lw / 2
        const ly = horizontal ? p0[1] - 13 : (p0[1] + p1[1]) / 2
        return (
          <g key={i}>
            <path
              d={roundedPath(pts)}
              fill="none"
              stroke="var(--docs-fg-muted)"
              strokeWidth={1.4}
              strokeDasharray={e.dashed ? "5 4" : undefined}
              markerEnd={dir === "->" || dir === "<->" ? `url(#${id}-arrow)` : undefined}
              markerStart={dir === "<-" || dir === "<->" ? `url(#${id}-arrow)` : undefined}
            />
            {e.label && (
              <g>
                <text x={lx} y={ly + 4.5} textAnchor="middle" fontSize={13} fill="var(--docs-fg-subtle)">
                  {e.label}
                </text>
              </g>
            )}
          </g>
        )
      })}
      {groups.map((g, i) => {
        const r = shift(g.r)
        return (
          <g key={i}>
            <rect x={r.x + r.w / 2 - cw(g.label) / 2 - 6} y={r.y + 8} width={cw(g.label) + 12} height={24} rx={6} fill="var(--docs-bg-component)" />
            <text x={r.x + r.w / 2} y={r.y + 25} textAnchor="middle" fill="var(--docs-fg-subtle)">
              {g.label}
            </text>
          </g>
        )
      })}
      {spec.nodes.map((n) => {
        const r = R(n.id)
        return (
          <g key={n.id}>
            <NodeShape n={n} r={r} />
            {n.note && (
              <text x={r.x + r.w / 2} y={r.y - 14} textAnchor="middle" fontSize={13} fill="var(--docs-fg-subtle)">
                {n.note}
              </text>
            )}
            {n.noteBelow && (
              <text x={r.x + r.w / 2} y={r.y + r.h + 22} textAnchor="middle" fontSize={13} fill="var(--docs-fg-subtle)">
                {n.noteBelow}
              </text>
            )}
          </g>
        )
      })}
    </Frame>
  )
}

const SequenceDiagramView = ({ spec }: { spec: SequenceDiagram }) => {
  const id = useId().replace(/:/g, "")
  const n = spec.actors.length
  const lines = spec.steps.map((s) => s.text.split("\n"))
  const gaps = new Array(Math.max(n - 1, 0)).fill(190)
  spec.steps.forEach((s, k) => {
    const a = Math.min(s.from, s.to)
    const b = Math.max(s.from, s.to)
    if (a === b) return
    const need = Math.max(...lines[k].map((l) => cw(l, 14))) + (spec.numbered ? 70 : 50)
    const have = gaps.slice(a, b).reduce((x, y) => x + y, 0)
    if (need > have) for (let i = a; i < b; i++) gaps[i] += (need - have) / (b - a)
  })
  const xs: number[] = []
  let x = 0
  for (let i = 0; i < n; i++) {
    xs[i] = x
    x += gaps[i] ?? 0
  }
  const half = (i: number) => (cw(spec.actors[i]) + 32) / 2
  const PAD = Math.max(56, half(0) + 20, half(n - 1) + 20)
  const top = spec.title ? 56 : 28
  const ys: number[] = []
  const framed = new Set((spec.frames ?? []).map((f) => f.steps[0]))
  let y = top + 90
  lines.forEach((l, i) => {
    y += (i === 0 ? 0 : 54) + (framed.has(i) && i > 0 ? 26 : 0) + (l.length - 1) * 16
    ys[i] = y
  })
  const total = (ys[ys.length - 1] ?? top + 90) + 34
  const W = x + PAD * 2 + (spec.numbered ? 24 : 0)
  const H = total + 30
  const X = (i: number) => PAD + (spec.numbered ? 12 : 0) + xs[i]
  const Y = (i: number) => ys[i]
  return (
    <Frame id={id} w={W} h={H} title={spec.title} alt={spec.alt}>
      {(spec.frames ?? []).map((f, i) => {
        const a = f.actors ? f.actors[0] : 0
        const b = f.actors ? f.actors[1] : n - 1
        const x0 = X(a) - 40
        const x1 = X(b) + 40
        const y0 = Y(f.steps[0]) - 20 - (lines[f.steps[0]].length - 1) * 16 - 38
        const y1 = Y(f.steps[1]) + 26
        return (
          <g key={i}>
            <rect x={x0} y={y0} width={x1 - x0} height={y1 - y0} rx={14} fill="none" stroke="var(--docs-border-strong)" />
            <rect x={x0 + 10} y={y0 + 8} width={cw(f.label, 12.5) + 16} height={22} rx={7} fill="var(--docs-bg-base)" stroke="var(--docs-border-strong)" />
            <text x={x0 + 18} y={y0 + 23} fontSize={12.5} fill="var(--docs-fg-subtle)">
              {f.label}
            </text>
          </g>
        )
      })}
      {spec.actors.map((a, i) => (
        <g key={i}>
          <line x1={X(i)} x2={X(i)} y1={top + 34} y2={total} stroke="var(--docs-border-strong)" strokeDasharray="4 5" />
          <rect x={X(i) - cw(a) / 2 - 16} y={top} width={cw(a) + 32} height={34} rx={17} fill="var(--docs-bg-base)" stroke="var(--docs-border-strong)" />
          <text x={X(i)} y={top + 22} textAnchor="middle" fill="var(--docs-fg-base)">
            {a}
          </text>
        </g>
      ))}
      {spec.steps.map((s, i) => {
        const yy = Y(i)
        const x0 = X(s.from)
        const x1 = X(s.to)
        const dirRight = x1 >= x0
        const mid = (x0 + x1) / 2
        return (
          <g key={i}>
            {s.from !== s.to && (
              <line
                x1={x0}
                x2={x1}
                y1={yy}
                y2={yy}
                stroke="var(--docs-fg-muted)"
                strokeWidth={1.4}
                strokeDasharray={s.dashed ? "5 4" : undefined}
                markerEnd={`url(#${id}-arrow)`}
              />
            )}
            {lines[i].map((l, k) => (
              <text key={k} x={s.from === s.to ? x0 + 10 : mid} y={yy - 9 - (lines[i].length - 1 - k) * 16} textAnchor={s.from === s.to ? "start" : "middle"} fontSize={14} fill="var(--docs-fg-base)">
                {l}
              </text>
            ))}
            {spec.numbered && (
              <g>
                <circle cx={x0 + (dirRight ? -16 : 16)} cy={yy} r={10} fill="var(--docs-bg-base)" stroke="var(--docs-border-strong)" />
                <text x={x0 + (dirRight ? -16 : 16)} y={yy + 4} textAnchor="middle" fontSize={11.5} fill="var(--docs-fg-subtle)">
                  {i + 1}
                </text>
              </g>
            )}
          </g>
        )
      })}
    </Frame>
  )
}

export const Diagram = ({ spec, alt }: { spec: DiagramSpec; alt?: string }) => {
  const s = alt ? { ...spec, alt } : spec
  return "type" in s && s.type === "sequence" ? <SequenceDiagramView spec={s} /> : <GridDiagramView spec={s as GridDiagram} />
}

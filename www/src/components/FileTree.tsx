import React from "react"

// A directory tree card, the replacement of the "directory structure" pictures of the upstream pages. A tree is a text: one entry per
// line, two spaces of indentation per level; an entry that ends with " *" is the file (or directory) the page is about and gets a dot.
type Tree = { title: string; tree: string }

const parse = (tree: string) =>
  tree
    .split("\n")
    .filter((l) => l.trim())
    .map((l) => {
      const depth = (l.length - l.trimStart().length) / 2
      const mark = / \*$/.test(l)
      return { depth, name: l.trim().replace(/ \*$/, ""), mark }
    })

const Arrow = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M3 3v6a2 2 0 0 0 2 2h9M11 8l3 3-3 3" />
  </svg>
)

const Card = ({ title, tree }: Tree) => (
  <div className="min-w-[220px] rounded-[18px] border border-medusa-border-base bg-medusa-bg-base px-docs_1 py-docs_0.75 shadow-elevation-card-rest">
    <div className="mb-docs_0.5 text-compact-medium-plus text-medusa-fg-base">{title}</div>
    <ul className="m-0 list-none p-0 text-compact-medium text-medusa-fg-muted">
      {parse(tree).map((row, i) => (
        <li key={i} className="flex items-center gap-docs_0.5 py-[7px]" style={{ paddingLeft: row.depth * 24 }}>
          {row.depth > 0 && <Arrow />}
          <span className="flex-1 whitespace-nowrap">{row.name}</span>
          {row.mark && <span className="ml-docs_1 h-[10px] w-[10px] shrink-0 rounded-full bg-medusa-tag-blue-icon" aria-label="highlighted" />}
        </li>
      ))}
    </ul>
  </div>
)

const dots = {
  backgroundImage: "radial-gradient(var(--docs-border-strong) 1px, transparent 1px)",
  backgroundSize: "22px 22px",
}

export const FileTree = ({ spec, alt }: { spec: { trees: Tree[] } | Tree; alt?: string }) => {
  const trees = "trees" in spec ? spec.trees : [spec]
  return (
    <figure
      className="my-docs_1 mx-0 flex flex-wrap items-center justify-center gap-docs_1.5 overflow-hidden rounded-docs_DEFAULT border border-medusa-border-base bg-medusa-bg-component px-docs_1 py-docs_2"
      style={dots}
      role="img"
      aria-label={alt}
    >
      {trees.map((t, i) => (
        <React.Fragment key={i}>
          {i > 0 && <span className="text-medusa-fg-muted text-h2">→</span>}
          <Card {...t} />
        </React.Fragment>
      ))}
    </figure>
  )
}

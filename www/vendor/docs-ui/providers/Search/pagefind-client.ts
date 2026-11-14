// MedusaGo has no Algolia: this object answers InstantSearch queries from the
// static Pagefind index generated at build time (dist/pagefind). It returns
// hits shaped like the Algolia DocSearch records the search UI renders.

type PagefindSub = { title: string; url: string; excerpt: string }
type PagefindData = {
  url: string
  excerpt: string
  meta: { title?: string }
  filters?: Record<string, string[]>
  sub_results?: PagefindSub[]
}
type PagefindApi = {
  init: () => Promise<void>
  search: (q: string) => Promise<{ results: { data: () => Promise<PagefindData> }[] }>
}

let loading: Promise<PagefindApi | null> | null = null
const load = () => {
  if (!loading) {
    loading = (async () => {
      try {
        const url = "/pagefind/pagefind.js"
        const api = (await import(/* @vite-ignore */ url)) as PagefindApi
        await api.init()
        return api
      } catch {
        return null
      }
    })()
  }
  return loading
}

const HIGHLIGHT = { open: "__ais-highlight__", close: "__/ais-highlight__" }
const toHighlight = (html: string) =>
  html
    .replace(/<mark>/g, HIGHLIGHT.open)
    .replace(/<\/mark>/g, HIGHLIGHT.close)
    .replace(/<[^>]+>/g, "")

const empty = (query: string) => ({
  hits: [],
  nbHits: 0,
  nbPages: 0,
  page: 0,
  hitsPerPage: 0,
  processingTimeMS: 0,
  query,
  params: "",
})

type Request = { indexName?: string; query?: string; params?: { query?: string; hitsPerPage?: number } }

async function searchOne(request: Request) {
  const query = (request.query ?? request.params?.query ?? "").trim()
  const limit = request.params?.hitsPerPage || 20
  if (!query) {
    return empty(query)
  }
  const pagefind = await load()
  if (!pagefind) {
    return empty(query)
  }
  const found = await pagefind.search(query)
  const pages = await Promise.all(found.results.slice(0, 12).map((r) => r.data()))
  const hits: Record<string, unknown>[] = []
  pages.forEach((page, p) => {
    const title = page.meta.title || page.url
    const section = page.filters?.section?.[0] || null
    const subs = page.sub_results?.length
      ? page.sub_results
      : [{ title, url: page.url, excerpt: page.excerpt }]
    subs.slice(0, 3).forEach((sub, s) => {
      const isHeading = sub.title && sub.title !== title
      const value = toHighlight(sub.excerpt)
      hits.push({
        objectID: `${page.url}#${s}`,
        url: sub.url,
        url_without_anchor: page.url,
        type: "content",
        content: value.replace(/__\/?ais-highlight__/g, ""),
        hierarchy: { lvl0: section, lvl1: title, lvl2: isHeading ? sub.title : null, lvl3: null, lvl4: null, lvl5: null },
        _tags: [],
        __position: p * 3 + s + 1,
        _snippetResult: {
          content: { value, matchLevel: "full" },
          hierarchy: {
            lvl1: { value: title, matchLevel: "none" },
            lvl2: { value: isHeading ? sub.title : "", matchLevel: "none" },
          },
        },
        _highlightResult: {},
      })
    })
  })
  return {
    ...empty(query),
    hits: hits.slice(0, limit),
    nbHits: Math.min(hits.length, limit),
    nbPages: 1,
    hitsPerPage: limit,
  }
}

export const pagefindClient = {
  search: async (requests: Request[]) => ({ results: await Promise.all(requests.map(searchOne)) }),
  searchForFacetValues: async (requests: unknown[]) => ({ results: requests.map(() => ({ facetHits: [] })) }),
  addAlgoliaAgent: () => {},
}

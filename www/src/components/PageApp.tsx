import React, { Suspense, useEffect } from "react"
import { MDXProvider } from "@mdx-js/react"
import { usePathname, setPath, registerNavigator, noTrailingSlash } from "next/navigation"
import { Frame } from "./Frame"
import { mdxComponents } from "./MDXComponents"
import { sections, type SectionKey } from "~/config/sections"

type PageModule = { default: React.ComponentType<{ components?: unknown }> }
const loaders = import.meta.glob("/content/**/page.mdx") as Record<string, () => Promise<PageModule>>
const cache = new Map<string, React.LazyExoticComponent<React.ComponentType<{ components?: unknown }>>>()

type Route = { file: string; section: SectionKey }

// Every docs page by its path, to navigate without leaving the page (see the router below).
const routes = new Map<string, Route>()
for (const file of Object.keys(loaders)) {
  const [, , section, ...rest] = file.split("/")
  rest.pop() // page.mdx
  routes.set("/" + [section, ...rest].join("/"), { file, section: section as SectionKey })
}

// The loaded page modules. In the browser the module of the current page is loaded BEFORE this island hydrates (top-level await): if React had
// to wait for it inside the page, it would swap the server-rendered article for the empty Suspense fallback until the module arrives, and the
// page would flash blank. The router loads the module of the next page before it switches to it, for the same reason.
const loaded = new Map<string, React.ComponentType<{ components?: unknown }>>()
const ensureLoaded = async (file: string) => {
  if (!loaded.has(file)) loaded.set(file, (await loaders[file]()).default)
}
if (typeof window !== "undefined") {
  const current = (window as { __PAGE_FILE__?: string }).__PAGE_FILE__
  if (current && loaders[current]) await ensureLoaded(current)
}

// While building, the Astro page loads the MDX module up front so the content is rendered inline in the HTML (no streaming placeholder).
const pageComponent = (file: string) => {
  if (typeof window === "undefined") {
    const ready = (globalThis as { __PAGE_CACHE__?: Map<string, React.ComponentType<{ components?: unknown }>> }).__PAGE_CACHE__?.get(file)
    if (ready) return ready
  }
  const ready = loaded.get(file)
  if (ready) return ready
  let c = cache.get(file)
  if (!c) {
    c = React.lazy(loaders[file])
    cache.set(file, c)
  }
  return c
}

const routeOf = (url: URL): Route | undefined => (url.origin === window.location.origin ? routes.get(noTrailingSlash(url.pathname)) : undefined)

// Moves to another docs page without a page load: the sidebar, the header and the table of contents stay mounted (like the client-side
// navigation of the Next.js original), and only the article changes. Returns false for anything that is not a docs page.
const navigate = (url: URL, replace: boolean) => {
  const route = routeOf(url)
  if (!route) return false
  ensureLoaded(route.file)
    .then(() => {
      const path = noTrailingSlash(url.pathname)
      const same = path === noTrailingSlash(window.location.pathname)
      window.history[replace || same ? "replaceState" : "pushState"](null, "", path + "/" + url.search + url.hash)
      setPath(path)
    })
    .catch(() => window.location.assign(url.href)) // e.g. offline: let the browser report it
  return true
}

const anchorOf = (event: Event) => (event.target as Element | null)?.closest?.("a") as HTMLAnchorElement | null

const useClientRouter = () => {
  useEffect(() => {
    const unregister = registerNavigator((href, options) => navigate(new URL(href, window.location.href), !!options?.replace))

    const onClick = (event: MouseEvent) => {
      if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
      const a = anchorOf(event)
      if (!a || (a.target && a.target !== "_self") || a.hasAttribute("download")) return
      const href = a.getAttribute("href")
      if (!href || href.startsWith("#")) return // the browser scrolls to the anchor of this page
      const url = new URL(a.href)
      // a link to a heading of the current page: let the browser scroll to it
      if (url.hash && noTrailingSlash(url.pathname) === noTrailingSlash(window.location.pathname)) return
      if (navigate(url, false)) event.preventDefault()
    }
    // load the page before the click, so the click switches at once
    const prefetch = (event: Event) => {
      const a = anchorOf(event)
      if (!a || !a.href) return
      const route = routeOf(new URL(a.href))
      if (route) void ensureLoaded(route.file).catch(() => {})
    }
    const onPopState = () => {
      const route = routes.get(noTrailingSlash(window.location.pathname))
      if (!route) return window.location.reload()
      ensureLoaded(route.file)
        .then(() => setPath(window.location.pathname))
        .catch(() => window.location.reload())
    }
    document.addEventListener("click", onClick)
    document.addEventListener("mouseover", prefetch, { passive: true })
    document.addEventListener("touchstart", prefetch, { passive: true })
    window.addEventListener("popstate", onPopState)
    return () => {
      unregister()
      document.removeEventListener("click", onClick)
      document.removeEventListener("mouseover", prefetch)
      document.removeEventListener("touchstart", prefetch)
      window.removeEventListener("popstate", onPopState)
    }
  }, [])
}

// The head of a page that was opened by the router: the canonical address and the title of its section.
const updateHead = (path: string, section: SectionKey) => {
  const config = sections[section]
  const site = "https://medusago.hitkey.io"
  const set = (selector: string, attribute: string, value: string) => document.querySelector(selector)?.setAttribute(attribute, value)
  document.title = config.titleSuffix
  set('link[rel="canonical"]', "href", site + path + "/")
  set('meta[property="og:url"]', "content", site + path + "/")
  for (const [selector, value] of [
    ['meta[name="description"]', config.description],
    ['meta[property="og:title"]', config.titleSuffix],
    ['meta[property="og:description"]', config.description],
    ['meta[name="twitter:title"]', config.titleSuffix],
    ['meta[name="twitter:description"]', config.description],
  ]) set(selector, "content", value)
}

type Props = { file: string; section: SectionKey; path: string }

export default function PageApp({ file, section, path }: Props) {
  // the "server snapshot" of usePathname: the path of the page that the server rendered, which React also uses while it hydrates it
  ;(globalThis as { __ASTRO_PATH__?: string }).__ASTRO_PATH__ = path
  useClientRouter()
  const current = usePathname()
  // the server and the first render use the props of the Astro page; after a navigation the path decides
  const route = routes.get(current) ?? { file, section }
  const opened = React.useRef(current)
  useEffect(() => {
    if (opened.current === current) return // the page that the server rendered has its head already
    opened.current = current
    updateHead(current, route.section)
  }, [current])

  const Content = pageComponent(route.file)
  // The page module is always loaded before it is shown (on the server, before the browser hydrates, and by the router), so there is nothing to
  // wait for. Without a Suspense boundary the article hydrates in the same pass as the frame around it, with the same state: React hydrates a
  // boundary in a later pass, after the effects of the frame have run, and the article would no longer match the HTML of the server.
  const ready = typeof window === "undefined" || loaded.has(route.file)
  return (
    // a new section has its own sidebar and navigation: start those over; inside a section the frame stays
    <Frame key={route.section} config={sections[route.section]} file={route.file} showPagination={route.section !== "resources"}>
      <MDXProvider components={mdxComponents}>
        {ready ? (
          <Content />
        ) : (
          <Suspense fallback={null}>
            <Content />
          </Suspense>
        )}
      </MDXProvider>
    </Frame>
  )
}

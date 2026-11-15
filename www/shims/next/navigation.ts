// Replacement for next/navigation on a static Astro site. The page island (src/components/PageApp.tsx) is a small client-side router:
// it keeps the current path in this store, so the sidebar, the header and the table of contents stay mounted between docs pages, like in the
// Next.js original, and only the article changes.
import { useSyncExternalStore } from "react"

// Next.js normalises away the trailing slash that static hosting adds.
export const noTrailingSlash = (path: string) => (path.length > 1 ? path.replace(/\/+$/, "") : path)

const serverPath = () => noTrailingSlash((globalThis as { __ASTRO_PATH__?: string }).__ASTRO_PATH__ || "/")

let path: string | null = null
const listeners = new Set<() => void>()
const getPath = () => (path ??= noTrailingSlash(window.location.pathname))
const subscribe = (listener: () => void) => {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** Sets the current path and tells everyone who reads it (usePathname). */
export const setPath = (next: string) => {
  const normalized = noTrailingSlash(next)
  if (normalized === path) return
  path = normalized
  listeners.forEach((listener) => listener())
}

export const usePathname = () => useSyncExternalStore(subscribe, getPath, serverPath)

// The router of the page island registers itself here; it answers true when it handled the navigation (a docs page), and false otherwise.
type Navigator = (url: string, options?: { replace?: boolean }) => boolean
let navigator: Navigator | null = null
export const registerNavigator = (next: Navigator) => {
  navigator = next
  return () => {
    if (navigator === next) navigator = null
  }
}

const go = (url: string, replace: boolean) => {
  if (url.startsWith("#")) {
    if (replace) window.history.replaceState(null, "", url)
    else window.location.hash = url.slice(1)
    return
  }
  if (navigator?.(url, { replace })) return
  if (replace) window.location.replace(url)
  else window.location.assign(url)
}

// one object, like the router of Next.js, so it is safe as a hook dependency
const router = {
  push: (url: string, _options?: unknown) => go(url, false),
  replace: (url: string, _options?: unknown) => go(url, true),
  back: () => window.history.back(),
  forward: () => window.history.forward(),
  refresh: () => window.location.reload(),
  prefetch: (_url?: string) => {},
}
export const useRouter = () => router

export const useSearchParams = () =>
  new URLSearchParams(typeof window === "undefined" ? "" : window.location.search)

// Minimal replacement for next/navigation on a static Astro site.
import { useEffect, useState } from "react"

// Next.js normalises away the trailing slash that static hosting adds.
const noTrailingSlash = (path: string) => (path.length > 1 ? path.replace(/\/+$/, "") : path)

const currentPath = () =>
  noTrailingSlash(
    typeof window === "undefined"
      ? (globalThis as { __ASTRO_PATH__?: string }).__ASTRO_PATH__ || "/"
      : window.location.pathname
  )

export const usePathname = () => {
  const [path, setPath] = useState(currentPath())
  useEffect(() => {
    const update = () => setPath(currentPath())
    window.addEventListener("popstate", update)
    return () => window.removeEventListener("popstate", update)
  }, [])
  return path
}

export const useRouter = () => ({
  push: (url: string) => {
    window.location.assign(url)
  },
  replace: (url: string) => {
    window.location.replace(url)
  },
  back: () => window.history.back(),
  forward: () => window.history.forward(),
  refresh: () => window.location.reload(),
  prefetch: () => {},
})

export const useSearchParams = () =>
  new URLSearchParams(typeof window === "undefined" ? "" : window.location.search)

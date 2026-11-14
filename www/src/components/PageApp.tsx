import React, { Suspense } from "react"
import { MDXProvider } from "@mdx-js/react"
import { Frame } from "./Frame"
import { mdxComponents } from "./MDXComponents"
import { sections, type SectionKey } from "~/config/sections"

const loaders = import.meta.glob("/content/**/page.mdx") as Record<
  string,
  () => Promise<{ default: React.ComponentType<{ components?: unknown }> }>
>
const cache = new Map<string, React.LazyExoticComponent<React.ComponentType<{ components?: unknown }>>>()

// While building, the Astro page loads the MDX module up front so the content is
// rendered inline in the HTML (no streaming placeholder). In the browser the
// module is loaded lazily and the server HTML stays in place until it arrives.
const pageComponent = (file: string) => {
  if (typeof window === "undefined") {
    const ready = (globalThis as { __PAGE_CACHE__?: Map<string, React.ComponentType<{ components?: unknown }>> }).__PAGE_CACHE__?.get(file)
    if (ready) return ready
  }
  let c = cache.get(file)
  if (!c) {
    c = React.lazy(loaders[file])
    cache.set(file, c)
  }
  return c
}

type Props = { file: string; section: SectionKey; path: string }

export default function PageApp({ file, section, path }: Props) {
  if (typeof window === "undefined") {
    ;(globalThis as { __ASTRO_PATH__?: string }).__ASTRO_PATH__ = path
  }
  const Content = pageComponent(file)
  return (
    <Frame config={sections[section]} file={file} showPagination={section !== "resources"}>
      <MDXProvider components={mdxComponents}>
        <Suspense fallback={null}>
          <Content />
        </Suspense>
      </MDXProvider>
    </Frame>
  )
}

import React, { Suspense } from "react"
import { Frame } from "./Frame"
import { mdxComponents } from "./MDXComponents"
import { sections, type SectionKey } from "~/config/sections"

const loaders = import.meta.glob("/content/**/page.mdx") as Record<
  string,
  () => Promise<{ default: React.ComponentType<{ components?: unknown }> }>
>
const cache = new Map<string, React.LazyExoticComponent<React.ComponentType<{ components?: unknown }>>>()

const pageComponent = (file: string) => {
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
  ;(globalThis as { __BASE_PATH__?: string }).__BASE_PATH__ = sections[section].basePath
  const Content = pageComponent(file)
  return (
    <Frame config={sections[section]}>
      <Suspense fallback={null}>
        <Content components={mdxComponents} />
      </Suspense>
    </Frame>
  )
}

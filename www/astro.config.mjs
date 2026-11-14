import { defineConfig } from "astro/config"
import react from "@astrojs/react"
import mdx from "@astrojs/mdx"
import path from "node:path"
import { fileURLToPath } from "node:url"

const root = path.dirname(fileURLToPath(import.meta.url))
const r = (p) => path.join(root, p)

export default defineConfig({
  site: "https://medusago.hitkey.io",
  integrations: [react(), mdx()],
  vite: {
    resolve: {
      alias: [
        { find: /^next\/link$/, replacement: r("shims/next/link.tsx") },
        { find: /^next\/image$/, replacement: r("shims/next/image.tsx") },
        { find: /^next\/dynamic$/, replacement: r("shims/next/dynamic.tsx") },
        { find: /^next\/navigation$/, replacement: r("shims/next/navigation.ts") },
        { find: /^@kapaai\/react-sdk$/, replacement: r("shims/kapa.ts") },
        { find: /^docs-ui$/, replacement: r("vendor/docs-ui/index.ts") },
        { find: /^docs-utils\/global-config$/, replacement: r("vendor/docs-utils/global-config.ts") },
        { find: /^docs-utils$/, replacement: r("vendor/docs-utils/index.ts") },
        { find: /^types$/, replacement: r("vendor/types/index.ts") },
        { find: /^@\//, replacement: r("vendor/docs-ui") + "/" },
        { find: /^~\//, replacement: r("src") + "/" },
      ],
    },
    ssr: { noExternal: ["@medusajs/ui", "@medusajs/icons", "react-tooltip", "react-medium-image-zoom"] },
  },
})

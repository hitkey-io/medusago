import { defineConfig } from "astro/config"
import { transformWithEsbuild } from "vite"
import react from "@astrojs/react"
import mdx from "@mdx-js/rollup"
import { mdxOptions } from "./mdx.options.mjs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const root = path.dirname(fileURLToPath(import.meta.url))
const r = (p) => path.join(root, p)

export default defineConfig({
  site: "https://medusago.hitkey.io",
  integrations: [react()],
  vite: {
    plugins: [
      { enforce: "pre", ...mdx({ ...mdxOptions, jsx: true }) },
      {
        // The recma plugins emit JSX: compile it after MDX.
        name: "mdx-jsx",
        enforce: "pre",
        async transform(code, id) {
          if (!id.endsWith(".mdx")) return
          const out = await transformWithEsbuild(code, id + ".jsx", { loader: "jsx", jsx: "automatic" })
          return { code: out.code, map: out.map }
        },
      },
    ],
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

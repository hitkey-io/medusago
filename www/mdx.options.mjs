// MDX pipeline: the Medusa docs remark/rehype/recma plugins (vendored), the
// same chain the upstream sites use.
import remarkFrontmatter from "remark-frontmatter"
import remarkDirective from "remark-directive"
import rehypeSlug from "rehype-slug"
import rehypeMdxCodeProps from "rehype-mdx-code-props"
import withExtractedTableOfContents from "@stefanprobst/rehype-extract-toc"
import { remarkIncludeCode } from "./plugins/include-code.mjs"
import { remarkPageNumber } from "./plugins/page-number.mjs"
import {
  crossProjectLinksPlugin,
  localLinksRehypePlugin,
  resolveAdmonitionsPlugin,
  remarkAttachFrontmatterDataPlugin,
  recmaInjectMdxDataPlugin,
  validateHighlightsPlugin,
  mdContentOverrideRehypePlugin,
} from "./vendor/remark-rehype-plugins/index.ts"

// localLinksRehypePlugin resolves "./x/page.mdx" links against <cwd>/app; the
// pages of this site live in content/<section>.
function localLinksBySection() {
  return async (tree, file) => {
    const m = /^(.*\/content\/([^/]+))\//.exec(file.history?.[0] || "")
    const { visit } = await import("unist-util-visit")
    const nodes = []
    visit(tree, ["element", "link"], (node) => {
      const href = node.tagName === "a" ? node.properties?.href : node.type === "link" ? node.url : null
      if (href && /page\.mdx?/.test(href)) nodes.push(node)
    })
    await localLinksRehypePlugin({ basePath: m?.[1] })(tree, file)
    // the plugin returns paths relative to the section: make them site paths
    for (const node of nodes) {
      const key = node.tagName === "a" ? "href" : "url"
      const holder = node.tagName === "a" ? node.properties : node
      const link = holder[key]
      if (m && typeof link === "string" && link.startsWith("/")) holder[key] = `/${m[2]}${link === "/" ? "" : link}`
    }
  }
}


export const mdxOptions = {
  jsxImportSource: "react",
  // partials (imported *.mdx files) get the components from the provider too
  providerImportSource: "@mdx-js/react",
  remarkPlugins: [
    [remarkIncludeCode],
    [remarkPageNumber],
    [remarkFrontmatter],
    [remarkDirective],
    [resolveAdmonitionsPlugin],
    [remarkAttachFrontmatterDataPlugin],
  ],
  rehypePlugins: [
    [
      crossProjectLinksPlugin,
      {
        baseUrl: "",
        projectUrls: { docs: { url: "", path: "" } },
        useBaseUrl: true,
      },
    ],
    [localLinksBySection],
    [rehypeMdxCodeProps, { tagName: "code" }],
    [validateHighlightsPlugin, { verbose: false }],
    [rehypeSlug],
    [withExtractedTableOfContents],
    [mdContentOverrideRehypePlugin],
  ],
  recmaPlugins: [[recmaInjectMdxDataPlugin]],
}

// MDX pipeline: the Medusa docs remark/rehype/recma plugins (vendored), the
// same chain the upstream sites use.
import remarkFrontmatter from "remark-frontmatter"
import remarkDirective from "remark-directive"
import rehypeSlug from "rehype-slug"
import rehypeMdxCodeProps from "rehype-mdx-code-props"
import withExtractedTableOfContents from "@stefanprobst/rehype-extract-toc"
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
    const m = /^(.*\/content\/[^/]+)\//.exec(file.history?.[0] || "")
    return localLinksRehypePlugin({ basePath: m?.[1] })(tree, file)
  }
}

export const siteUrl = process.env.PUBLIC_BASE_URL || "https://medusago.hitkey.io"

export const mdxOptions = {
  jsxImportSource: "react",
  remarkPlugins: [
    [remarkFrontmatter],
    [remarkDirective],
    [resolveAdmonitionsPlugin],
    [remarkAttachFrontmatterDataPlugin],
  ],
  rehypePlugins: [
    [
      crossProjectLinksPlugin,
      {
        baseUrl: siteUrl,
        projectUrls: { docs: { url: siteUrl, path: "/learn" } },
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

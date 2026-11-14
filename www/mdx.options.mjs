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
    [localLinksRehypePlugin],
    [rehypeMdxCodeProps, { tagName: "code" }],
    [validateHighlightsPlugin, { verbose: false }],
    [rehypeSlug],
    [withExtractedTableOfContents],
    [mdContentOverrideRehypePlugin],
  ],
  recmaPlugins: [[recmaInjectMdxDataPlugin]],
}

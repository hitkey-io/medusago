import type { MDXComponents as MDXComponentsType } from "mdx/types"
import { MDXComponents as UiMdxComponents, InlineIcon, InlineThemeImage, H1 } from "docs-ui"

export const mdxComponents: MDXComponentsType = {
  ...UiMdxComponents,
  InlineIcon,
  InlineThemeImage,
  h1: (props) => <H1 variant="content" {...props} />,
}

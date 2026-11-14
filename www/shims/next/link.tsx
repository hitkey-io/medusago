import React from "react"

type LinkProps = Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, "href"> & {
  href: string | { pathname?: string }
  prefetch?: boolean
  scroll?: boolean
  replace?: boolean
}

// Next.js prefixes internal links with the app's basePath; the docs sections
// of this site live under /<section>, set by the page before rendering.
const withBasePath = (href: string) => {
  const base = (globalThis as { __BASE_PATH__?: string }).__BASE_PATH__ || ""
  if (!base || !href.startsWith("/") || href.startsWith("//")) return href
  if (href === base || href.startsWith(base + "/") || href.startsWith(base + "#")) return href
  return base + (href === "/" ? "" : href)
}

const Link = React.forwardRef<HTMLAnchorElement, LinkProps>(
  ({ href, prefetch: _p, scroll: _s, replace: _r, ...rest }, ref) => (
    <a
      ref={ref}
      href={withBasePath(typeof href === "string" ? href : href.pathname || "")}
      {...rest}
    />
  )
)
Link.displayName = "Link"
export default Link

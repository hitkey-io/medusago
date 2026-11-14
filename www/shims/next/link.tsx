import React from "react"

type LinkProps = Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, "href"> & {
  href: string | { pathname?: string }
  prefetch?: boolean
  scroll?: boolean
  replace?: boolean
}

const Link = React.forwardRef<HTMLAnchorElement, LinkProps>(
  ({ href, prefetch: _p, scroll: _s, replace: _r, ...rest }, ref) => (
    <a
      ref={ref}
      href={typeof href === "string" ? href : href.pathname || ""}
      {...rest}
    />
  )
)
Link.displayName = "Link"
export default Link

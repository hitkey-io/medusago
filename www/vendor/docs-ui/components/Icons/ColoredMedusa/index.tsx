import clsx from "clsx"
import React from "react"

type MedusaIconProps = React.SVGProps<SVGSVGElement> & {
  variant?: "base" | "subtle" | "muted"
}

// MedusaGo symbol: hexagon shell (follows the text colour of the current
// theme) around the cyan Go core.
export const ColoredMedusaIcon = ({
  className,
  variant = "base",
  ...props
}: MedusaIconProps) => {
  return (
    <svg
      viewBox="0 0 256 256"
      width={26}
      height={26}
      role="img"
      aria-label="MedusaGo"
      {...props}
      className={className}
    >
      <path
        fillRule="evenodd"
        className={clsx(
          variant === "base" && "fill-medusa-fg-base",
          variant === "subtle" && "fill-medusa-fg-base",
          variant === "muted" && "fill-medusa-fg-muted"
        )}
        d="M120.5 25.01A15 15 0 0 1 135.5 25.01L213.44 70.01A15 15 0 0 1 220.94 83L220.94 173A15 15 0 0 1 213.44 185.99L135.5 230.99A15 15 0 0 1 120.5 230.99L42.56 185.99A15 15 0 0 1 35.06 173L35.06 83A15 15 0 0 1 42.56 70.01L120.5 25.01ZM128 55.32L190.94 91.66L190.94 164.34L128 200.68L65.06 164.34L65.06 91.66Z"
      />
      <circle cx="128" cy="128" r="36" fill="#00ADD8" />
    </svg>
  )
}

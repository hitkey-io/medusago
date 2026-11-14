import React, { Suspense } from "react"

type Loader<P> = () => Promise<{ default: React.ComponentType<P> } | React.ComponentType<P>>

export default function dynamic<P extends object>(
  loader: Loader<P>,
  _options?: { ssr?: boolean; loading?: () => React.ReactNode }
) {
  const Lazy = React.lazy(async () => {
    const mod = await loader()
    return "default" in (mod as object) ? (mod as { default: React.ComponentType<P> }) : { default: mod as React.ComponentType<P> }
  })
  return (props: P) => (
    <Suspense fallback={null}>
      <Lazy {...(props as React.JSX.IntrinsicAttributes & P)} />
    </Suspense>
  )
}

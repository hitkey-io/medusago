import React from "react"
import clsx from "clsx"
import { Breadcrumbs, RootProviders, Sidebar, Footer } from "docs-ui"
import { MainContentLayout } from "@/layouts/main-content"
import { Providers } from "./Providers"

export const Frame = ({ children }: { children?: React.ReactNode }) => (
  <div
    className={clsx(
      "bg-medusa-bg-subtle font-base text-medium w-full",
      "text-medusa-fg-base",
      "h-full overflow-hidden",
      "grid grid-cols-1 lg:mx-auto lg:grid-cols-[221px_1fr]"
    )}
    id="root-layout"
  >
    <RootProviders>
      <Providers>
        <Sidebar />
        <div className={clsx("relative", "h-screen", "flex")}>
          <MainContentLayout>
            <div
              className={clsx(
                "w-full h-fit",
                "max-w-inner-content-xs sm:max-w-inner-content-sm md:max-w-inner-content-md",
                "lg:max-w-inner-content-lg xl:max-w-inner-content-xl xxl:max-w-inner-content-xxl",
                "xxxl:max-w-inner-content-xxxl",
                "px-docs_1 md:px-docs_4 lg:px-0"
              )}
            >
              <Breadcrumbs />
              {children}
              <Footer showPagination />
            </div>
          </MainContentLayout>
        </div>
      </Providers>
    </RootProviders>
  </div>
)

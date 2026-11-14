import React from "react"
import clsx from "clsx"
import { MainNav, RootProviders } from "docs-ui"
import { Providers } from "./Providers"
import { homeConfig } from "~/config/sections"
import HomepageTopSection from "./Homepage/TopSection"
import HomepageSectionsSeparator from "./Homepage/SectionsSeparator"
import HomepageStartPrompt from "./Homepage/StartPrompt"
import HomepageLinksSection from "./Homepage/LinksSection"
import HomepageFrameworkSection from "./Homepage/FrameworkSection"
import HomepageCommerceModulesSection from "./Homepage/CommerceModulesSection"
import HomepageFooter from "./Homepage/Footer"

export default function HomeApp() {
  if (typeof window === "undefined") {
    ;(globalThis as { __ASTRO_PATH__?: string }).__ASTRO_PATH__ = "/"
  }
  return (
    <RootProviders layoutProviderProps={{ disableResizeObserver: true }}>
      <Providers config={homeConfig}>
        <div
          className={clsx(
            "bg-medusa-bg-base font-base text-medium text-medusa-fg-base",
            "shadow-elevation-card-rest dark:shadow-elevation-card-rest-dark",
            "h-full w-full overflow-y-scroll overflow-x-hidden"
          )}
          id="main"
        >
          <MainNav className="border-b border-medusa-border-base bg-medusa-bg-component" />
          <div
            className={clsx(
              "xl:mx-auto xl:max-w-[1026px] w-full",
              "flex flex-col justify-center items-start",
              "xl:border-x xl:border-medusa-border-base"
            )}
          >
            <HomepageSectionsSeparator />
            <HomepageTopSection />
            <HomepageStartPrompt />
            <HomepageSectionsSeparator />
            <HomepageLinksSection />
            <HomepageSectionsSeparator />
            <HomepageFrameworkSection />
            <HomepageSectionsSeparator />
            <HomepageCommerceModulesSection />
            <HomepageSectionsSeparator />
            <HomepageFooter />
          </div>
        </div>
      </Providers>
    </RootProviders>
  )
}

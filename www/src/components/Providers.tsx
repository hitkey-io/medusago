import React, { useMemo } from "react"
import {
  AnalyticsProvider,
  ColorModeProvider,
  getNavDropdownItems,
  HooksLoader,
  LearningPathProvider,
  MainNavProvider,
  MobileProvider,
  ModalProvider,
  NotificationProvider,
  PaginationProvider,
  ScrollControllerProvider,
  SearchProvider,
  SidebarProvider,
  SiteConfigProvider,
  useScrollController,
} from "docs-ui"
import type { DocsConfig } from "types"

const Sidebar = ({ children, config }: { children?: React.ReactNode; config: DocsConfig }) => {
  const { scrollableElement } = useScrollController()
  return (
    <SidebarProvider
      scrollableElement={scrollableElement}
      sidebars={config.sidebars}
    >
      {children}
    </SidebarProvider>
  )
}

const MainNav = ({ children, config }: { children?: React.ReactNode; config: DocsConfig }) => {
  const items = useMemo(
    () => getNavDropdownItems({ basePath: config.baseUrl }),
    [config.baseUrl]
  )
  return <MainNavProvider navItems={items}>{children}</MainNavProvider>
}

export const Providers = ({ children, config }: { children?: React.ReactNode; config: DocsConfig }) => (
  <AnalyticsProvider>
    <SiteConfigProvider config={config}>
      <MobileProvider>
        <ColorModeProvider>
          <ModalProvider>
            <LearningPathProvider>
              <NotificationProvider>
                <ScrollControllerProvider scrollableSelector="#main">
                  <Sidebar config={config}>
                    <PaginationProvider>
                      <MainNav config={config}>
                        <SearchProvider
                          algolia={{ appId: "temp", apiKey: "temp", mainIndexName: "temp" }}
                          indices={[{ value: "temp", title: "Docs" }]}
                          defaultIndex="temp"
                          searchProps={{
                            isLoading: false,
                            suggestions: [],
                            checkInternalPattern: new RegExp(`^${config.basePath || ""}`),
                          }}
                        >
                          <HooksLoader
                            options={{
                              pageScrollManager: true,
                              currentLearningPath: false,
                            }}
                          >
                            {children}
                          </HooksLoader>
                        </SearchProvider>
                      </MainNav>
                    </PaginationProvider>
                  </Sidebar>
                </ScrollControllerProvider>
              </NotificationProvider>
            </LearningPathProvider>
          </ModalProvider>
        </ColorModeProvider>
      </MobileProvider>
    </SiteConfigProvider>
  </AnalyticsProvider>
)

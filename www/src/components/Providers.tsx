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
import { config } from "~/config/site"

const Sidebar = ({ children }: { children?: React.ReactNode }) => {
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

const MainNav = ({ children }: { children?: React.ReactNode }) => {
  const items = useMemo(
    () => getNavDropdownItems({ basePath: config.baseUrl }),
    []
  )
  return <MainNavProvider navItems={items}>{children}</MainNavProvider>
}

export const Providers = ({ children }: { children?: React.ReactNode }) => (
  <AnalyticsProvider>
    <SiteConfigProvider config={config}>
      <MobileProvider>
        <ColorModeProvider>
          <ModalProvider>
            <LearningPathProvider>
              <NotificationProvider>
                <ScrollControllerProvider scrollableSelector="#main">
                  <Sidebar>
                    <PaginationProvider>
                      <MainNav>
                        <SearchProvider
                          algolia={{ appId: "temp", apiKey: "temp", mainIndexName: "temp" }}
                          indices={[{ value: "temp", title: "Docs" }]}
                          defaultIndex="temp"
                          searchProps={{
                            isLoading: false,
                            suggestions: [],
                            checkInternalPattern: /^\/user-guide/,
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

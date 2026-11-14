import type { DocsConfig, Sidebar } from "types"
import { globalConfig } from "docs-ui"
import { generatedSidebars as userGuideSidebars } from "~/generated/user-guide-sidebar.mjs"

export const baseUrl =
  (import.meta.env.PUBLIC_BASE_URL as string | undefined) ||
  "https://medusago.hitkey.io"

const common = {
  ...globalConfig,
  baseUrl,
  logo: "/brand/medusago-symbol-light.svg",
  breadcrumbOptions: {
    startItems: [{ title: "Documentation", link: baseUrl }],
  },
  version: { ...globalConfig.version, hide: true },
}

export type SectionKey = "user-guide"

export const sections: Record<SectionKey, DocsConfig> = {
  "user-guide": {
    ...common,
    titleSuffix: "MedusaGo Admin User Guide",
    description: "Learn how to manage a MedusaGo store with the admin dashboard.",
    basePath: "/user-guide",
    sidebars: userGuideSidebars as Sidebar.Sidebar[],
    project: { title: "User Guide", key: "user-guide" },
  },
}

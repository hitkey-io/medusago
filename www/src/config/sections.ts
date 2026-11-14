import type { DocsConfig, Sidebar } from "types"
import { globalConfig } from "docs-ui"
import { generatedSidebars as userGuideSidebars } from "~/generated/user-guide-sidebar.mjs"
import { generatedSidebars as resourcesSidebars } from "~/generated/resources-sidebar.mjs"

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
  contentMenuSections: { products: true, whatsNew: false, cloudStatus: false },
}

export type SectionKey = "user-guide" | "resources"

export const sections: Record<SectionKey, DocsConfig> = {
  "user-guide": {
    ...common,
    titleSuffix: "MedusaGo Admin User Guide",
    description: "Learn how to manage a MedusaGo store with the admin dashboard.",
    basePath: "/user-guide",
    sidebars: userGuideSidebars as Sidebar.Sidebar[],
    project: { title: "User Guide", key: "user-guide" },
  },
  resources: {
    ...common,
    titleSuffix: "MedusaGo Resources",
    description: "Guides, references and integrations for building with MedusaGo.",
    basePath: "/resources",
    sidebars: resourcesSidebars as Sidebar.Sidebar[],
    project: { title: "Resources", key: "resources" },
  },
}

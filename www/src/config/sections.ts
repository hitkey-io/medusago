import type { DocsConfig, Sidebar } from "types"
import { globalConfig } from "docs-ui"
import { generatedSidebars as userGuideSidebars } from "~/generated/user-guide-sidebar.mjs"
import { generatedSidebars as resourcesSidebars } from "~/generated/resources-sidebar.mjs"

// Links are relative to the host that serves the site (local preview, staging or
// production), so no absolute URL is configured anywhere.
export const baseUrl = ""

const common = {
  ...globalConfig,
  baseUrl,
  logo: "/brand/medusago-symbol-light.svg",
  breadcrumbOptions: {
    startItems: [{ title: "Documentation", link: "/" }],
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
    basePath: "",
    sidebars: userGuideSidebars as Sidebar.Sidebar[],
    project: { title: "User Guide", key: "user-guide" },
  },
  resources: {
    ...common,
    titleSuffix: "MedusaGo Resources",
    description: "Guides, references and integrations for building with MedusaGo.",
    basePath: "",
    sidebars: resourcesSidebars as Sidebar.Sidebar[],
    project: { title: "Resources", key: "resources" },
  },
}

// The homepage has no sidebar, but the providers expect one sidebar to exist.
export const homeConfig: DocsConfig = {
  ...common,
  titleSuffix: "MedusaGo Documentation",
  description: "Documentation of MedusaGo, the Go implementation of the Medusa commerce backend.",
  basePath: "",
  sidebars: [{ sidebar_id: "home", title: "Home", items: [] }] as unknown as Sidebar.Sidebar[],
  project: { title: "Documentation", key: "docs" },
}

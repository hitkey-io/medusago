import type { DocsConfig, Sidebar } from "types"
import { globalConfig } from "docs-ui"
import { generatedSidebars as userGuideSidebars } from "~/generated/user-guide-sidebar.mjs"
import { generatedSidebars as resourcesSidebars } from "~/generated/resources-sidebar.mjs"
import { generatedSidebars as learnSidebars } from "~/generated/learn-sidebar.mjs"
import releaseVersion from "~/generated/version.json"
import { generatedSidebars as apiSidebars } from "~/generated/api-sidebar.mjs"

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
  version: releaseVersion,
  contentMenuSections: { products: true, whatsNew: false, cloudStatus: false },
}

export type SectionKey = "user-guide" | "resources" | "learn" | "api"

export const sections: Record<SectionKey, DocsConfig> = {
  "user-guide": {
    ...common,
    titleSuffix: "MedusaGo Admin User Guide",
    description: "Learn how to manage a MedusaGo store with the admin dashboard.",
    basePath: "",
    sidebars: userGuideSidebars as Sidebar.Sidebar[],
    project: { title: "User Guide", key: "user-guide" },
  },
  learn: {
    ...common,
    titleSuffix: "MedusaGo Documentation",
    description: "Learn how to install, customize and deploy MedusaGo.",
    basePath: "",
    sidebars: learnSidebars as Sidebar.Sidebar[],
    project: { title: "Learn", key: "docs" },
  },
  api: {
    ...common,
    titleSuffix: "MedusaGo API Reference",
    description: "The Admin and Store REST APIs of MedusaGo.",
    basePath: "",
    sidebars: apiSidebars as Sidebar.Sidebar[],
    project: { title: "API Reference", key: "api" },
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

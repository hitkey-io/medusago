import type { DocsConfig, Sidebar } from "types"
import { globalConfig } from "docs-ui"
import { generatedSidebars } from "~/generated/user-guide-sidebar.mjs"

export const baseUrl =
  (import.meta.env.PUBLIC_BASE_URL as string | undefined) ||
  "https://medusago.hitkey.io"

export const config: DocsConfig = {
  ...globalConfig,
  titleSuffix: "MedusaGo Admin User Guide",
  description: "Learn how to manage a MedusaGo store with the admin dashboard.",
  baseUrl,
  basePath: "",
  sidebars: generatedSidebars as Sidebar.Sidebar[],
  project: { title: "User Guide", key: "user-guide" },
  logo: "/images/logo.png",
  breadcrumbOptions: {
    startItems: [{ title: "Documentation", link: baseUrl }],
  },
  version: { ...globalConfig.version, hide: true },
}

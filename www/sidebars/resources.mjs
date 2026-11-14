import { infrastructureModulesSidebar } from "./resources/infrastructure-modules.mjs"
import { integrationsSidebar } from "./resources/integrations.mjs"
import { recipesSidebar } from "./resources/recipes.mjs"
import { referencesSidebar } from "./resources/references.mjs"
import { toolsSidebar } from "./resources/tools.mjs"
import { storefrontDevelopmentSidebar } from "./resources/storefront.mjs"
import { troubleshootingSidebar } from "./resources/troubleshooting.mjs"
import { commerceModulesSidebar } from "./resources/commerce-modules.mjs"

/** @type {import("types").Sidebar.RawSidebar[]} */
export const sidebar = [
  {
    sidebar_id: "recipes",
    title: "Recipes",
    items: recipesSidebar,
  },
  {
    sidebar_id: "integrations",
    title: "Integrations",
    items: integrationsSidebar,
  },
  {
    sidebar_id: "storefront-development",
    title: "Storefront Development",
    items: storefrontDevelopmentSidebar,
  },
  {
    sidebar_id: "tools",
    title: "Tools",
    items: toolsSidebar,
  },
  {
    sidebar_id: "references",
    title: "References",
    items: referencesSidebar,
  },
  {
    sidebar_id: "commerce-modules",
    title: "Commerce Modules",
    items: commerceModulesSidebar,
  },
  {
    sidebar_id: "infrastructure-modules",
    title: "Infrastructure Modules",
    items: infrastructureModulesSidebar,
  },
  {
    sidebar_id: "troubleshooting",
    title: "Troubleshooting",
    items: troubleshootingSidebar,
  },
]

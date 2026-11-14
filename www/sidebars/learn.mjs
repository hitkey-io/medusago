/** @type {import('types').Sidebar.RawSidebar[]} */
// The MedusaGo "Learn" sidebar. Entries whose page does not exist yet are left
// out by scripts/generate-sidebars.mjs, so pages can be added one by one.
const L = (path, title, children) => ({ type: "link", path, title, ...(children ? { children } : {}) })
const C = (title, children, extra = {}) => ({ type: "category", title, children, ...extra })
const F = "/fundamentals"

export const sidebar = [
  {
    sidebar_id: "docs",
    title: "Learn",
    items: [
      C("Get Started", [
        L("/", "Introduction"),
        L("/installation", "Installation"),
        L("/introduction/architecture", "Architecture"),
      ], { initialOpen: true }),
      C("Customization Tutorial", [
        L("/customization/custom-features", "Build Custom Features", [
          L("/customization/custom-features/module", "Brand Module"),
          L("/customization/custom-features/workflow", "Brand Workflow"),
          L("/customization/custom-features/api-route", "Brand API Route"),
        ]),
        L("/customization/extend-features", "Extend Features", [
          L("/customization/extend-features/define-link", "Link Brands and Products"),
          L("/customization/extend-features/extend-create-product", "Extend Core Flow"),
          L("/customization/extend-features/query-linked-records", "Query Linked Records"),
        ]),
        L("/customization/customize-admin", "Customize Admin", [
          L("/customization/customize-admin/widget", "Add Widget"),
          L("/customization/customize-admin/route", "Add UI Route"),
        ]),
        L("/customization/next-steps", "Next Steps"),
      ]),
      C("Framework", [
        L(`${F}/framework`, "Overview"),
        L(`${F}/medusa-container`, "Medusa Container"),
        L(`${F}/modules`, "Modules", [
          L(`${F}/modules/modules-directory-structure`, "Directory Structure"),
          L(`${F}/modules/loaders`, "Loaders"),
          L(`${F}/modules/options`, "Module Options"),
          L(`${F}/modules/service-factory`, "Service Factory"),
          L(`${F}/modules/db-operations`, "Database Operations"),
        ]),
        L(`${F}/module-links`, "Module Links", [
          L(`${F}/module-links/link`, "Link"),
          L(`${F}/module-links/custom-columns`, "Add Custom Columns"),
          L(`${F}/module-links/read-only`, "Read-Only Links"),
        ]),
        L(`${F}/query`, "Query"),
        L(`${F}/data-models`, "Data Models", [
          L(`${F}/data-models/properties`, "Properties"),
          L(`${F}/data-models/relationships`, "Relationships"),
          L(`${F}/data-models/indexes`, "Define Index"),
          L(`${F}/data-models/write-migration`, "Migrations"),
        ]),
        L(`${F}/api-routes`, "API Routes", [
          L(`${F}/api-routes/http-methods`, "HTTP Methods"),
          L(`${F}/api-routes/parameters`, "Parameters"),
          L(`${F}/api-routes/responses`, "Response"),
          L(`${F}/api-routes/middlewares`, "Middlewares"),
          L(`${F}/api-routes/validation`, "Validation"),
          L(`${F}/api-routes/protected-routes`, "Protected Routes"),
          L(`${F}/api-routes/errors`, "Errors"),
          L(`${F}/api-routes/cors`, "Handling CORS"),
          L(`${F}/api-routes/allowed-fields`, "Allowed Fields"),
        ]),
        L(`${F}/workflows`, "Workflows", [
          L(`${F}/workflows/compensation-function`, "Compensation Function"),
          L(`${F}/workflows/errors`, "Error Handling"),
          L(`${F}/workflows/workflow-hooks`, "Workflow Hooks"),
          L(`${F}/workflows/retry-failed-steps`, "Retry Failed Steps"),
          L(`${F}/workflows/parallel-steps`, "Run Steps in Parallel"),
          L(`${F}/workflows/workflow-timeout`, "Workflow Timeout"),
          L(`${F}/workflows/store-executions`, "Store Workflow Executions"),
          L(`${F}/workflows/long-running-workflow`, "Long-Running Workflow"),
          L(`${F}/workflows/execute-another-workflow`, "Nested Workflows"),
          L(`${F}/workflows/locks`, "Using Locks"),
        ]),
        L(`${F}/events-and-subscribers`, "Events and Subscribers", [
          L(`${F}/events-and-subscribers/data-payload`, "Events Data Payload"),
          L(`${F}/events-and-subscribers/emit-event`, "Emit Event"),
        ]),
        L(`${F}/scheduled-jobs`, "Scheduled Jobs"),
        L(`${F}/plugins`, "Plugins", [L(`${F}/plugins/create`, "Create Plugin")]),
        L(`${F}/custom-cli-scripts`, "Custom CLI Scripts", [L(`${F}/custom-cli-scripts/seed-data`, "Seed Data")]),
      ]),
      C("Admin Development", [
        L(`${F}/admin`, "Overview"),
        L(`${F}/admin/widgets`, "Admin Widgets"),
        L(`${F}/admin/custom-injection-zones`, "Custom Injection Zones"),
        L(`${F}/admin/ui-routes`, "Admin UI Routes"),
        L(`${F}/admin/environment-variables`, "Environment Variables"),
        L(`${F}/admin/routing`, "Routing Customizations"),
        L(`${F}/admin/translations`, "Translations"),
        L(`${F}/admin/constraints`, "Constraints"),
        L(`${F}/admin/tips`, "Tips"),
      ]),
      C("Configurations", [
        L(`${F}/environment-variables`, "Environment Variables"),
        L("/configurations/medusa-config", "MedusaGo Configurations"),
      ]),
      C("Debugging & Testing", [
        L("/debugging-and-testing/logging", "Logging"),
        L("/debugging-and-testing/feature-flags", "Feature Flags"),
      ]),
      C("Production", [
        L("/build", "Build"),
        L("/production/worker-mode", "Worker Modes"),
        L("/deployment", "Deployment Overview", [L("/deployment/general", "General Deployment")]),
      ]),
      C("Upgrade", [L("/update", "Update MedusaGo")]),
    ],
  },
]

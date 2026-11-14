// One entry per picture of the user guide. `page` + `alt` identify the upstream picture it replaces
// (images.json); the importer puts our file at the place where the upstream page had it.
//   id     file name in public/images/shots
//   route  dashboard path, or async (ctx) => path
//   run    extra steps before the picture (open a form, fill a field...)
//   el     (page, ctx) => locator: crop to this element (a section, a form) instead of the whole window
//   anonymous: the page is shown before login
const card = (heading) => (page) =>
  page.locator("div.shadow-elevation-card-rest", { has: page.getByRole("heading", { name: heading, exact: true }) })
const dialog = (page) => page.getByRole("dialog")
// the top of a form in a focus modal: from the heading down to the last field, with a margin
const formClip = async (page) => {
  const d = page.getByRole("dialog")
  const head = await d.getByRole("heading").first().boundingBox()
  const fields = d.locator("input:visible, textarea:visible")
  const last = await fields.last().boundingBox()
  const m = 48
  const x = Math.max(0, Math.min(head.x, last.x) - m)
  const y = Math.max(0, head.y - m)
  return { x, y, width: Math.max(head.width, last.width) + 2 * m, height: last.y + last.height - head.y + 2 * m }
}
const idOf = (list, key, find) => async (ctx) => {
  const r = await ctx.api(list)
  const x = r[key].find(find)
  if (!x) throw new Error(`no ${key} matching in ${list}`)
  return x.id
}
const firstId = (list, key) => idOf(list, key, () => true)

const UG = "user-guide"
export const shots = [
  // --- lists and overviews -------------------------------------------------------------------
  { id: "orders-list", page: `${UG}/orders`, alt: "Orders List", route: "/orders" },
  { id: "products-list", page: `${UG}/products`, alt: "Product List", route: "/products" },
  { id: "settings-store", page: `${UG}/settings`, alt: "Settings Page", route: "/settings/store" },
  { id: "customers-list", page: `${UG}/customers`, alt: "Customers List", route: "/customers" },
  { id: "customer-groups-list", page: `${UG}/customers/groups`, alt: "Customer Groups List", route: "/customer-groups" },
  { id: "inventory-list", page: `${UG}/inventory`, alt: "Inventory List", route: "/inventory" },
  { id: "reservations-list", page: `${UG}/inventory/reservations`, alt: "Reservations List", route: "/reservations" },
  { id: "draft-orders-list", page: `${UG}/orders/draft-orders`, alt: "Draft Orders List", route: "/draft-orders" },
  { id: "price-lists-list", page: `${UG}/price-lists`, alt: "Price Lists list", route: "/price-lists" },
  { id: "categories-list", page: `${UG}/products/categories`, alt: "Product Categories List", route: "/categories" },
  { id: "collections-list", page: `${UG}/products/collections`, alt: "Product Collections List", route: "/collections" },
  { id: "promotions-list", page: `${UG}/promotions`, alt: "Promotions list", route: "/promotions" },
  { id: "campaigns-list", page: `${UG}/promotions/campaigns`, alt: "Campaigns list", route: "/campaigns" },
  { id: "publishable-keys-list", page: `${UG}/settings/developer/publishable-api-keys`, alt: "Publishable API keys list", route: "/settings/publishable-api-keys" },
  { id: "secret-keys-list", page: `${UG}/settings/developer/secret-api-keys`, alt: "Secret API keys list", route: "/settings/secret-api-keys" },
  { id: "workflow-executions-list", page: `${UG}/settings/developer/workflows`, alt: "Workflow executions list", route: "/settings/workflows" },
  { id: "locations-shipping", page: `${UG}/settings/locations-and-shipping`, alt: "Location & shipping settings", route: "/settings/locations" },

  // --- detail pages --------------------------------------------------------------------------
  { id: "customer-group-details", page: `${UG}/customers/groups`, alt: "Customer group details page", route: async (c) => "/customer-groups/" + (await idOf("/customer-groups", "customer_groups", (g) => g.name === "VIP")(c)) },
  { id: "customer-details", page: `${UG}/customers/manage`, alt: "Customers Detail Page", route: async (c) => "/customers/" + (await idOf("/customers", "customers", (x) => x.email === "emma.larsen@example.com")(c)) },
  { id: "inventory-details", page: `${UG}/inventory/inventory`, alt: "Inventory details page", route: async (c) => "/inventory/" + (await firstId("/inventory-items", "inventory_items")(c)) },
  { id: "order-details", page: `${UG}/orders/manage`, alt: "Order Details Page", route: async (c) => "/orders/" + (await firstId("/orders?order=display_id", "orders")(c)) },
  { id: "price-list-details", page: `${UG}/price-lists/manage`, alt: "Price List details page", route: async (c) => "/price-lists/" + (await idOf("/price-lists", "price_lists", (x) => x.title === "Black Friday")(c)) },
  { id: "category-details", page: `${UG}/products/categories`, alt: "Category details page", route: async (c) => "/categories/" + (await idOf("/product-categories", "product_categories", (x) => x.name === "Pants")(c)) },
  { id: "collection-details", page: `${UG}/products/collections`, alt: "Collection details page", route: async (c) => "/collections/" + (await firstId("/collections", "collections")(c)) },
  { id: "product-details", page: `${UG}/products/edit`, alt: "Product Details Page", route: async (c) => "/products/" + (await idOf("/products", "products", (x) => x.title === "Medusa T-Shirt")(c)) },
  { id: "campaign-details", page: `${UG}/promotions/campaigns`, alt: "Campaign Details", route: async (c) => "/campaigns/" + (await firstId("/campaigns", "campaigns")(c)) },
  { id: "promotion-details", page: `${UG}/promotions/manage`, alt: "Promotion Details", route: async (c) => "/promotions/" + (await idOf("/promotions", "promotions", (x) => x.code === "SUMMER10")(c)) },

  // --- sections of the order page ------------------------------------------------------------
  ...[["order-summary", "Order Summary", "Summary"], ["order-customer", "Customer Section", "Customer"], ["order-activity", "Activity Section", "Activity"],
    ["order-payments", "Payments Section", "Payments"], ["order-unfulfilled", "Unfulfilled Items Section", "Unfulfilled Items"]].map(([id, alt, heading]) => ({
    id, page: `${UG}/orders/manage`, alt,
    route: async (c) => "/orders/" + (await idOf("/orders", "orders", (o) => o.display_id === 4)(c)), el: card(heading),
  })),

  // --- forms ----------------------------------------------------------------------------------
  { id: "create-customer-group-form", page: `${UG}/customers/groups`, alt: "Create customer group form", route: "/customer-groups/create", clip: formClip },
  { id: "create-customer-form", page: `${UG}/customers/manage`, alt: "Create Customer Form", route: "/customers/create", clip: formClip },

  // --- before login ------------------------------------------------------------------------------
  { id: "login-page", page: `${UG}/index`, alt: "Log In Page", route: "/login", anonymous: true },
  { id: "reset-password-request", page: `${UG}/reset-password`, alt: "Request reset password form", route: "/reset-password", anonymous: true },
]

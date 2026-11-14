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

// a modal: a drawer as it is (down to its last content), a full-screen focus modal cropped to its content
const modalClip = async (page) => {
  const d = page.getByRole("dialog").last()
  const box = await d.boundingBox()
  const cancel = await d.getByRole("button", { name: /^(Cancel|Close)$/ }).last().boundingBox().catch(() => null)
  const focus = box.width > 1000
  const r = await d.evaluate((el, { focus, footerY, boxY }) => {
    const W = el.getBoundingClientRect().width
    let top = 1e9, left = 1e9, right = 0, bottom = 0
    for (const n of el.querySelectorAll("*")) {
      const b = n.getBoundingClientRect()
      if (!b.width || !b.height) continue
      if (focus && b.top < boxY + 60) continue // the header with the close button
      if (footerY && b.top >= footerY - 8) continue // the footer with Cancel/Save
      // only what the user sees: text, fields, buttons (not empty layout boxes)
      const control = n.matches("input,textarea,button,[role=combobox],[role=switch],svg")
      if (!control && (n.children.length || !n.textContent.trim())) continue
      top = Math.min(top, b.top); left = Math.min(left, b.left); right = Math.max(right, b.right); bottom = Math.max(bottom, b.bottom)
    }
    return { top, left, right, bottom }
  }, { focus, footerY: cancel?.y ?? null, boxY: box.y })
  if (!focus) {
    // a drawer: short forms are cut below their last field, long ones (they scroll) are shown whole
    const fields = d.locator("input:visible, textarea:visible, [role=combobox]:visible")
    const n = await fields.count()
    let lastBottom = 0
    for (let i = 0; i < n; i++) {
      const bb = await fields.nth(i).boundingBox()
      if (bb) lastBottom = Math.max(lastBottom, bb.y + bb.height)
    }
    const short = lastBottom < box.y + box.height * 0.5
    return { x: box.x, y: box.y, width: box.width, height: short ? lastBottom - box.y + 40 : box.height }
  }
  const m = 48
  return { x: r.left - m, y: r.top - m, width: r.right - r.left + 2 * m, height: r.bottom - r.top + 2 * m }
}
const ordId = (n) => idOf("/orders?limit=50", "orders", (o) => o.display_id === n)
const orderRoute = (n, tail = "") => async (c) => `/orders/${await ordId(n)(c)}${tail}`
const draftId = firstId("/draft-orders", "draft_orders")
const draftRoute = (tail = "") => async (c) => `/draft-orders/${await draftId(c)}${tail}`
const fulfillmentOf = (n) => async (c) => {
  const o = await ordId(n)(c)
  const r = await c.api(`/orders/${o}?fields=+fulfillments.id`)
  return { order: o, fulfillment: r.order.fulfillments[0].id }
}

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

  // =========================== Orders ===========================================================
  // sections of orders in a given state (seed: #7 return, #12 claim, #10 exchange, #9 order edit)
  { id: "return-activity", page: `${UG}/orders/returns`, alt: "Return requested in Activity section", route: orderRoute(7), el: card("Activity") },
  { id: "return-summary-items", page: `${UG}/orders/returns`, alt: "Items to be returned in Summary section", route: orderRoute(7), el: card("Summary") },
  { id: "return-outstanding", page: `${UG}/orders/returns`, alt: "Outstanding amount in summary", route: orderRoute(7), el: card("Summary") },
  { id: "claim-activity", page: `${UG}/orders/claims`, alt: "Claim in Activity section", route: orderRoute(12), el: card("Activity") },
  { id: "exchange-activity", page: `${UG}/orders/exchanges`, alt: "Exchange requested in Activity section", route: orderRoute(10), el: card("Activity") },
  { id: "order-edit-requested", page: `${UG}/orders/edit`, alt: "Order edit requested section", route: orderRoute(9), el: (p) => p.getByText("Order Edit", { exact: false }).first().locator("xpath=ancestor::div[contains(@class,'shadow-elevation-card-rest')][1]") },
  { id: "order-fulfillment-section", page: `${UG}/orders/manage`, alt: "Fulfillment Section", route: orderRoute(3), el: (p) => p.locator("div.shadow-elevation-card-rest", { has: p.getByRole("heading", { name: /^Fulfillment #/ }) }) },
  { id: "order-capture-payment", page: `${UG}/orders/payments`, alt: "Capture Payment Button", route: orderRoute(6), el: card("Payments") },
  { id: "order-outstanding-summary", page: `${UG}/orders/payments`, alt: "Outstanding amount in Summary", route: orderRoute(5), el: card("Summary") },

  // forms (routes of the dashboard's drawers and modals)
  { id: "form-fulfillment", page: `${UG}/orders/fulfillments`, alt: "Fulfillment Form", route: orderRoute(4, "/fulfillment?requires_shipping=true"), clip: async (page) => {
    // this form has no heading: from the Location field down to the notification switch
    const d = page.getByRole("dialog").last()
    const top = await d.getByText("Location", { exact: true }).first().boundingBox()
    const bottom = await d.getByText("Notify customers about the created fulfillment.").boundingBox()
    const m = 48
    return { x: top.x - m, y: top.y - m, width: 700 + 2 * m, height: bottom.y + bottom.height - top.y + 2 * m }
  } },
  { id: "form-shipment", page: `${UG}/orders/fulfillments`, alt: "Shipment Form", route: async (c) => { const f = await fulfillmentOf(3)(c); return `/orders/${f.order}/${f.fulfillment}/create-shipment` }, clip: modalClip },
  { id: "form-edit-shipping-address", page: `${UG}/orders/manage`, alt: "Edit Shipping Address Form", route: orderRoute(4, "/shipping-address"), clip: modalClip },
  { id: "form-edit-billing-address", page: `${UG}/orders/manage`, alt: "Edit Billing Address Form", route: orderRoute(4, "/billing-address"), clip: modalClip },
  { id: "form-edit-email", page: `${UG}/orders/manage`, alt: "Edit Email Address Form", route: orderRoute(4, "/email"), clip: modalClip },
  { id: "form-transfer-ownership", page: `${UG}/orders/manage`, alt: "Transfer Ownership Form", route: orderRoute(4, "/transfer"), clip: modalClip },
  { id: "form-refund", page: `${UG}/orders/payments`, alt: "Refund Payment Form", route: orderRoute(1, "/refund"), clip: modalClip },
  { id: "form-create-return", page: `${UG}/orders/returns`, alt: "Create Return Form", route: orderRoute(2, "/returns"), clip: modalClip },
  { id: "form-create-claim", page: `${UG}/orders/claims`, alt: "Create Claim Form", route: orderRoute(1, "/claims"), clip: modalClip },
  { id: "form-create-exchange", page: `${UG}/orders/exchanges`, alt: "Create Exchange Form", route: orderRoute(5, "/exchanges"), clip: modalClip },
  { id: "form-order-edit", page: `${UG}/orders/edit`, alt: "Order Edit Form", route: orderRoute(6, "/edits"), clip: modalClip },

  // draft orders
  { id: "draft-form", page: `${UG}/orders/draft-orders/create`, alt: "Draft order form", route: "/draft-orders/create", clip: modalClip },
  ...[["draft-summary", "Order Summary", "Summary"], ["draft-customer", "Customer Section", "Customer"], ["draft-activity", "Activity Section", "Activity"], ["draft-shipping", "Shipping Section", "Shipping"]].map(([id, alt, heading]) => ({
    id, page: `${UG}/orders/draft-orders/manage`, alt, route: draftRoute(), el: card(heading),
  })),
  { id: "draft-form-sales-channel", page: `${UG}/orders/draft-orders/manage`, alt: "Edit Sales Channel form", route: draftRoute("/sales-channel"), clip: modalClip },
  { id: "draft-form-items", page: `${UG}/orders/draft-orders/manage`, alt: "Draft Order Items Form", route: draftRoute("/items"), clip: async (page) => {
    const box = await page.getByRole("dialog").last().boundingBox()
    return { x: box.x + box.width * 0.2, y: box.y + 70, width: box.width * 0.6, height: Math.min(560, box.height - 170) }
  } },
  { id: "draft-form-promotions", page: `${UG}/orders/draft-orders/manage`, alt: "Draft Order Promotions Form", route: draftRoute("/promotions"), clip: modalClip },
  { id: "draft-form-shipping", page: `${UG}/orders/draft-orders/manage`, alt: "Draft Order Shipping Form", route: draftRoute("/shipping"), clip: async (page) => {
    const d = page.getByRole("dialog").last()
    const head = await d.getByRole("heading").first().boundingBox()
    const btn = await d.getByRole("button", { name: "Add shipping option" }).last().boundingBox()
    const m = 48
    return { x: head.x - m, y: head.y - m, width: 1210 / 2 + 2 * m + 120, height: btn.y + btn.height - head.y + 2 * m }
  } },
  { id: "draft-form-transfer", page: `${UG}/orders/draft-orders/manage`, alt: "Transfer ownership form for draft order", route: draftRoute("/transfer-ownership"), clip: modalClip },
  { id: "draft-form-email", page: `${UG}/orders/draft-orders/manage`, alt: "Edit email in draft order form", route: draftRoute("/email"), clip: modalClip },
  { id: "draft-form-shipping-address", page: `${UG}/orders/draft-orders/manage`, alt: "Edit shipping address in draft order form", route: draftRoute("/shipping-address"), clip: modalClip },
]

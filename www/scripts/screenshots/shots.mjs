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
    // a drawer: a short form is cut below its content, a long one (it scrolls) is shown whole
    const short = r.bottom < box.y + box.height * 0.6
    return { x: box.x, y: box.y, width: box.width, height: short ? r.bottom - box.y + 40 : box.height }
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

// an order whose items have no reservations (to show "Not allocated" and the allocation form)
const deallocated = (n) => async (c) => {
  const id = await ordId(n)(c)
  const o = (await c.api(`/orders/${id}?fields=*items`)).order
  const rs = (await c.api("/reservations?limit=100")).reservations.filter((r) => o.items.some((i) => i.id === r.line_item_id))
  for (const r of rs) await c.send("DELETE", `/reservations/${r.id}`)
  return id
}
const returnOf = (n) => async (c) => {
  const o = await ordId(n)(c)
  return { order: o, ret: (await c.api(`/returns?order_id=${o}`)).returns[0].id }
}
// a confirmation pop-up: the dialog with some air around it
const promptClip = async (page) => {
  const b = await page.getByRole("alertdialog").boundingBox()
  const m = 48
  return { x: b.x - m, y: b.y - m, width: b.width + 2 * m, height: b.height + 2 * m }
}
const openFulfillmentAction = (label) => async (page) => {
  await page.getByRole("button", { name: label }).first().click()
  await page.getByRole("alertdialog").waitFor()
}

const cardRe = (re) => (page) =>
  page.locator("div.shadow-elevation-card-rest", { has: page.getByRole("heading", { name: re }) })
const tshirt = idOf("/products", "products", (x) => x.title === "Medusa T-Shirt")
const tshirtRoute = (tail = "") => async (c) => `/products/${await tshirt(c)}${tail}`
const variantRoute = (tail = "") => async (c) => {
  const id = await tshirt(c)
  const p = await c.api(`/products/${id}?fields=+variants.id`)
  return `/products/${id}/variants/${p.product.variants[0].id}${tail}`
}
const categoryWithProducts = async (c) => {
  const r = await c.api("/product-categories?fields=+products.id&limit=50")
  return r.product_categories.find((x) => x.products?.length).id
}

// the "Create product" wizard, driven up to a stage: 1 details, 2 organize, 3 variants, 4 variants with an inventory kit, 5 inventory kits.
// (nothing is published: the picture is taken before the last button)
const createProductFlow = (stage) => async (page) => {
  const d = page.locator("[role=dialog]").last()
  const next = async () => { await d.locator("button:has-text('Continue')").click(); await page.waitForTimeout(800) }
  await d.locator("input[name=title]").fill("Medusa Hoodie")
  await d.locator("input[name=subtitle]").fill("Warm and cosy")
  await d.locator("textarea[name=description]").fill("A warm and cozy hoodie")
  if (stage === 1) return
  await d.getByText("Yes, this is a product with variants").click()
  await d.getByText("Product options").scrollIntoViewIfNeeded()
  await d.locator("[role=combobox]").first().click()
  await page.keyboard.type("Size"); await page.waitForTimeout(400); await page.keyboard.press("Enter")
  await d.locator('input[placeholder="Red, Blue, Green"]').click()
  for (const v of ["S", "M", "L"]) { await page.keyboard.type(v); await page.keyboard.press("Enter") }
  await next()
  if (stage === 2) return
  await next()
  for (const [i, sku] of ["HOODIE-S", "HOODIE-M", "HOODIE-L"].entries()) {
    await page.mouse.click(383, 183 + i * 41); await page.keyboard.type(sku); await page.keyboard.press("Enter")
    await page.mouse.click(985, 183 + i * 41); await page.keyboard.type("45"); await page.keyboard.press("Enter")
    await page.mouse.click(533, 183 + i * 41)
  }
  if (stage === 3) { await page.mouse.click(1000, 500); return }
  await page.mouse.click(833, 183)
  await page.waitForTimeout(500)
  if (stage === 4) return
  await next()
  await d.locator("[role=combobox]").first().click()
  await page.locator("[role=option]").first().click()
  await d.locator('input[placeholder^="How many"]').first().fill("1")
}

const emma = idOf("/customers", "customers", (x) => x.email === "emma.larsen@example.com")
const vipGroup = idOf("/customer-groups", "customer_groups", (g) => g.name === "VIP")
const firstItem = firstId("/inventory-items", "inventory_items")
const firstReservation = firstId("/reservations", "reservations")
const promo = (code) => idOf("/promotions", "promotions", (x) => x.code === code)
const campaign = firstId("/campaigns", "campaigns")
const blackFriday = idOf("/price-lists", "price_lists", (x) => x.title === "Black Friday")
// tick the first row of a table inside a card (the command bar with "Remove" shows up)
const tickFirstRow = (heading) => async (page) => {
  await card(heading)(page).getByRole("checkbox").nth(1).click()
  await page.waitForTimeout(500)
}

// "Create promotion": the details step of a promotion type (radio cards: "Amount off products", "Free shipping"...), with one condition added in a section
const promotionSection = (type, heading, attribute, comboIndex = 0) => ({
  route: "/promotions/create",
  run: async (page) => {
    const d = page.locator("[role=dialog]").last()
    await d.getByText(type, { exact: true }).first().click()
    await d.locator("button:has-text('Continue')").click()
    await page.waitForTimeout(900)
    const section = d.locator("div").filter({ has: page.locator(`text="${heading}"`) }).filter({ has: page.locator("button:has-text('Add condition')") }).last()
    await section.scrollIntoViewIfNeeded()
    await section.locator("button:has-text('Add condition')").first().click()
    await page.waitForTimeout(500)
    if (attribute) {
      await section.locator("[role=combobox]").nth(comboIndex).click()
      await page.locator("[role=option]", { hasText: attribute }).first().click()
      await page.waitForTimeout(400)
    }
  },
  el: (page) => page.locator("[role=dialog]").last().locator("div").filter({ has: page.locator(`text="${heading}"`) }).filter({ has: page.locator("button:has-text('Add condition')") }).last(),
})

// "Create shipping option": step 1 filled in, step 2 (prices), or the conditional prices dialog of a price cell
const shippingOptionFlow = (stage) => async (page) => {
  const d = page.locator("[role=dialog]").last()
  await d.locator("input[name=name]").fill("Same-day Delivery")
  const combos = d.locator("[role=combobox]")
  for (const i of [0, 1, 2, 3]) { await combos.nth(i).click(); await page.waitForTimeout(300); await page.locator("[role=option]:visible").first().click(); await page.waitForTimeout(400) }
  if (stage === 1) return
  await d.locator("button:has-text('Continue')").click()
  await page.waitForTimeout(1000)
  if (stage === 2) { await page.mouse.click(233, 183); await page.keyboard.type("15"); await page.keyboard.press("Enter"); return }
  await page.mouse.click(233, 183)
  await page.mouse.click(203, 183)
  await page.waitForTimeout(700)
}

// "Create price list": details filled in, then the products step (two products ticked), then the prices step
const priceListFlow = (stage) => async (page) => {
  const d = page.locator("[role=dialog]").last()
  await d.locator("input[name=title]").fill("Summer Sale")
  await d.locator("textarea[name=description]").fill("Summer prices for selected products")
  await d.locator("button:has-text('Continue')").click()
  await page.waitForTimeout(1200)
  await page.mouse.click(43, 195); await page.mouse.click(43, 243)
  await page.waitForTimeout(400)
  if (stage === 2) return
  await d.locator("button:has-text('Continue')").click()
  await page.waitForTimeout(1200)
}

const UG = "user-guide"
// --- settings -----------------------------------------------------------------------------------------
const S = "/settings"
const GS = `${UG}/settings`
const keyOf = (type) => firstIdBy(`/api-keys?type=${type}`, "api_keys", (k) => k.type === type)
const firstIdBy = (list, key, find) => idOf(list, key, find)
const locationId = firstId("/stock-locations", "stock_locations")
// the first location with its first delivery service zone and one of its shipping options
const zoneInfo = async (c) => {
  const r = await c.api("/stock-locations?fields=id,*fulfillment_sets,*fulfillment_sets.service_zones,*fulfillment_sets.service_zones.shipping_options")
  const loc = r.stock_locations[0]
  const set = loc.fulfillment_sets.find((f) => f.type === "shipping")
  const zone = set.service_zones[0]
  return { loc: loc.id, set: set.id, zone: zone.id, option: zone.shipping_options?.[0]?.id }
}
const zoneRoute = (tail) => async (c) => { const z = await zoneInfo(c); return `${S}/locations/${z.loc}/fulfillment-set/${z.set}/service-zone/${z.zone}${tail}` }
const byName = (list, key, name, field = "name") => idOf(list, key, (x) => x[field] === name)
const firstOf = (list, key) => firstId(list, key)
const taxRegionOf = (cc, province) => idOf("/tax-regions?limit=50&fields=id,country_code,province_code,parent_id", "tax_regions", (r) => r.country_code === cc && (province ? r.province_code === province : !r.parent_id))
const regionId = firstId("/regions", "regions")
const channelId = idOf("/sales-channels", "sales_channels", (x) => x.name === "Webshop")
const workflowId = async (c) => (await c.api("/workflows-executions?limit=1")).workflow_executions[0].id

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

  // the rest of the order page: pop-ups, allocation, receiving a return
  { id: "order-not-allocated", page: `${UG}/orders/manage`, alt: "Not Allocated Badge", route: async (c) => `/orders/${await deallocated(11)(c)}`, el: card("Summary") },
  { id: "form-allocate-items", page: `${UG}/orders/manage`, alt: "Allocate Items Form", route: async (c) => `/orders/${await deallocated(11)(c)}/allocate-items`, wait: 1500, run: async (page) => {
    await page.getByRole("dialog").last().getByRole("combobox").first().click()
    await page.getByRole("option", { name: /European Warehouse/ }).click()
    await page.waitForTimeout(1200)
  }, clip: modalClip },
  { id: "order-cancel-dropdown", page: `${UG}/orders/manage`, alt: "Cancel Order Dropdown", route: orderRoute(11), run: async (page) => {
    await page.locator("main div.shadow-elevation-card-rest", { hasText: / from (Webshop|B2B Portal)/ }).first().locator("button[aria-haspopup=menu]").click()
    await page.getByRole("menuitem", { name: /Cancel/ }).waitFor()
  }, clip: async (page) => {
    const head = await page.locator("main div.shadow-elevation-card-rest", { hasText: / from (Webshop|B2B Portal)/ }).first().boundingBox()
    const menu = await page.getByRole("menu").boundingBox()
    const right = Math.max(head.x + head.width, menu.x + menu.width) + 16
    return { x: head.x, y: head.y, width: right - head.x, height: menu.y + menu.height - head.y + 24 }
  } },
  { id: "popup-mark-delivered", page: `${UG}/orders/fulfillments`, alt: "Mark as Delivered Pop-up", route: orderRoute(2), run: openFulfillmentAction("Mark as delivered"), clip: promptClip },
  { id: "popup-mark-picked-up", page: `${UG}/orders/fulfillments`, alt: "Mark as Picked Up Pop-up", route: orderRoute(14), run: openFulfillmentAction("Mark as picked up"), clip: promptClip },
  { id: "popup-cancel-fulfillment", page: `${UG}/orders/fulfillments`, alt: "Cancel Fulfillment Pop-up", route: orderRoute(3), run: async (page) => {
    await page.locator("div.shadow-elevation-card-rest", { has: page.getByRole("heading", { name: /^Fulfillment #/ }) }).first().getByRole("button").filter({ has: page.locator("svg") }).first().click()
    await page.getByRole("menuitem", { name: /Cancel/ }).click()
    await page.getByRole("alertdialog").waitFor()
  }, clip: promptClip },
  { id: "form-mark-received", page: `${UG}/orders/returns`, alt: "Mark Received Form", route: async (c) => { const r = await returnOf(7)(c); return `/orders/${r.order}/returns/${r.ret}/receive` }, clip: modalClip },

  // =========================== Products =========================================================
  { id: "form-create-category", page: `${UG}/products/categories`, alt: "Create Category Form", route: "/categories/create", clip: modalClip },
  { id: "form-edit-category", page: `${UG}/products/categories`, alt: "Edit category details", route: async (c) => `/categories/${await categoryWithProducts(c)}/edit`, clip: modalClip },
  { id: "form-category-add-products", page: `${UG}/products/categories`, alt: "Add products to category form", route: async (c) => `/categories/${await categoryWithProducts(c)}/products` },
  { id: "form-category-ranking", page: `${UG}/products/categories`, alt: "Edit categories ranking form", route: "/categories/organize" },
  { id: "form-create-collection", page: `${UG}/products/collections`, alt: "Create Collection Form", route: "/collections/create", clip: modalClip },
  { id: "form-edit-collection", page: `${UG}/products/collections`, alt: "Edit collection form", route: async (c) => `/collections/${await firstId("/collections", "collections")(c)}/edit`, clip: modalClip },
  { id: "form-collection-add-products", page: `${UG}/products/collections`, alt: "Add products to collection form", route: async (c) => `/collections/${await firstId("/collections", "collections")(c)}/products` },
  { id: "form-import-products", page: `${UG}/products/import`, alt: "Import Products Form", route: "/products/import", clip: async (page) => {
    // the drawer, down to the template file card
    return page.evaluate(() => {
      const d = [...document.querySelectorAll("[role=dialog]")].pop().getBoundingClientRect()
      const t = [...document.querySelectorAll("*")].filter((e) => !e.children.length && /product-import-template/.test(e.textContent)).pop().getBoundingClientRect()
      return { x: d.x, y: d.y, width: d.width, height: t.bottom - d.y + 60 }
    })
  } },
  { id: "product-options-list", page: `${UG}/products/options`, alt: "Product Options List in the Medusa Admin dashboard", route: "/product-options" },
  { id: "form-create-option", page: `${UG}/products/options`, alt: "Create Product Option form in the Medusa Admin dashboard", route: "/product-options/create", clip: modalClip },
  { id: "form-edit-option", page: `${UG}/products/options`, alt: "Edit Product Option form in the Medusa Admin dashboard", route: async (c) => `/product-options/${await firstId("/product-options", "product_options")(c)}/edit`, clip: modalClip },
  { id: "form-manage-product-options", page: `${UG}/products/edit`, alt: "Manage Product Options form in the Medusa Admin dashboard", route: tshirtRoute("/options/manage"), clip: modalClip },
  { id: "product-section-details", page: `${UG}/products/edit`, alt: "Product Details Section", route: tshirtRoute(), el: (p) => p.locator("main div.shadow-elevation-card-rest").first() },
  ...[["product-section-media", "Media Section", /^Media$/], ["product-section-options", "Options Section", /^Options$/], ["product-section-variants", "Variants Section", /^Variants$/],
    ["product-section-channels", "Sales Channels Section", /^Sales Channels$/], ["product-section-shipping", "Shipping Configuration Section", /^Shipping/],
    ["product-section-organize", "Organize Section", /^Organize$/], ["product-section-attributes", "Attributes Section", /^Attributes$/]].map(([id, alt, re]) => ({
    id, page: `${UG}/products/edit`, alt, route: tshirtRoute(), el: cardRe(re),
  })),
  { id: "variant-details", page: `${UG}/products/variants`, alt: "Product Variant Details", route: variantRoute() },
  { id: "variant-inventory-editor", page: `${UG}/products/variants`, alt: "Product Variant Inventory Bulk Editor", route: tshirtRoute("/stock") },

  { id: "category-remove-products", page: `${UG}/products/categories`, alt: "Remove products from category", route: async (c) => `/categories/${await categoryWithProducts(c)}`, run: async (page) => {
    await card("Products")(page).getByRole("checkbox").nth(1).click()
    await page.waitForTimeout(500)
  } },
  { id: "create-product-1", page: `${UG}/products/create`, alt: "Create Product First Step", route: "/products/create", run: createProductFlow(1) },
  { id: "create-product-2", page: `${UG}/products/create`, alt: "Create Product Second Step", route: "/products/create", run: createProductFlow(2) },
  { id: "create-product-3", page: `${UG}/products/create`, alt: "Create Product Third Step", route: "/products/create", run: createProductFlow(3) },
  { id: "create-product-4", page: `${UG}/products/create`, alt: "Create Product Fourth Step", route: "/products/create", run: createProductFlow(5) },
  { id: "create-product-kit-enabled", page: `${UG}/products/create/bundle`, alt: "Enable has inventory kit", route: "/products/create", run: createProductFlow(4) },
  // the same pictures on the pages of bundled and multi-part products
    { file: "create-product-4", page: `${UG}/products/create/bundle`, alt: "Create Product Fourth Step" },
  { file: "create-product-kit-enabled", page: `${UG}/products/create/multi-part`, alt: "Enable has inventory kit" },
  { file: "create-product-4", page: `${UG}/products/create/multi-part`, alt: "Create Product Fourth Step" },

  // =========================== Customers ========================================================
  { id: "form-edit-customer-group", page: `${UG}/customers/groups`, alt: "Edit Customer Group", route: async (c) => `/customer-groups/${await vipGroup(c)}/edit`, clip: modalClip },
  { id: "form-group-add-customers", page: `${UG}/customers/groups`, alt: "Add Customers to Group", route: async (c) => `/customer-groups/${await vipGroup(c)}/add-customers` },
  { id: "group-remove-customers", page: `${UG}/customers/groups`, alt: "Remove customers from group", route: async (c) => `/customer-groups/${await vipGroup(c)}`, run: tickFirstRow("Customers") },
  { id: "form-edit-customer", page: `${UG}/customers/manage`, alt: "Edit Customer Form", route: async (c) => `/customers/${await emma(c)}/edit`, clip: modalClip },
  { id: "form-customer-transfer", page: `${UG}/customers/manage`, alt: "Transfer ownership form", route: async (c) => {
    const id = await emma(c)
    const o = (await c.api(`/orders?customer_id=${id}&limit=1`)).orders[0]
    return `/customers/${id}/${o.id}/transfer`
  }, clip: modalClip },
  { id: "form-customer-add-groups", page: `${UG}/customers/manage`, alt: "Add groups form", route: async (c) => `/customers/${await emma(c)}/add-customer-groups` },
  { id: "customer-remove-groups", page: `${UG}/customers/manage`, alt: "Remove customer from groups", route: async (c) => `/customers/${await emma(c)}`, run: tickFirstRow("Customer Groups") },
  { id: "form-customer-add-address", page: `${UG}/customers/manage`, alt: "Add address form", route: async (c) => `/customers/${await emma(c)}/create-address`, clip: modalClip },

  // =========================== Inventory ========================================================
  { id: "form-create-inventory-item", page: `${UG}/inventory/inventory`, alt: "Create Inventory Item Form", route: "/inventory/create" },
  { id: "form-edit-inventory-item", page: `${UG}/inventory/inventory`, alt: "Edit Inventory Item Form", route: async (c) => `/inventory/${await firstItem(c)}/edit`, clip: modalClip },
  { id: "form-inventory-attributes", page: `${UG}/inventory/inventory`, alt: "Edit Inventory Item Attributes Form", route: async (c) => `/inventory/${await firstItem(c)}/attributes`, clip: modalClip },
  { id: "form-inventory-locations", page: `${UG}/inventory/inventory`, alt: "Manage Locations Form", route: async (c) => `/inventory/${await firstItem(c)}/locations`, clip: modalClip },
  { id: "form-inventory-quantity", page: `${UG}/inventory/inventory`, alt: "Manage Quantity Form", route: async (c) => `/inventory/${await firstItem(c)}/locations/${await firstId("/stock-locations", "stock_locations")(c)}`, clip: modalClip },
  { id: "form-create-reservation", page: `${UG}/inventory/reservations`, alt: "Create Reservation form", route: "/reservations/create", clip: modalClip },
  { id: "form-edit-reservation", page: `${UG}/inventory/reservations`, alt: "Edit Reservation form", route: async (c) => `/reservations/${await firstReservation(c)}/edit`, clip: modalClip },
  { id: "reservation-details", page: `${UG}/inventory/reservations`, alt: "Reservation details page", route: async (c) => `/reservations/${await firstReservation(c)}` },
  { file: "form-create-reservation", page: `${UG}/inventory/inventory`, alt: "Create Reservation form" },
  { file: "form-edit-reservation", page: `${UG}/inventory/inventory`, alt: "Edit Reservation form" },

  // =========================== Price lists ======================================================
  { id: "price-list-create-details", page: `${UG}/price-lists/create`, alt: "Create Price List Details Form", route: "/price-lists/create" },
  { id: "price-list-create-products", page: `${UG}/price-lists/create`, alt: "Create Price List Products Form", route: "/price-lists/create", run: priceListFlow(2) },
  { id: "price-list-create-prices", page: `${UG}/price-lists/create`, alt: "Create Price List Pricing Form", route: "/price-lists/create", run: priceListFlow(3) },
  { id: "form-edit-price-list", page: `${UG}/price-lists/manage`, alt: "Edit price list form", route: async (c) => `/price-lists/${await blackFriday(c)}/edit`, clip: modalClip },
  { id: "form-price-list-configuration", page: `${UG}/price-lists/manage`, alt: "Edit price list configurations form", route: async (c) => `/price-lists/${await blackFriday(c)}/configuration`, clip: modalClip },
  { id: "price-list-remove-products", page: `${UG}/price-lists/manage`, alt: "Remove products from price list", route: async (c) => `/price-lists/${await blackFriday(c)}`, run: tickFirstRow("Products") },
  { id: "form-price-list-prices", page: `${UG}/price-lists/manage`, alt: "Edit prices in price list", route: async (c) => `/price-lists/${await blackFriday(c)}/products/edit` },

  // =========================== Promotions =======================================================
  { id: "form-create-campaign", page: `${UG}/promotions/campaigns`, alt: "Create Campaign Form", route: "/campaigns/create" },
  { id: "form-edit-campaign", page: `${UG}/promotions/campaigns`, alt: "Edit Campaign Details", route: async (c) => `/campaigns/${await campaign(c)}/edit`, clip: modalClip },
  { id: "form-campaign-budget", page: `${UG}/promotions/campaigns`, alt: "Edit Campaign Budget Limit", route: async (c) => `/campaigns/${await campaign(c)}/edit-budget`, clip: modalClip },
  { id: "form-campaign-add-promotions", page: `${UG}/promotions/campaigns`, alt: "Add Promotions to Campaign", route: async (c) => `/campaigns/${await campaign(c)}/add-promotions` },
  { id: "campaign-remove-promotions", page: `${UG}/promotions/campaigns`, alt: "Remove Promotions from Campaign", route: async (c) => `/campaigns/${await campaign(c)}`, run: tickFirstRow("Promotions") },
  { id: "form-edit-promotion", page: `${UG}/promotions/manage`, alt: "Edit Promotion Form", route: async (c) => `/promotions/${await promo("SUMMER10")(c)}/edit`, clip: modalClip },
  { id: "form-promotion-who", page: `${UG}/promotions/manage`, alt: "Edit who can use the promotion form", route: async (c) => `/promotions/${await promo("SUMMER10")(c)}/rules/edit`, clip: modalClip },
  { id: "form-promotion-items", page: `${UG}/promotions/manage`, alt: "Edit the items the promotion applies to form", route: async (c) => `/promotions/${await promo("SUMMER10")(c)}/target-rules/edit`, clip: modalClip },
  { id: "form-promotion-buy", page: `${UG}/promotions/manage`, alt: "Edit Buy X Promotion Condition form", route: async (c) => `/promotions/${await promo("BUY2GET1")(c)}/buy-rules/edit`, clip: modalClip },
  { id: "form-promotion-shipping", page: `${UG}/promotions/manage`, alt: "Edit shipping methods condition form", route: async (c) => `/promotions/${await promo("FREESHIP")(c)}/target-rules/edit`, clip: modalClip },
  { id: "form-promotion-add-to-campaign", page: `${UG}/promotions/manage`, alt: "Add Promotion to Campaign form", route: async (c) => `/promotions/${await promo("FREESHIP")(c)}/add-to-campaign`, clip: modalClip },
  { id: "promotion-create-1", page: `${UG}/promotions/create`, alt: "Create Promotion Form First Step", route: "/promotions/create" },

  { id: "promotion-section-who", page: `${UG}/promotions/create`, alt: "Who can use this section", ...promotionSection("Amount off products", "Who can use this code?", "Customer Group", 1) },
  { id: "promotion-section-items", page: `${UG}/promotions/create`, alt: "What items will the promotion be applied to section", ...promotionSection("Amount off products", "What items will the promotion be applied to?", "Product") },
  { id: "promotion-section-shipping", page: `${UG}/promotions/create`, alt: "What shipping methods will the promotion be applied to section", ...promotionSection("Free shipping", "What shipping methods will the promotion be applied to?", "Shipping") },
  { file: "promotion-section-who", page: `${UG}/promotions/create`, alt: "Who can use this code form" },
  { file: "promotion-section-who", page: `${UG}/promotions/create`, alt: "Who can use this code section" },


  // =========================== Settings =========================================================
  // publishable and secret API keys
  { id: "form-create-publishable-key", page: `${GS}/developer/publishable-api-keys`, alt: "Create publishable API key form", route: `${S}/publishable-api-keys/create`, clip: modalClip },
  { id: "publishable-key-details", page: `${GS}/developer/publishable-api-keys`, alt: "Publishable API key details page", route: async (c) => `${S}/publishable-api-keys/${await keyOf("publishable")(c)}` },
  { id: "publishable-key-copy", page: `${GS}/developer/publishable-api-keys`, alt: "Copy publishable API key", route: async (c) => `${S}/publishable-api-keys/${await keyOf("publishable")(c)}`, run: async (page) => { await page.getByText(/^pk_/).first().hover(); await page.waitForTimeout(400) }, el: (p) => p.locator("main div.shadow-elevation-card-rest").first() },
  { id: "form-edit-publishable-key", page: `${GS}/developer/publishable-api-keys`, alt: "Edit publishable API key form", route: async (c) => `${S}/publishable-api-keys/${await keyOf("publishable")(c)}/edit`, clip: modalClip },
  { id: "form-key-add-channels", page: `${GS}/developer/publishable-api-keys`, alt: "Add sales channels form", route: async (c) => `${S}/publishable-api-keys/${await keyOf("publishable")(c)}/sales-channels` },
  { id: "key-remove-channels", page: `${GS}/developer/publishable-api-keys`, alt: "Remove sales channels from publishable API key", route: async (c) => `${S}/publishable-api-keys/${await keyOf("publishable")(c)}`, run: tickFirstRow("Sales Channels") },
  { id: "form-create-secret-key", page: `${GS}/developer/secret-api-keys`, alt: "Create secret API key form", route: `${S}/secret-api-keys/create`, clip: modalClip },
  { id: "secret-key-details", page: `${GS}/developer/secret-api-keys`, alt: "Secret API key details page", route: async (c) => `${S}/secret-api-keys/${await keyOf("secret")(c)}` },
  { id: "form-edit-secret-key", page: `${GS}/developer/secret-api-keys`, alt: "Edit secret API key form", route: async (c) => `${S}/secret-api-keys/${await keyOf("secret")(c)}/edit`, clip: modalClip },
  { id: "workflow-details", page: `${GS}/developer/workflows`, alt: "Workflow executions details", route: async (c) => `${S}/workflows/${await workflowId(c)}` },
  { id: "workflow-history", page: `${GS}/developer/workflows`, alt: "Workflow executions history", route: async (c) => `${S}/workflows/${await workflowId(c)}`, run: async (page) => { await page.mouse.move(700, 500); await page.mouse.wheel(0, 700); await page.waitForTimeout(500) } },

  // locations and shipping
  { id: "form-create-location", page: `${GS}/locations-and-shipping/locations`, alt: "Create location form", route: `${S}/locations/create`, clip: modalClip },
  { id: "location-details", page: `${GS}/locations-and-shipping/locations`, alt: "Locations details page", route: async (c) => `${S}/locations/${await locationId(c)}` },
  { id: "form-edit-location", page: `${GS}/locations-and-shipping/locations`, alt: "Edit location form", route: async (c) => `${S}/locations/${await locationId(c)}/edit`, clip: modalClip },
  { id: "form-create-service-zone", page: `${GS}/locations-and-shipping/locations`, alt: "Create service zone form", route: async (c) => { const z = await zoneInfo(c); return `${S}/locations/${z.loc}/fulfillment-set/${z.set}/service-zones/create` }, clip: modalClip },
  { id: "form-create-shipping-option-1", page: `${GS}/locations-and-shipping/locations`, alt: "Create shipping option form step 1", route: zoneRoute("/shipping-option/create") },
  { id: "form-create-shipping-option-2", page: `${GS}/locations-and-shipping/locations`, alt: "Create shipping option form step 2", route: zoneRoute("/shipping-option/create"), run: shippingOptionFlow(2) },
  { id: "form-conditional-price", page: `${GS}/locations-and-shipping/locations`, alt: "Conditional shipping option price form", route: zoneRoute("/shipping-option/create"), run: shippingOptionFlow(3), clip: modalClip },
  { id: "form-edit-service-zone", page: `${GS}/locations-and-shipping/locations`, alt: "Edit service zone form", route: zoneRoute("/edit"), clip: modalClip },
  { id: "form-service-zone-areas", page: `${GS}/locations-and-shipping/locations`, alt: "Manage service zone areas", route: zoneRoute("/areas"), clip: modalClip },
  { id: "form-edit-shipping-option", page: `${GS}/locations-and-shipping/locations`, alt: "Edit shipping option details form", route: async (c) => { const z = await zoneInfo(c); return `${S}/locations/${z.loc}/fulfillment-set/${z.set}/service-zone/${z.zone}/shipping-option/${z.option}/edit` }, clip: modalClip },
  { id: "form-shipping-option-prices", page: `${GS}/locations-and-shipping/locations`, alt: "Edit shipping option prices form", route: async (c) => { const z = await zoneInfo(c); return `${S}/locations/${z.loc}/fulfillment-set/${z.set}/service-zone/${z.zone}/shipping-option/${z.option}/pricing` } },
  { id: "form-location-channels", page: `${GS}/locations-and-shipping/locations`, alt: "Change associated sales channels", route: async (c) => `${S}/locations/${await locationId(c)}/sales-channels` },
  { id: "form-location-providers", page: `${GS}/locations-and-shipping/locations`, alt: "Manage fulfillment providers", route: async (c) => `${S}/locations/${await locationId(c)}/fulfillment-providers` },

  // shipping option types and shipping profiles
  { id: "shipping-option-types-list", page: `${GS}/locations-and-shipping/shipping-option-types`, alt: "Shipping Option Types list", route: `${S}/locations/shipping-option-types` },
  { id: "form-create-shipping-option-type", page: `${GS}/locations-and-shipping/shipping-option-types`, alt: "Create shipping option type form", route: `${S}/locations/shipping-option-types/create`, clip: modalClip },
  { id: "shipping-option-type-details", page: `${GS}/locations-and-shipping/shipping-option-types`, alt: "Shipping Option Type details page", route: async (c) => `${S}/locations/shipping-option-types/${await firstOf("/shipping-option-types", "shipping_option_types")(c)}` },
  { id: "form-edit-shipping-option-type", page: `${GS}/locations-and-shipping/shipping-option-types`, alt: "Edit shipping option type form", route: async (c) => `${S}/locations/shipping-option-types/${await firstOf("/shipping-option-types", "shipping_option_types")(c)}/edit`, clip: modalClip },
  { id: "shipping-profiles-list", page: `${GS}/locations-and-shipping/shipping-profiles`, alt: "Shipping profiles list", route: `${S}/locations/shipping-profiles` },
  { id: "form-create-shipping-profile", page: `${GS}/locations-and-shipping/shipping-profiles`, alt: "Create shipping profile form", route: `${S}/locations/shipping-profiles/create`, clip: modalClip },
  { id: "shipping-profile-details", page: `${GS}/locations-and-shipping/shipping-profiles`, alt: "Shipping profile details", route: async (c) => `${S}/locations/shipping-profiles/${await firstOf("/shipping-profiles", "shipping_profiles")(c)}` },

  // product tags and types
  { id: "product-tags-list", page: `${GS}/product-tags`, alt: "Product tags list", route: `${S}/product-tags` },
  { id: "form-create-product-tag", page: `${GS}/product-tags`, alt: "Create product tag form", route: `${S}/product-tags/create`, clip: modalClip },
  { id: "product-tag-details", page: `${GS}/product-tags`, alt: "Product tag details", route: async (c) => `${S}/product-tags/${await firstOf("/product-tags", "product_tags")(c)}` },
  { id: "form-edit-product-tag", page: `${GS}/product-tags`, alt: "Edit product tag form", route: async (c) => `${S}/product-tags/${await firstOf("/product-tags", "product_tags")(c)}/edit`, clip: modalClip },
  { id: "product-types-list", page: `${GS}/product-types`, alt: "Product types list", route: `${S}/product-types` },
  { id: "form-create-product-type", page: `${GS}/product-types`, alt: "Create product type form", route: `${S}/product-types/create`, clip: modalClip },
  { id: "product-type-details", page: `${GS}/product-types`, alt: "Product type details", route: async (c) => `${S}/product-types/${await firstOf("/product-types", "product_types")(c)}` },
  { id: "form-edit-product-type", page: `${GS}/product-types`, alt: "Edit product type form", route: async (c) => `${S}/product-types/${await firstOf("/product-types", "product_types")(c)}/edit`, clip: modalClip },

  // profile, refund and return reasons
  { id: "form-edit-profile", page: `${GS}/profile`, alt: "Edit profile form", route: `${S}/profile/edit`, clip: modalClip },
  { id: "refund-reasons-list", page: `${GS}/refund-reasons`, alt: "Refund reasons list", route: `${S}/refund-reasons` },
  { id: "form-create-refund-reason", page: `${GS}/refund-reasons`, alt: "Create refund reason form", route: `${S}/refund-reasons/create`, clip: modalClip },
  { id: "form-edit-refund-reason", page: `${GS}/refund-reasons`, alt: "Edit refund reason form", route: async (c) => `${S}/refund-reasons/${await firstOf("/refund-reasons", "refund_reasons")(c)}/edit`, clip: modalClip },
  { id: "return-reasons-list", page: `${GS}/return-reasons`, alt: "Return reasons list", route: `${S}/return-reasons` },
  { id: "form-create-return-reason", page: `${GS}/return-reasons`, alt: "Create return reason form", route: `${S}/return-reasons/create`, clip: modalClip },
  { id: "form-edit-return-reason", page: `${GS}/return-reasons`, alt: "Edit return reason form", route: async (c) => `${S}/return-reasons/${await firstOf("/return-reasons", "return_reasons")(c)}/edit`, clip: modalClip },

  // regions
  { id: "regions-list", page: `${GS}/regions`, alt: "Regions list", route: `${S}/regions` },
  { id: "form-create-region", page: `${GS}/regions`, alt: "Create region form", route: `${S}/regions/create` },
  { id: "region-details", page: `${GS}/regions`, alt: "Region details", route: async (c) => `${S}/regions/${await regionId(c)}` },
  { id: "form-edit-region", page: `${GS}/regions`, alt: "Edit region details form", route: async (c) => `${S}/regions/${await regionId(c)}/edit`, clip: modalClip },
  { id: "region-remove-countries", page: `${GS}/regions`, alt: "Remove countries from region", route: async (c) => `${S}/regions/${await regionId(c)}`, run: tickFirstRow("Countries") },

  // sales channels
  { id: "sales-channels-list", page: `${GS}/sales-channels`, alt: "Sales channels list", route: `${S}/sales-channels` },
  { id: "form-create-sales-channel", page: `${GS}/sales-channels`, alt: "Create sales channel form", route: `${S}/sales-channels/create`, clip: modalClip },
  { id: "sales-channel-details", page: `${GS}/sales-channels`, alt: "Sales channel details", route: async (c) => `${S}/sales-channels/${await channelId(c)}` },
  { id: "form-edit-sales-channel", page: `${GS}/sales-channels`, alt: "Edit sales channel form", route: async (c) => `${S}/sales-channels/${await channelId(c)}/edit`, clip: modalClip },
  { id: "form-channel-add-products", page: `${GS}/sales-channels`, alt: "Add products to sales channel", route: async (c) => `${S}/sales-channels/${await channelId(c)}/add-products` },
  { id: "channel-remove-products", page: `${GS}/sales-channels`, alt: "Remove products from sales channel", route: async (c) => `${S}/sales-channels/${await channelId(c)}`, run: tickFirstRow("Products") },

  // store
  { id: "store-details", page: `${GS}/store`, alt: "Store Details", route: `${S}/store` },
  { id: "form-edit-store", page: `${GS}/store`, alt: "Edit store form", route: `${S}/store/edit`, clip: modalClip },
  { id: "form-store-currencies", page: `${GS}/store`, alt: "Add currencies to store", route: `${S}/store/currencies` },
  { id: "store-remove-currencies", page: `${GS}/store`, alt: "Remove currencies from store", route: `${S}/store`, run: tickFirstRow("Currencies") },

  // tax regions
  { id: "tax-regions-list", page: `${GS}/tax-regions`, alt: "Tax regions list", route: `${S}/tax-regions` },
  { id: "form-create-tax-region", page: `${GS}/tax-regions`, alt: "Create tax region form", route: `${S}/tax-regions/create`, clip: modalClip },
  { id: "tax-region-details", page: `${GS}/tax-regions`, alt: "Tax region details", route: async (c) => `${S}/tax-regions/${await taxRegionOf("dk")(c)}` },
  { id: "form-edit-tax-region", page: `${GS}/tax-regions`, alt: "Edit tax region form", route: async (c) => `${S}/tax-regions/${await taxRegionOf("dk")(c)}/edit`, clip: modalClip },
  { id: "form-edit-default-tax-rate", page: `${GS}/tax-regions`, alt: "Edit default tax rate form", route: async (c) => {
    const id = await taxRegionOf("dk")(c)
    const r = (await c.api(`/tax-regions/${id}?fields=*tax_rates`)).tax_region
    return `${S}/tax-regions/${id}/tax-rates/${r.tax_rates[0].id}/edit`
  }, clip: modalClip },
  { id: "form-create-tax-override", page: `${GS}/tax-regions`, alt: "Create tax rate override form", route: async (c) => `${S}/tax-regions/${await taxRegionOf("us")(c)}/overrides/create`, clip: modalClip },
  { id: "form-edit-tax-override", page: `${GS}/tax-regions`, alt: "Edit tax rate override form", route: async (c) => {
    const id = await taxRegionOf("us")(c)
    const r = (await c.api(`/tax-regions/${id}?fields=*tax_rates,*tax_rates.rules`)).tax_region
    const o = r.tax_rates.find((x) => x.rules?.length) || r.tax_rates[0]
    return `${S}/tax-regions/${id}/overrides/${o.id}/edit`
  }, clip: modalClip },
  { id: "tax-sublevel-section", page: `${GS}/tax-regions`, alt: "Sublevel regions section", route: async (c) => `${S}/tax-regions/${await taxRegionOf("us")(c)}`, el: cardRe(/^(States|Provinces)/) },
  { id: "form-create-tax-sublevel", page: `${GS}/tax-regions`, alt: "Create sublevel region form", route: async (c) => `${S}/tax-regions/${await taxRegionOf("dk")(c)}/provinces/create`, clip: modalClip },
  { id: "tax-sublevel-details", page: `${GS}/tax-regions`, alt: "Sublevel tax region details page", route: async (c) => `${S}/tax-regions/${await taxRegionOf("us", "ca")(c)}` },

  { id: "tax-sublevel-alert", page: `${GS}/tax-regions`, alt: "Enable sublevel regions alert", route: async (c) => `${S}/tax-regions/${await taxRegionOf("dk")(c)}`, el: (p) => p.getByText("Sublevel regions are disabled").first().locator("xpath=ancestor::div[contains(@class,'shadow-elevation-card-rest') or contains(@class,'rounded-lg')][1]") },
  { id: "tax-region-actions-icon", page: `${GS}/tax-regions`, alt: "The icon to click on", route: async (c) => `${S}/tax-regions/${await taxRegionOf("dk")(c)}`, run: async (page) => {
    // the "..." of the region's header card
    await page.mouse.click(1385, 219)
    await page.getByRole("menuitem").first().waitFor()
  }, clip: async (page) => {
    const menu = await page.getByRole("menu").boundingBox()
    return { x: 232, y: 182, width: 1203, height: menu.y + menu.height - 182 + 24 }
  } },


  // users and invites
  { id: "users-list", page: `${GS}/users`, alt: "Users List", route: `${S}/users` },
  { id: "user-details", page: `${GS}/users`, alt: "User details page", route: async (c) => `${S}/users/${await idOf("/users", "users", (u) => u.email === "sara.holm@example.com")(c)}` },
  { id: "form-edit-user", page: `${GS}/users`, alt: "Edit user details form", route: async (c) => `${S}/users/${await idOf("/users", "users", (u) => u.email === "sara.holm@example.com")(c)}/edit`, clip: modalClip },
  { id: "form-invite-user", page: `${GS}/users/invites`, alt: "Invite user form", route: `${S}/users/invite`, clip: modalClip },
  { id: "invites-table", page: `${GS}/users/invites`, alt: "Pending invites table", route: `${S}/users/invite`, clip: async (page) => {
    // "Pending Invites" and its table, without the invite form above
    return page.evaluate(() => {
      const d = [...document.querySelectorAll("[role=dialog]")].pop()
      const h = [...d.querySelectorAll("*")].find((e) => !e.children.length && /^Pending Invites$/.test(e.textContent.trim())).getBoundingClientRect()
      const t = [...d.querySelectorAll("*")].filter((e) => !e.children.length && /of \d+ results/.test(e.textContent)).pop().getBoundingClientRect()
      return { x: h.x - 48, y: h.y - 20, width: 820, height: t.bottom - h.y + 100 }
    })
  } },


  // =========================== The admin overview ===============================================
  { id: "admin-overview", page: `${UG}/index`, alt: "Medusa Admin Overview", route: "/orders" },
  { id: "sidebar-menu", page: `${UG}/index`, alt: "Sidebar Menu", route: "/orders", clip: async () => ({ x: 0, y: 0, width: 220, height: 807 }) },
  { id: "sidebar-shortcuts", page: `${UG}/index`, alt: "Sidebar Shortcuts", route: "/orders", run: async (page) => {
    await page.getByRole("link", { name: "Products", exact: true }).first().hover()
    await page.waitForTimeout(2200)
  }, clip: async () => ({ x: 0, y: 40, width: 420, height: 260 }) },
  { id: "sidebar-settings-profile", page: `${UG}/index`, alt: "Settings and profile", route: "/orders", clip: async () => ({ x: 0, y: 687, width: 220, height: 120 }) },
  { id: "profile-quick-actions", page: `${UG}/index`, alt: "Profile Quick Actions", route: "/orders", run: async (page) => {
    await page.getByText("Alex Nielsen").last().click()
    await page.getByRole("menuitem").first().waitFor()
  }, clip: async () => ({ x: 0, y: 427, width: 440, height: 380 }) },
  { id: "search-window", page: `${UG}/index`, alt: "Search Window", route: "/orders", run: async (page) => {
    await page.keyboard.press("Meta+k")
    await page.waitForTimeout(900)
  } },
  { id: "notifications-window", page: `${UG}/index`, alt: "Notifications Window", route: "/orders", run: async (page) => {
    await page.mouse.click(1410, 26)
    await page.waitForTimeout(900)
  } },
  { id: "dark-theme", page: `${UG}/index`, alt: "Dark Theme", route: "/orders", run: async (page) => {
    await page.evaluate(() => { sessionStorage.setItem("shoot-theme", "dark"); localStorage.setItem("medusa_admin_theme", "dark") })
    await page.reload({ waitUntil: "networkidle" })
    await page.waitForTimeout(900)
  }, after: (page) => page.evaluate(() => sessionStorage.removeItem("shoot-theme")) },
  { id: "reset-password-form", page: `${UG}/reset-password`, alt: "Reset password form", route: () => {
    // the dashboard only decodes the token to show the form
    const b64 = (o) => Buffer.from(JSON.stringify(o)).toString("base64url")
    const now = Math.floor(Date.now() / 1000)
    return `/reset-password?token=${b64({ alg: "HS256", typ: "JWT" })}.${b64({ entity_id: "admin@medusa-test.com", provider: "emailpass", iat: now, exp: now + 900 })}.x`
  }, anonymous: true },


  // =========================== Tips: lists and the bulk editor ==================================
  { id: "tips-list-overview", page: `${UG}/tips/lists`, alt: "List Overview", route: "/products" },
  { id: "tips-list-filters", page: `${UG}/tips/lists`, alt: "Add Filters", route: "/products", run: async (page) => {
    await page.getByRole("button", { name: "Add filter" }).click()
    await page.getByRole("menuitem").first().waitFor()
  } },
  { id: "tips-list-search", page: `${UG}/tips/lists`, alt: "Search Items", route: "/products", run: async (page) => {
    await page.getByPlaceholder("Search").first().fill("Sweat")
    await page.waitForTimeout(1500)
  } },
  { id: "tips-list-sort", page: `${UG}/tips/lists`, alt: "Sort Items", route: "/products", run: async (page) => {
    await page.locator("main button[aria-haspopup=menu]").first().click()
    await page.waitForTimeout(700)
  } },
  { id: "tips-bulk-editor", page: `${UG}/tips/bulk-editor`, alt: "Bulk Editor Example", route: tshirtRoute("/prices") },
  { id: "tips-bulk-columns", page: `${UG}/tips/bulk-editor`, alt: "Change viewed columns", route: tshirtRoute("/prices"), run: async (page) => {
    await page.getByRole("button", { name: "View" }).click()
    await page.waitForTimeout(600)
  } },
  { id: "tips-bulk-select", page: `${UG}/tips/bulk-editor`, alt: "Select multiple cells", route: tshirtRoute("/prices"), run: async (page) => {
    // drag from the first price cell down to the third one
    await page.mouse.move(233, 183); await page.mouse.down(); await page.mouse.move(233, 262, { steps: 6 }); await page.mouse.up()
    await page.waitForTimeout(500)
  } },
  { id: "tips-bulk-dot", page: `${UG}/tips/bulk-editor`, alt: "Draggable blue dot", route: tshirtRoute("/prices"), run: async (page) => {
    await page.mouse.click(233, 224)
    await page.waitForTimeout(500)
  }, clip: async () => ({ x: 150, y: 150, width: 330, height: 150 }) },

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
  { id: "draft-form-billing-address", page: `${UG}/orders/draft-orders/manage`, alt: "Edit billing address in draft order form", route: draftRoute("/billing-address"), clip: modalClip },
]

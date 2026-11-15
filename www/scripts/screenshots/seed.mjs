// Fills a fresh MedusaGo database with the demo store used for the documentation screenshots.
// Everything goes through the public Admin and Store APIs (like a real merchant would).
//   node scripts/screenshots/seed.mjs [http://localhost:9000]
// The admin user must exist: medusa user -e admin@medusa-test.com -p supersecret
const BASE = process.argv[2] || "http://localhost:9000"
const EMAIL = process.env.SEED_EMAIL || "admin@medusa-test.com"
const PASSWORD = process.env.SEED_PASSWORD || "supersecret"

let token
let pk
async function call(method, path, body, headers = {}) {
  const res = await fetch(BASE + path, {
    method,
    headers: { "content-type": "application/json", ...(token ? { authorization: `Bearer ${token}` } : {}), ...headers },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await res.text()
  let json
  try { json = text ? JSON.parse(text) : {} } catch { json = { raw: text } }
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 400)}`)
  return json
}
const admin = (m, p, b) => call(m, "/admin" + p, b)
const store = (m, p, b) => call(m, "/store" + p, b, { "x-publishable-api-key": pk })
const log = (...a) => console.log(...a)

token = (await call("POST", "/auth/user/emailpass", { email: EMAIL, password: PASSWORD })).token

// --- store, sales channel, regions -------------------------------------------------------------
const { stores } = await admin("GET", "/stores")
const storeRec = stores[0]
const { sales_channels } = await admin("GET", "/sales-channels")
const sc = sales_channels[0]
await admin("POST", `/stores/${storeRec.id}`, {
  name: "MedusaGo Store",
  supported_currencies: [
    { currency_code: "eur", is_default: true },
    { currency_code: "usd" },
  ],
})
const eu = (await admin("POST", "/regions", {
  name: "Europe", currency_code: "eur", payment_providers: ["pp_system_default"],
  countries: ["dk", "fr", "de", "it", "es", "se", "gb"],
})).region
const us = (await admin("POST", "/regions", {
  name: "United States", currency_code: "usd", payment_providers: ["pp_system_default"], countries: ["us"],
})).region
for (const [cc, rate] of [["de", 19], ["dk", 25], ["fr", 20], ["us", 0]]) {
  await admin("POST", "/tax-regions", { country_code: cc, provider_id: "tp_system", default_tax_rate: { name: "Default", rate, code: "default" } }).catch((e) => log("tax region", cc, e.message.slice(0, 120)))
}
log("regions", eu.id, us.id)

// --- stock location, fulfillment ------------------------------------------------------------
const sl = (await admin("POST", "/stock-locations", {
  name: "European Warehouse",
  address: { address_1: "Copenhagen Warehouse 1", city: "Copenhagen", country_code: "dk", postal_code: "2100" },
})).stock_location
const webshop = (await admin("POST", "/sales-channels", { name: "Webshop", description: "Our online store" })).sales_channel
const b2b = (await admin("POST", "/sales-channels", { name: "B2B Portal", description: "Orders from business customers" })).sales_channel
const channels = [webshop, b2b]
await admin("POST", `/stock-locations/${sl.id}/sales-channels`, { add: [sc.id, webshop.id, b2b.id] })
await admin("POST", `/stock-locations/${sl.id}/fulfillment-providers`, { add: ["manual_manual"] })
await admin("POST", `/stock-locations/${sl.id}/fulfillment-sets`, { name: "European Warehouse delivery", type: "delivery" })
const fset = (await admin("GET", `/stock-locations/${sl.id}?fields=*fulfillment_sets`)).stock_location.fulfillment_sets
  .find((f) => f.name === "European Warehouse delivery")
const zone = (await admin("POST", `/fulfillment-sets/${fset.id}/service-zones`, {
  name: "Europe", geo_zones: ["dk", "fr", "de", "it", "es", "se", "gb"].map((c) => ({ country_code: c, type: "country" })),
})).fulfillment_set.service_zones.find((z) => z.name === "Europe")
const profiles = (await admin("GET", "/shipping-profiles")).shipping_profiles
const profile = profiles.find((p) => p.type === "default") || profiles[0]
const rules = [
  { attribute: "enabled_in_store", value: "true", operator: "eq" },
  { attribute: "is_return", value: "false", operator: "eq" },
]
for (const [name, code, desc, eur, usd] of [
  ["Standard Shipping", "standard", "Ship in 2-3 days.", 10, 12],
  ["Express Shipping", "express", "Ship in 24 hours.", 20, 24],
]) {
  await admin("POST", "/shipping-options", {
    name, price_type: "flat", provider_id: "manual_manual", service_zone_id: zone.id, shipping_profile_id: profile.id,
    type: { label: name.split(" ")[0], description: desc, code },
    prices: [{ currency_code: "eur", amount: eur }, { currency_code: "usd", amount: usd }],
    rules,
  })
}
const returnOption = (await admin("POST", "/shipping-options", {
  name: "Return Shipping", price_type: "flat", provider_id: "manual_manual", service_zone_id: zone.id, shipping_profile_id: profile.id,
  type: { label: "Return", description: "Send the items back.", code: "return" },
  prices: [{ currency_code: "eur", amount: 5 }, { currency_code: "usd", amount: 6 }],
  rules: [{ attribute: "enabled_in_store", value: "false", operator: "eq" }, { attribute: "is_return", value: "true", operator: "eq" }],
})).shipping_option
log("fulfillment ready", zone.id)

// --- publishable key ----------------------------------------------------------------------------
const key = (await admin("POST", "/api-keys", { title: "Webshop", type: "publishable" })).api_key
await admin("POST", `/api-keys/${key.id}/sales-channels`, { add: [sc.id, webshop.id, b2b.id] })
pk = key.token

// --- catalogue ----------------------------------------------------------------------------------
const cats = {}
for (const name of ["Shirts", "Sweatshirts", "Pants", "Merch"]) {
  cats[name] = (await admin("POST", "/product-categories", { name, is_active: true })).product_category.id
}
const collection = (await admin("POST", "/collections", { title: "Spring Collection", handle: "spring" })).collection.id
const sizes = ["S", "M", "L", "XL"]
const catalogue = [
  ["Medusa T-Shirt", "t-shirt", "Shirts", 10, 12, "Reimagine the feeling of a classic T-shirt. With our cotton T-shirts, everyday essentials no longer have to be ordinary."],
  ["Medusa Sweatshirt", "sweatshirt", "Sweatshirts", 10, 12, "Reimagine the feeling of a classic sweatshirt. With our cotton sweatshirt, everyday essentials no longer have to be ordinary."],
  ["Medusa Sweatpants", "sweatpants", "Pants", 10, 12, "Reimagine the feeling of classic sweatpants. With our cotton sweatpants, everyday essentials no longer have to be ordinary."],
  ["Medusa Shorts", "shorts", "Pants", 10, 12, "Reimagine the feeling of classic shorts. With our cotton shorts, everyday essentials no longer have to be ordinary."],
  ["Medusa Mug", "mug", "Merch", 8, 9, "A ceramic mug with the Medusa logo, made to last."],
]
const products = []
for (const [title, handle, cat, eur, usd, description] of catalogue) {
  const isMug = title.endsWith("Mug")
  const p = (await admin("POST", "/products", {
    title, handle, description, status: "published", shipping_profile_id: profile.id,
    categories: [{ id: cats[cat] }], collection_id: collection, sales_channels: [sc, webshop, b2b].map((c) => ({ id: c.id })),
    options: [{ title: isMug ? "Color" : "Size", values: isMug ? ["White"] : sizes }],
    variants: (isMug ? ["White"] : sizes).map((v) => ({
      title: v, sku: `${handle.toUpperCase()}-${v.toUpperCase()}`, manage_inventory: true,
      options: { [isMug ? "Color" : "Size"]: v },
      prices: [{ currency_code: "eur", amount: eur }, { currency_code: "usd", amount: usd }],
    })),
  })).product
  products.push(p)
}
log("products", products.length)

// stock: every inventory item gets a level at the warehouse
const items = (await admin("GET", "/inventory-items?limit=200")).inventory_items
for (const it of items) {
  await admin("POST", `/inventory-items/${it.id}/location-levels`, { location_id: sl.id, stocked_quantity: 100 + Math.floor(Math.random() * 100) })
}
log("inventory levels", items.length)

// --- customers --------------------------------------------------------------------------------
const group = (await admin("POST", "/customer-groups", { name: "VIP" })).customer_group
const people = [
  ["Emma", "Larsen", "emma.larsen@example.com"], ["Noah", "Berg", "noah.berg@example.com"], ["Olivia", "Schmidt", "olivia.schmidt@example.com"],
  ["Liam", "Dubois", "liam.dubois@example.com"], ["Mia", "Rossi", "mia.rossi@example.com"], ["Lucas", "Garcia", "lucas.garcia@example.com"],
]
const customers = []
for (const [first_name, last_name, email] of people) {
  customers.push((await admin("POST", "/customers", { first_name, last_name, email, company_name: undefined })).customer)
}
await admin("POST", `/customer-groups/${group.id}/customers`, { add: [customers[0].id, customers[1].id] })

// --- return reasons, promotions ---------------------------------------------------------------
const reasons = []
for (const [value, label] of [["wrong_size", "Wrong size"], ["damaged", "Damaged"], ["not_as_described", "Not as described"]]) {
  reasons.push(await admin("POST", "/return-reasons", { value, label }).then((r) => r.return_reason).catch((e) => log("return reason", e.message.slice(0, 100))))
}
const campaign = (await admin("POST", "/campaigns", {
  name: "Summer Sale", campaign_identifier: "SUMMER26", description: "Seasonal discounts",
  starts_at: new Date(Date.now() - 7 * 864e5).toISOString(), ends_at: new Date(Date.now() + 60 * 864e5).toISOString(),
  budget: { type: "usage", limit: 1000 },
})).campaign
await admin("POST", "/promotions", {
  code: "SUMMER10", type: "standard", status: "active", is_automatic: false, campaign_id: campaign.id,
  application_method: { type: "percentage", target_type: "items", allocation: "across", value: 10, currency_code: "eur" },
}).catch((e) => log("promotion", e.message.slice(0, 200)))
await admin("POST", "/promotions", {
  code: "FREESHIP", type: "standard", status: "active", is_automatic: false,
  application_method: { type: "fixed", target_type: "shipping_methods", allocation: "each", value: 10, max_quantity: 1, currency_code: "eur" },
}).catch((e) => log("promotion 2", e.message.slice(0, 200)))

// --- orders (real checkouts through the Store API) ------------------------------------------
const variants = (await admin("GET", "/products?fields=+variants.id,+variants.title&limit=50")).products
const allVariants = variants.flatMap((p) => p.variants.map((v) => ({ id: v.id, product: p.title, title: v.title })))
const pick = (n) => allVariants[(n * 3) % allVariants.length]
const countries = ["dk", "de", "fr", "it", "es", "se"]
const orders = []
for (let i = 0; i < customers.length * 2; i++) {
  const c = customers[i % customers.length]
  const cc = countries[i % countries.length]
  const address = { first_name: c.first_name, last_name: c.last_name, address_1: `${10 + i} Main Street`, city: "City", postal_code: `${1000 + i}`, country_code: cc, phone: "+4500000000" }
  const cart = (await store("POST", "/carts", {
    region_id: eu.id, email: c.email, sales_channel_id: channels[i % 2].id,
    items: [{ variant_id: pick(i).id, quantity: 1 + (i % 3) }, { variant_id: pick(i + 4).id, quantity: 1 }],
  })).cart
  await store("POST", `/carts/${cart.id}`, { shipping_address: address, billing_address: address, email: c.email })
  const opts = (await store("GET", `/shipping-options?cart_id=${cart.id}`)).shipping_options
  await store("POST", `/carts/${cart.id}/shipping-methods`, { option_id: opts[i % opts.length].id })
  const pc = (await store("POST", "/payment-collections", { cart_id: cart.id })).payment_collection
  await store("POST", `/payment-collections/${pc.id}/payment-sessions`, { provider_id: "pp_system_default" })
  const done = await store("POST", `/carts/${cart.id}/complete`, {})
  if (done.type !== "order") throw new Error("checkout failed: " + JSON.stringify(done).slice(0, 300))
  orders.push(done.order)
}
log("orders", orders.length)

// --- order lifecycle: payments, fulfillments, shipments, deliveries, one refund --------------------
// outcome per order index: [capture, fulfil, ship, deliver, refund]
const lifecycle = [
  [1, 1, 1, 1, 0], [1, 1, 1, 0, 0], [1, 1, 0, 0, 0], [1, 0, 0, 0, 0], [1, 1, 1, 1, 1], [0, 0, 0, 0, 0],
  [1, 1, 1, 1, 0], [1, 1, 0, 0, 0], [1, 0, 0, 0, 0], [1, 1, 1, 0, 0], [0, 0, 0, 0, 0], [1, 1, 1, 1, 0],
]
for (let i = 0; i < orders.length; i++) {
  const [capture, fulfil, ship, deliver, refund] = lifecycle[i]
  const id = orders[i].id
  const o = (await admin("GET", `/orders/${id}?fields=*items,+payment_collections.payments.id,+payment_collections.payments.amount`)).order
  const payment = o.payment_collections?.[0]?.payments?.[0]
  if (capture && payment) await admin("POST", `/payments/${payment.id}/capture`, {})
  if (fulfil) {
    const f = (await admin("POST", `/orders/${id}/fulfillments`, { location_id: sl.id, items: o.items.map((it) => ({ id: it.id, quantity: it.quantity })) })).order
    const fid = (await admin("GET", `/orders/${id}?fields=+fulfillments.id`)).order.fulfillments.at(-1).id
    if (ship) await admin("POST", `/orders/${id}/fulfillments/${fid}/shipments`, { items: o.items.map((it) => ({ id: it.id, quantity: it.quantity })), labels: [{ tracking_number: `TRK${100000 + i}`, tracking_url: "https://example.com/track", label_url: "https://example.com/label" }] })
    if (deliver) await admin("POST", `/orders/${id}/fulfillments/${fid}/mark-as-delivered`, {})
  }
  if (refund && payment) await admin("POST", `/payments/${payment.id}/refund`, { amount: 10 })
}
log("lifecycle done")

// --- returns, claim, exchange and order edit in progress -----------------------------------------
const itemsOf = async (n) => (await admin("GET", `/orders/${orders[n - 1].id}?fields=*items`)).order.items
const tryStep = async (label, fn) => { try { await fn() } catch (e) { log("!!", label, e.message.slice(0, 260)) } }
await tryStep("return", async () => {
  const [it] = await itemsOf(7)
  const ret = (await admin("POST", "/returns", { order_id: orders[6].id, location_id: sl.id })).return
  await admin("POST", `/returns/${ret.id}/request-items`, { items: [{ id: it.id, quantity: 1, reason_id: reasons[0].id }] })
  await admin("POST", `/returns/${ret.id}/shipping-method`, { shipping_option_id: returnOption.id })
  await admin("POST", `/returns/${ret.id}/request`, {})
})
await tryStep("claim", async () => {
  const [it] = await itemsOf(12)
  const claim = (await admin("POST", "/claims", { order_id: orders[11].id, type: "replace" })).claim
  await admin("POST", `/claims/${claim.id}/inbound/items`, { items: [{ id: it.id, quantity: 1, reason_id: reasons[1].id }] })
  await admin("POST", `/claims/${claim.id}/outbound/items`, { items: [{ variant_id: it.variant_id, quantity: 1 }] })
  await admin("POST", `/claims/${claim.id}/request`, {})
})
await tryStep("exchange", async () => {
  const [it] = await itemsOf(10)
  const ex = (await admin("POST", "/exchanges", { order_id: orders[9].id })).exchange
  await admin("POST", `/exchanges/${ex.id}/inbound/items`, { items: [{ id: it.id, quantity: 1, reason_id: reasons[0].id }] })
  await admin("POST", `/exchanges/${ex.id}/outbound/items`, { items: [{ variant_id: pick(7).id, quantity: 1 }] })
  await admin("POST", `/exchanges/${ex.id}/request`, {})
})
await tryStep("order edit", async () => {
  const edit = (await admin("POST", "/order-edits", { order_id: orders[8].id })).order_change
  await admin("POST", `/order-edits/${orders[8].id}/items`, { items: [{ variant_id: pick(9).id, quantity: 1 }] })
  await admin("POST", `/order-edits/${orders[8].id}/request`, {})
})

// --- reservation, draft order, price lists, secret key --------------------------------------------
await admin("POST", "/reservations", { inventory_item_id: items[0].id, location_id: sl.id, quantity: 5, description: "Reserved for the spring photo shoot" }).catch((e) => log("reservation", e.message.slice(0, 160)))
await admin("POST", "/draft-orders", {
  email: customers[2].email, region_id: eu.id, sales_channel_id: webshop.id, customer_id: customers[2].id,
  items: [{ variant_id: pick(1).id, quantity: 2 }],
  shipping_address: { first_name: customers[2].first_name, last_name: customers[2].last_name, address_1: "12 Nyhavn", city: "Copenhagen", postal_code: "1051", country_code: "dk", phone: "+4512345678" },
  billing_address: { first_name: customers[2].first_name, last_name: customers[2].last_name, address_1: "12 Nyhavn", city: "Copenhagen", postal_code: "1051", country_code: "dk" },
}).catch((e) => log("draft order", e.message.slice(0, 200)))
const sale = (variantIds, amount) => variantIds.map((variant_id) => ({ variant_id, amount, currency_code: "eur" }))
await admin("POST", "/price-lists", { title: "Black Friday", description: "Black Friday prices", status: "active", type: "sale", prices: sale(allVariants.slice(0, 4).map((v) => v.id), 7) }).catch((e) => log("price list", e.message.slice(0, 200)))
await admin("POST", "/price-lists", { title: "Wholesale", description: "Prices for B2B customers", status: "draft", type: "override", prices: sale(allVariants.slice(4, 8).map((v) => v.id), 6) }).catch((e) => log("price list 2", e.message.slice(0, 200)))
await admin("POST", "/api-keys", { title: "Backend integration", type: "secret" })
log("done. publishable key:", pk)

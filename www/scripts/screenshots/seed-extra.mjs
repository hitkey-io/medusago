// Extra demo data on top of seed.mjs, for pictures that need a record of a kind the main seed does not create:
//   - an order that is picked up in the store (a pickup fulfillment set and option), fulfilled and awaiting pickup
//   node scripts/screenshots/seed-extra.mjs [http://localhost:9000]
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
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status} ${text.slice(0, 400)}`)
  return text ? JSON.parse(text) : {}
}
const admin = (m, p, b) => call(m, "/admin" + p, b)
const store = (m, p, b) => call(m, "/store" + p, b, { "x-publishable-api-key": pk })

token = (await call("POST", "/auth/user/emailpass", { email: EMAIL, password: PASSWORD })).token
pk = (await admin("GET", "/api-keys?type=publishable")).api_keys[0].token
const sl = (await admin("GET", "/stock-locations?fields=*fulfillment_sets")).stock_locations[0]
const region = (await admin("GET", "/regions")).regions.find((r) => r.name === "Europe") || (await admin("GET", "/regions")).regions[0]
const customer = (await admin("GET", "/customers?limit=1")).customers[0]
const profile = (await admin("GET", "/shipping-profiles")).shipping_profiles[0]

// pickup: skipped when the stand already has it (the script can be run again)
if (!sl.fulfillment_sets.some((f) => f.type === "pickup")) {
  // pickup: a fulfillment set of the "pickup" type with one option
  let pickup = undefined
  if (!pickup) {
    await admin("POST", `/stock-locations/${sl.id}/fulfillment-sets`, { name: "European Warehouse pickup", type: "pickup" })
    pickup = (await admin("GET", `/stock-locations/${sl.id}?fields=*fulfillment_sets`)).stock_location.fulfillment_sets.find((f) => f.type === "pickup")
  }
  const zone = (await admin("POST", `/fulfillment-sets/${pickup.id}/service-zones`, {
    name: "Copenhagen", geo_zones: [{ country_code: "dk", type: "country" }],
  })).fulfillment_set.service_zones.find((z) => z.name === "Copenhagen")
  const option = (await admin("POST", "/shipping-options", {
    name: "Store Pickup", price_type: "flat", provider_id: "manual_manual", service_zone_id: zone.id, shipping_profile_id: profile.id,
    type: { label: "Pickup", description: "Pick the order up at the warehouse.", code: "pickup" },
    prices: [{ currency_code: "eur", amount: 0 }, { currency_code: "usd", amount: 0 }],
    rules: [{ attribute: "enabled_in_store", value: "true", operator: "eq" }, { attribute: "is_return", value: "false", operator: "eq" }],
  })).shipping_option

  // the order, through a real checkout
  const variants = (await admin("GET", "/products?fields=+variants.id&limit=5")).products.flatMap((p) => p.variants)
  const address = { first_name: customer.first_name, last_name: customer.last_name, address_1: "1 Harbour Street", city: "Copenhagen", postal_code: "2100", country_code: "dk", phone: "+4500000000" }
  const cart = (await store("POST", "/carts", { region_id: region.id, email: customer.email, items: [{ variant_id: variants[2].id, quantity: 1 }] })).cart
  await store("POST", `/carts/${cart.id}`, { shipping_address: address, billing_address: address, email: customer.email })
  await store("POST", `/carts/${cart.id}/shipping-methods`, { option_id: option.id })
  const pc = (await store("POST", "/payment-collections", { cart_id: cart.id })).payment_collection
  await store("POST", `/payment-collections/${pc.id}/payment-sessions`, { provider_id: "pp_system_default" })
  const done = await store("POST", `/carts/${cart.id}/complete`, {})
  if (done.type !== "order") throw new Error("checkout failed: " + JSON.stringify(done).slice(0, 300))
  const id = done.order.id
  const o = (await admin("GET", `/orders/${id}?fields=*items,+payment_collections.payments.id`)).order
  await admin("POST", `/payments/${o.payment_collections[0].payments[0].id}/capture`, {})
  await admin("POST", `/orders/${id}/fulfillments`, { location_id: sl.id, items: o.items.map((it) => ({ id: it.id, quantity: it.quantity })) })
  console.log("pickup order", done.order.display_id, id)
}

// --- promotion conditions: who can use SUMMER10 and what it applies to, and a "buy X get Y" promotion -----------------
const promos = (await admin("GET", "/promotions?limit=50&fields=*rules")).promotions
const summer = promos.find((p) => p.code === "SUMMER10")
const vip = (await admin("GET", "/customer-groups")).customer_groups.find((g) => g.name === "VIP")
const tshirt = (await admin("GET", "/products")).products.find((p) => p.title === "Medusa T-Shirt")
const sweatshirt = (await admin("GET", "/products")).products.find((p) => p.title === "Medusa Sweatshirt")
const has = summer && (await admin("GET", `/promotions/${summer.id}/target-rules`)).rules?.length
if (summer && vip && !has) {
  await admin("POST", `/promotions/${summer.id}/rules/batch`, { create: [{ attribute: "customer.groups.id", operator: "in", values: [vip.id] }] }).catch((e) => console.log("rules", e.message.slice(0, 160)))
  await admin("POST", `/promotions/${summer.id}/target-rules/batch`, { create: [{ attribute: "items.product.id", operator: "in", values: [tshirt.id, sweatshirt.id] }] }).catch((e) => console.log("target rules", e.message.slice(0, 160)))
}
if (!promos.find((p) => p.code === "BUY2GET1")) {
  await admin("POST", "/promotions", {
    code: "BUY2GET1", type: "buyget", status: "active", is_automatic: false,
    application_method: {
      type: "percentage", target_type: "items", allocation: "each", value: 100, max_quantity: 1, apply_to_quantity: 1, buy_rules_min_quantity: 2,
      buy_rules: [{ attribute: "items.product.id", operator: "eq", values: [tshirt.id] }],
      target_rules: [{ attribute: "items.product.id", operator: "eq", values: [tshirt.id] }],
    },
  }).catch((e) => console.log("buy-get", e.message.slice(0, 200)))
}

// --- product types and tags, tax sublevels and overrides, more users, pending invites ------------------------------------
const allProducts = (await admin("GET", "/products?fields=id,title,*tags&limit=50")).products
if (!(await admin("GET", "/product-types")).product_types.length) {
  const types = {}
  for (const value of ["Clothing", "Accessories"]) types[value] = (await admin("POST", "/product-types", { value })).product_type
  const tags = []
  for (const value of ["Summer", "Cotton", "Bestseller"]) tags.push((await admin("POST", "/product-tags", { value })).product_tag)
  for (const [i, p] of allProducts.entries()) {
    await admin("POST", `/products/${p.id}`, { type_id: (p.title.includes("Mug") ? types.Accessories : types.Clothing).id, tags: tags.filter((_, j) => (i + j) % 2 === 0).map((t) => ({ id: t.id })) })
  }
}

const taxRegions = (await admin("GET", "/tax-regions?fields=id,country_code,parent_id&limit=50")).tax_regions
const us = taxRegions.find((r) => r.country_code === "us" && !r.parent_id)
if (us && !taxRegions.some((r) => r.parent_id === us.id)) {
  const ca = (await admin("POST", "/tax-regions", { country_code: "us", province_code: "ca", parent_id: us.id, default_tax_rate: { name: "California sales tax", rate: 7.25, code: "ca-sales" } })).tax_region
  await admin("POST", "/tax-rates", { tax_region_id: us.id, name: "Clothing exemption", code: "clothing", rate: 0, rules: [{ reference: "product_type", reference_id: (await admin("GET", "/product-types")).product_types[0].id }] }).catch((e) => console.log("override", e.message.slice(0, 160)))
}

const users = (await admin("GET", "/users")).users
if (users.length < 3) {
  for (const [email, first_name, last_name] of [["sara.holm@example.com", "Sara", "Holm"], ["jonas.berg@example.com", "Jonas", "Berg"]]) {
    const invite = (await admin("POST", "/invites", { email })).invite
    const reg = await call("POST", "/auth/user/emailpass/register", { email, password: "supersecret" })
    await call("POST", `/admin/invites/accept?token=${invite.token}`, { email, first_name, last_name }, { authorization: `Bearer ${reg.token}` }).catch((e) => console.log("accept", e.message.slice(0, 200)))
  }
}
if (!(await admin("GET", "/invites")).invites.length) {
  for (const email of ["amelie.martin@example.com", "tobias.kruger@example.com"]) await admin("POST", "/invites", { email }).catch((e) => console.log("invite", e.message.slice(0, 160)))
}

// --- the admin user's name and the store's defaults (shown in the profile, the users list and the store details) --------
await admin("POST", "/users/me", { first_name: "Alex", last_name: "Nielsen" }).catch((e) => console.log("profile", e.message.slice(0, 160)))
{
  const store = (await admin("GET", "/stores")).stores[0]
  await admin("POST", `/stores/${store.id}`, { default_region_id: region.id, default_location_id: sl.id }).catch((e) => console.log("store defaults", e.message.slice(0, 200)))
}

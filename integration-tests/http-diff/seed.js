// Seeds the HTTP harness database through the upstream workflows, so the
// data is exactly what Medusa itself writes. Run by generate-seed.sh with
// `medusa exec` from the reference application.
const { createRequire } = require("module")
const req = createRequire(process.cwd() + "/package.json")
const {
  createRegionsWorkflow,
  createTaxRegionsWorkflow,
  createSalesChannelsWorkflow,
  createStockLocationsWorkflow,
  linkSalesChannelsToStockLocationWorkflow,
  createProductCategoriesWorkflow,
  createCollectionsWorkflow,
  createProductTypesWorkflow,
  createProductTagsWorkflow,
  createProductsWorkflow,
  createInventoryLevelsWorkflow,
  deleteProductsWorkflow,
  updateStoresWorkflow,
} = req("@medusajs/medusa/core-flows")

exports.default = async function seed({ container }) {
  const query = container.resolve("query")
  const storeModule = container.resolve("store")
  const salesChannelModule = container.resolve("sales_channel")

  const [store] = await storeModule.listStores()
  const [defaultChannel] = await salesChannelModule.listSalesChannels({ name: "Default Sales Channel" })

  await updateStoresWorkflow(container).run({
    input: {
      selector: { id: store.id },
      update: {
        supported_currencies: [
          { currency_code: "eur", is_default: true },
          { currency_code: "usd" },
        ],
      },
    },
  })

  const { result: [webChannel] } = await createSalesChannelsWorkflow(container).run({
    input: { salesChannelsData: [{ name: "Webshop", description: "Online store" }] },
  })

  const { result: regions } = await createRegionsWorkflow(container).run({
    input: {
      regions: [
        { name: "Europe", currency_code: "eur", countries: ["de", "fr", "it"], payment_providers: ["pp_system_default"] },
        { name: "United States", currency_code: "usd", countries: ["us"], payment_providers: ["pp_system_default"] },
      ],
    },
  })
  await createTaxRegionsWorkflow(container).run({
    input: ["de", "fr", "it", "us"].map((country_code) => ({ country_code, provider_id: "tp_system" })),
  })

  const { result: [warehouse] } = await createStockLocationsWorkflow(container).run({
    input: {
      locations: [
        { name: "European Warehouse", address: { city: "Berlin", country_code: "DE", address_1: "Main 1" } },
      ],
    },
  })
  await linkSalesChannelsToStockLocationWorkflow(container).run({
    input: { id: warehouse.id, add: [defaultChannel.id, webChannel.id] },
  })

  const { result: [apparel] } = await createProductTypesWorkflow(container).run({
    input: { product_types: [{ value: "Apparel" }] },
  })
  const { result: [summer] } = await createCollectionsWorkflow(container).run({
    input: { collections: [{ title: "Summer", handle: "summer" }] },
  })
  const { result: [cotton, sale] } = await createProductTagsWorkflow(container).run({
    input: { product_tags: [{ value: "cotton" }, { value: "sale" }] },
  })
  const { result: [clothing] } = await createProductCategoriesWorkflow(container).run({
    input: { product_categories: [{ name: "Clothing", handle: "clothing", is_active: true }] },
  })
  const { result: [shirts] } = await createProductCategoriesWorkflow(container).run({
    input: {
      product_categories: [
        { name: "Shirts", handle: "shirts", parent_category_id: clothing.id, is_active: true },
      ],
    },
  })

  const {
    data: [shippingProfile],
  } = await query.graph({ entity: "shipping_profile", fields: ["id"] })

  const { result: products } = await createProductsWorkflow(container).run({
    input: {
      products: [
        {
          title: "Medusa T-Shirt",
          handle: "t-shirt",
          status: "published",
          description: "A comfortable cotton t-shirt",
          weight: 400,
          type_id: apparel.id,
          collection_id: summer.id,
          tag_ids: [cotton.id, sale.id],
          category_ids: [shirts.id],
          shipping_profile_id: shippingProfile.id,
          metadata: { featured: true },
          images: [{ url: "https://example.com/shirt-front.png" }, { url: "https://example.com/shirt-back.png" }],
          thumbnail: "https://example.com/shirt-front.png",
          options: [
            { title: "Size", values: ["S", "M"] },
            { title: "Color", values: ["Black", "White"] },
          ],
          variants: [
            {
              title: "S / Black",
              sku: "SHIRT-S-BLACK",
              manage_inventory: true,
              options: { Size: "S", Color: "Black" },
              prices: [
                { amount: 10, currency_code: "eur" },
                { amount: 15, currency_code: "usd" },
                { amount: 9.5, currency_code: "eur", rules: { region_id: regions[0].id } },
              ],
            },
            {
              title: "M / White",
              sku: "SHIRT-M-WHITE",
              manage_inventory: true,
              options: { Size: "M", Color: "White" },
              prices: [
                { amount: 12, currency_code: "eur" },
                { amount: 17, currency_code: "usd", min_quantity: 2 },
              ],
            },
          ],
          sales_channels: [{ id: defaultChannel.id }, { id: webChannel.id }],
        },
        {
          title: "Medusa Sweatpants",
          handle: "sweatpants",
          status: "published",
          collection_id: summer.id,
          category_ids: [clothing.id],
          shipping_profile_id: shippingProfile.id,
          options: [{ title: "Size", values: ["M", "L"] }],
          variants: [
            {
              title: "M",
              sku: "PANTS-M",
              manage_inventory: false,
              options: { Size: "M" },
              prices: [{ amount: 30, currency_code: "eur" }],
            },
            {
              title: "L",
              sku: "PANTS-L",
              allow_backorder: true,
              options: { Size: "L" },
              prices: [{ amount: 32.99, currency_code: "eur" }],
            },
          ],
          sales_channels: [{ id: defaultChannel.id }],
        },
        {
          title: "Draft Hoodie",
          handle: "hoodie",
          status: "draft",
          discountable: false,
          shipping_profile_id: shippingProfile.id,
          options: [{ title: "Default option", values: ["Default option value"] }],
          variants: [
            { title: "Default", options: { "Default option": "Default option value" }, prices: [{ amount: 50, currency_code: "usd" }] },
          ],
        },
        {
          title: "Deleted Cap",
          handle: "cap",
          status: "published",
          shipping_profile_id: shippingProfile.id,
          options: [{ title: "Size", values: ["One size"] }],
          variants: [{ title: "One size", sku: "CAP", options: { Size: "One size" }, prices: [{ amount: 5, currency_code: "eur" }] }],
        },
      ],
    },
  })

  const { data: inventoryItems } = await query.graph({ entity: "inventory_item", fields: ["id", "sku"] })
  await createInventoryLevelsWorkflow(container).run({
    input: {
      inventory_levels: inventoryItems
        .filter((i) => i.sku !== "CAP")
        .map((i, n) => ({ inventory_item_id: i.id, location_id: warehouse.id, stocked_quantity: 10 * (n + 1) })),
    },
  })

  await deleteProductsWorkflow(container).run({ input: { ids: [products[3].id] } })
}

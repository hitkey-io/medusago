// Seeds the differential test database through the upstream module
// services, so the data is exactly what Medusa itself writes. Executed with
// `medusa exec` from the reference application.

exports.default = async function seed({ container }) {
  const region = container.resolve("region")
  const product = container.resolve("product")

  await region.createRegions([
    { name: "Europe", currency_code: "eur", countries: ["de", "fr", "it"], metadata: { zone: "eu" } },
    { name: "North America", currency_code: "usd", countries: ["us", "ca"], automatic_taxes: false },
    { name: "Archived", currency_code: "gbp", countries: ["gb"] },
  ])
  const [archived] = await region.listRegions({ name: "Archived" })
  await region.softDeleteRegions([archived.id])

  const [apparel] = await product.createProductTypes([{ value: "Apparel" }])
  const [summer] = await product.createProductCollections([{ title: "Summer", handle: "summer" }])
  const [tagA, tagB] = await product.createProductTags([{ value: "cotton" }, { value: "sale" }])
  const [clothing] = await product.createProductCategories([{ name: "Clothing", handle: "clothing", is_active: true }])
  await product.createProductCategories([{ name: "Shirts", handle: "shirts", parent_category_id: clothing.id, is_active: true, rank: 0 }])

  await product.createProducts([
    {
      title: "Medusa T-Shirt",
      handle: "t-shirt",
      status: "published",
      description: "A comfortable cotton t-shirt",
      weight: 400,
      type_id: apparel.id,
      collection_id: summer.id,
      tag_ids: [tagA.id, tagB.id],
      category_ids: [clothing.id],
      metadata: { featured: true },
      options: [
        { title: "Size", values: ["S", "M", "L"] },
        { title: "Color", values: ["Black", "White"] },
      ],
      variants: [
        { title: "S / Black", sku: "SHIRT-S-BLACK", options: { Size: "S", Color: "Black" }, variant_rank: 0 },
        { title: "M / Black", sku: "SHIRT-M-BLACK", options: { Size: "M", Color: "Black" }, variant_rank: 1 },
        { title: "L / White", sku: "SHIRT-L-WHITE", options: { Size: "L", Color: "White" }, variant_rank: 2 },
      ],
      images: [{ url: "https://example.com/shirt-1.png" }, { url: "https://example.com/shirt-2.png" }],
    },
    {
      title: "Medusa Sweatpants",
      handle: "sweatpants",
      status: "draft",
      options: [{ title: "Size", values: ["M"] }],
      variants: [{ title: "M", sku: "PANTS-M", options: { Size: "M" } }],
    },
    {
      title: "Discontinued Hoodie",
      handle: "hoodie",
      status: "published",
      tag_ids: [tagB.id],
    },
  ])

  const [hoodie] = await product.listProducts({ handle: "hoodie" })
  await product.softDeleteProducts([hoodie.id])

  const [shirt] = await product.listProducts({ handle: "t-shirt" }, { relations: ["variants"] })
  const white = shirt.variants.find((v) => v.sku === "SHIRT-L-WHITE")
  await product.softDeleteProductVariants([white.id])
}

const { defineLink } = require("@medusajs/framework/utils")

// The links of the brand test module, defined only when the module is
// configured (MEDUSA_TEST_BRAND=true): a link needs both modules loaded.
const def = (m) => m.default ?? m

if (process.env.MEDUSA_TEST_BRAND === "true") {
  const BrandModule = def(require("../modules/brand"))
  const ProductModule = def(require("@medusajs/medusa/product"))
  const SalesChannelModule = def(require("@medusajs/medusa/sales-channel"))
  const CurrencyModule = def(require("@medusajs/medusa/currency"))

  // product -> brand (a brand has many products)
  module.exports.productBrand = defineLink(
    { linkable: ProductModule.linkable.product, isList: true },
    BrandModule.linkable.brand
  )

  // brand <-> sales channels, with extra columns
  module.exports.brandSalesChannel = defineLink(
    BrandModule.linkable.brand,
    { linkable: SalesChannelModule.linkable.salesChannel, isList: true },
    {
      database: {
        table: "brand_sales_channel",
        extraColumns: {
          priority: { type: "integer", defaultValue: "0" },
          featured: { type: "boolean", defaultValue: "false" },
          note: { type: "text", nullable: true },
          weight: { type: "decimal", nullable: true },
          settings: { type: "json", nullable: true },
          starts_at: { type: "datetime", nullable: true },
        },
      },
    }
  )

  // brand.currency_code -> currency (read-only)
  module.exports.brandCurrency = defineLink(
    { linkable: BrandModule.linkable.brand, field: "currency_code" },
    CurrencyModule.linkable.currency,
    { readOnly: true }
  )
}

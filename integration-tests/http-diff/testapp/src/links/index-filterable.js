const { defineLink } = require("@medusajs/framework/utils")

// A link with filterable fields (the index module adds them to its schema),
// defined with MEDUSA_TEST_BRAND=true and MEDUSA_TEST_INDEX_FILTERABLE=true.
// The Go version is DefineIndexFilterableLink (testapp/brand/links.go).
const def = (m) => m.default ?? m

if (
  process.env.MEDUSA_TEST_BRAND === "true" &&
  process.env.MEDUSA_TEST_INDEX_FILTERABLE === "true"
) {
  const BrandModule = def(require("../modules/brand"))
  const CustomerModule = def(require("@medusajs/medusa/customer"))

  module.exports.brandCustomerGroup = defineLink(
    {
      linkable: BrandModule.linkable.brand,
      filterable: ["name", "status", "rank", "launched_at"],
    },
    {
      linkable: CustomerModule.linkable.customerGroup,
      isList: true,
      filterable: ["name"],
    }
  )
}

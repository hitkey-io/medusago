const { model } = require("@medusajs/framework/utils")

const BrandRegion = model.define("brand_region", {
  id: model.id({ prefix: "breg" }).primaryKey(),
  code: model.text(),
  brand: model.belongsTo(() => require("./brand").default, { mappedBy: "regions" }),
})

exports.default = BrandRegion

const { model } = require("@medusajs/framework/utils")

const BrandTag = model.define("brand_tag", {
  id: model.id({ prefix: "btag" }).primaryKey(),
  value: model.text(),
  brands: model.manyToMany(() => require("./brand").default, { mappedBy: "brand_tags" }),
})

exports.default = BrandTag

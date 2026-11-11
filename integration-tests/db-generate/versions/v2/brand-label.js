const { model } = require("@medusajs/framework/utils")

const BrandLabel = model.define("brand_label", {
  id: model.id({ prefix: "blabel" }).primaryKey(),
  value: model.text(),
  brands: model.manyToMany(() => require("./brand").default, { mappedBy: "labels" }),
})

exports.default = BrandLabel

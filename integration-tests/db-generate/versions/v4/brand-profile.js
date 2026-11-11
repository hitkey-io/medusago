const { model } = require("@medusajs/framework/utils")

const BrandProfile = model.define("brand_profile", {
  id: model.id({ prefix: "bprof" }).primaryKey(),
  website: model.text().nullable(),
  brand: model.belongsTo(() => require("./brand").default, { mappedBy: "profile" }),
})

exports.default = BrandProfile

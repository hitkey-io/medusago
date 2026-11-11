const { model } = require("@medusajs/framework/utils")

const BrandLogo = model.define("brand_logo", {
  id: model.id({ prefix: "blogo" }).primaryKey(),
  url: model.text(),
  position: model.number().default(0),
  brand: model.belongsTo(() => require("./brand").default, { mappedBy: "logos" }),
})

exports.default = BrandLogo

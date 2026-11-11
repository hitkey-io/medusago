const { model } = require("@medusajs/framework/utils")

const BrandOwner = model.define("brand_owner", {
  id: model.id({ prefix: "bown" }).primaryKey(),
  name: model.text(),
})

exports.default = BrandOwner

const { model } = require("@medusajs/framework/utils")

// composite primary key
const BrandStat = model.define("brand_stat", {
  brand_id: model.text().primaryKey(),
  period: model.text().primaryKey(),
  views: model.number().default(0),
  score: model.number().nullable(),
  notes: model.text().nullable(),
})

exports.default = BrandStat

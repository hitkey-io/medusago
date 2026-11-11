const { model } = require("@medusajs/framework/utils")

// composite primary key, with a third key column
const BrandStat = model.define("brand_stat", {
  brand_id: model.text().primaryKey(),
  period: model.text().primaryKey(),
  region: model.text().primaryKey(),
  views: model.number().default(0),
  score: model.number().nullable(),
  notes: model.text().nullable(),
})

exports.default = BrandStat

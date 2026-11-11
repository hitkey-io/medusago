const { model } = require("@medusajs/framework/utils")

const BrandNote = model.define("brand_note", {
  id: model.autoincrement().primaryKey(),
  body: model.text(),
  brand: model.belongsTo(() => require("./brand").default, { mappedBy: "notes" }),
})

exports.default = BrandNote

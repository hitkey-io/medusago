const { model } = require("@medusajs/framework/utils")

const BrandColor = model
  .define("brand_color", {
    id: model.id({ prefix: "bcol" }).primaryKey(),
    hex: model.text().default("#000000"),
    tone: model.enum(["light", "dark"]).default("light"),
    ratio: model.float().default(1.5),
    weight: model.number().index("IDX_brand_color_weight_custom").nullable(),
    is_primary: model.boolean().default(false),
    palette: model.json().default(["red", "blue"]),
    shades: model.array().nullable(),
    brand: model.belongsTo(() => require("./brand").default, { mappedBy: "colors" }),
  })
  .indexes([
    { on: ["tone"], type: "brin" },
    { name: "IDX_brand_color_ratio_all", on: ["ratio"], where: null },
    { on: ["hex", "tone"], where: "is_primary = true" },
  ])

exports.default = BrandColor

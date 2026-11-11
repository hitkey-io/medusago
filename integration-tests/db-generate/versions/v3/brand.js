const { model } = require("@medusajs/framework/utils")

// Version 3: a renamed column (and its index), a removed model (with its
// pivot table), enum items removed, a column type change, a removed index,
// a changed check and a changed foreign key (no more delete cascade).
const Brand = model
  .define("brand", {
    id: model.id({ prefix: "brand" }).primaryKey(),
    name: model.text().searchable(),
    handle: model.text().unique(),
    description: model.text(),
    tagline: model.text().index().nullable(),
    is_active: model.boolean().default(true),
    rank: model.number().default(10),
    rating: model.float().default(0),
    revenue: model.bigNumber().nullable(),
    budget: model.bigNumber().default(100),
    status: model.enum(["draft", "published", "retired"]).default("draft"),
    country: model.enum(["us", "de"]).default("us"),
    tags: model.array().default(["new"]),
    metadata: model.json().nullable(),
    settings: model.json().default({ theme: "dark" }),
    launched_at: model.dateTime().nullable(),
    founded_year: model.text().nullable(),
    sequence: model.autoincrement(),
    logos: model.hasMany(() => require("./brand-logo").default, { mappedBy: "brand" }),
    profile: model.hasOne(() => require("./brand-profile").default, { mappedBy: "brand" }).nullable(),
    regions: model.hasMany(() => require("./brand-region").default, { mappedBy: "brand" }),
  })
  .indexes([
    { on: ["name", "status"], where: "deleted_at IS NULL" },
    { name: "IDX_brand_rank_rating", on: ["rank", "rating"], where: "rank > 0" },
  ])
  .checks([{ name: "brand_rank_check", expression: "rank >= 1" }, (columns) => `${columns.rating} >= 0`])
  .cascades({ delete: ["profile"] })

exports.default = Brand

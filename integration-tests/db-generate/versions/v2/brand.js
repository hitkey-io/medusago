const { model } = require("@medusajs/framework/utils")

// Version 2: added, removed and changed properties (nullable, default,
// enum items, type), new relations (many to many with a generated pivot
// table, nullable belongs to), new indexes and checks.
const Brand = model
  .define("brand", {
    id: model.id({ prefix: "brand" }).primaryKey(),
    name: model.text().searchable(),
    handle: model.text().unique(),
    description: model.text(),
    slogan: model.text().index().nullable(),
    is_active: model.boolean().default(true),
    rank: model.number().default(10),
    rating: model.float().default(0),
    revenue: model.bigNumber().nullable(),
    budget: model.bigNumber().default(100),
    status: model.enum(["draft", "published", "archived", "retired"]).default("draft"),
    country: model.enum(["us", "de"]).nullable(),
    tags: model.array().default(["new"]),
    metadata: model.json().nullable(),
    settings: model.json().default({ theme: "dark" }),
    launched_at: model.dateTime().nullable(),
    founded_year: model.number().nullable(),
    sequence: model.autoincrement(),
    logos: model.hasMany(() => require("./brand-logo").default, { mappedBy: "brand" }),
    profile: model.hasOne(() => require("./brand-profile").default, { mappedBy: "brand" }).nullable(),
    regions: model.hasMany(() => require("./brand-region").default, { mappedBy: "brand" }),
    labels: model.manyToMany(() => require("./brand-label").default, { mappedBy: "brands" }),
  })
  .indexes([
    { on: ["name", "status"], where: "deleted_at IS NULL" },
    { on: ["handle", "country"], unique: true },
    { name: "IDX_brand_rank_rating", on: ["rank", "rating"], where: "rank > 0" },
  ])
  .checks([{ name: "brand_rank_check", expression: "rank >= 0" }, (columns) => `${columns.rating} >= 0`])
  .cascades({ delete: ["logos", "profile"] })

exports.default = Brand

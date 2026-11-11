const { model } = require("@medusajs/framework/utils")

// Version 6: a removed model other tables reference (orphaned foreign
// keys, the foreign key column renamed), a many to many relationship with
// its own pivot table and join columns, a has one relationship owning a
// unique foreign key, and a model with defaults of every kind, an index
// using another method and an index not scoped to the rows not soft
// deleted. (Numeric enums are left out: upstream cannot build the GraphQL
// schema of a service over them.)
const Brand = model
  .define("brand", {
    id: model.id({ prefix: "brand" }).primaryKey(),
    name: model.text().searchable(),
    handle: model.text().unique(),
    description: model.text(),
    tagline: model.text().index().nullable(),
    code: model.text().unique("IDX_brand_code_custom").nullable(),
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
    profile: model.hasOne(() => require("./brand-profile").default, { mappedBy: "brand" }).nullable(),
    regions: model.hasMany(() => require("./brand-region").default, { mappedBy: "brand" }),
    notes: model.hasMany(() => require("./brand-note").default, { mappedBy: "brand" }),
    colors: model.hasMany(() => require("./brand-color").default, { mappedBy: "brand" }),
    brand_tags: model.manyToMany(() => require("./brand-tag").default, {
      mappedBy: "brands",
      pivotTable: "brand_tag_links",
      joinColumn: "brand_ref_id",
      inverseJoinColumn: "tag_ref_id",
    }),
    owner: model.hasOne(() => require("./brand-owner").default, { mappedBy: undefined }).nullable(),
  })
  .indexes([
    { on: ["name", "status"], where: "deleted_at IS NULL" },
    { name: "IDX_brand_rank_rating", on: ["rank", "rating"], where: "rank > 0" },
  ])
  .checks([{ name: "brand_rank_check", expression: "rank >= 1" }, (columns) => `${columns.rating} >= 0`])
  .cascades({ delete: ["profile", "colors"] })

exports.default = Brand

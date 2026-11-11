const { model } = require("@medusajs/framework/utils")

// Test module of the http-diff harness (custom modules and links); the Go
// version is integration-tests/http-diff/testapp/brand.
const Brand = model
  .define("brand", {
    id: model.id({ prefix: "brand" }).primaryKey(),
    name: model.text().searchable(),
    handle: model.text().unique(),
    description: model.text().nullable(),
    is_active: model.boolean().default(true),
    rank: model.number().default(0),
    rating: model.float().nullable(),
    revenue: model.bigNumber().nullable(),
    status: model.enum(["draft", "published", "archived"]).default("draft"),
    tags: model.array().default([]),
    currency_code: model.text().nullable(),
    metadata: model.json().nullable(),
    launched_at: model.dateTime().nullable(),
    logos: model.hasMany(() => require("./brand-logo").default, { mappedBy: "brand" }),
    profile: model.hasOne(() => require("./brand-profile").default, { mappedBy: "brand" }).nullable(),
  })
  .indexes([{ on: ["name", "status"], where: "deleted_at IS NULL" }])
  .cascades({ delete: ["logos", "profile"] })

exports.default = Brand

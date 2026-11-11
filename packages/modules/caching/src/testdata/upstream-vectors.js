// Builds upstream-vectors.json: the hash, stable stringify, tags computed
// from query results and invalidation tags of events of the upstream
// caching module. Run from a reference application on the http-diff seed:
//   CACHING_VECTORS_OUT=<path>/upstream-vectors.json medusa exec ./upstream-vectors.js
const fs = require("fs")
const path = require("path")
const { createRequire } = require("module")

exports.default = async ({ container }) => {
  const req = createRequire(path.join(process.cwd(), "package.json"))
  const { MedusaModule } = req("@medusajs/framework/modules-sdk")
  const { GraphQLUtils } = req("@medusajs/framework/utils")
  const { CacheInvalidationParser } = req("@medusajs/caching/dist/utils/parser")
  const stringify = req("fast-json-stable-stringify")
  const xx = await req("xxhash-wasm")()

  const joinerConfigs = MedusaModule.getAllJoinerConfigs()
  const loadedSchema = joinerConfigs.map((c) => c?.schema ?? "").join("\n")
  const defaultMedusaSchema = `
    scalar DateTime
    scalar JSON
    directive @enumValue(value: String) on ENUM_VALUE
  `
  const { schema: cleaned } = GraphQLUtils.cleanGraphQLSchema(defaultMedusaSchema + loadedSchema)
  const merged = GraphQLUtils.mergeTypeDefs(cleaned)
  const schema = GraphQLUtils.makeExecutableSchema({ typeDefs: merged })
  const parser = new CacheInvalidationParser(schema, joinerConfigs)

  const query = container.resolve("query")
  const g = async (entity, fields, filters) =>
    JSON.parse(JSON.stringify(await query.graph({ entity, fields, filters })))
  const samples = [
    await g("product", ["*", "variants.*", "variants.options.*", "options.*", "options.values.*", "tags.*", "images.*", "collection.*", "type.*", "categories.*", "sales_channels.*"]),
    await g("product", ["id", "title", "variants.id", "variants.price_set.*", "variants.price_set.prices.*"]),
    await g("region", ["*", "countries.*", "payment_providers.*"]),
    await g("store", ["*", "supported_currencies.*"]),
    await g("customer", ["*", "addresses.*", "groups.*"]),
    await g("api_key", ["id", "token", "revoked_at", "sales_channels.id"]),
    await g("stock_location", ["*", "address.*", "sales_channels.*", "fulfillment_sets.*", "fulfillment_sets.service_zones.*"]),
    { data: [{ id: "prod_x", title: "a" }, { name: "no id" }], metadata: { count: 2, skip: 0, take: 20 } },
    [{ id: "prod_1" }, [{ id: "variant_1" }], { nested: { id: "reg_1" } }],
  ]
  const tagsOf = (entities, operation) => {
    const tags = new Set()
    parser.buildInvalidationEvents(entities, operation).forEach((e) => e.cacheKeys.forEach((k) => tags.add(k)))
    return Array.from(tags)
  }
  // Objects whose id has no registered prefix are matched against the
  // schema types in type map order, which follows the module load order
  // (nondeterministic upstream): the samples only hold registered ids.
  const out = {
    hashes: ["", "Product:list:*", "héllo wörld", "{\"a\":1}", "x".repeat(200)].map((s) => ({ input: s, hash: xx.h64ToString(s) })),
    stringify: [
      { b: 1, a: [1, { d: null, c: 2.5 }], e: "x\"y<>&", f: 1e21, g: 0.1, h: -0, i: true },
      { queryOptions: { entity: "product", fields: ["id", "title"], filters: { id: ["a", "b"] }, pagination: { take: 20, skip: 0, order: { created_at: "DESC" } } }, options: { locale: "fr-FR" } },
      [3, "a", null, [], {}],
    ].map((v) => ({ input: v, output: stringify(v) })),
    samples: samples.map((s) => {
      const entities = parser.parseObjectForEntities(s)
      return { input: s, entities, tags: entities.length ? tagsOf(entities) : [] }
    }),
    events: [
      ["product.updated", { id: "prod_1" }],
      ["product.product-variant.created", [{ id: ["variant_1", "variant_2"] }]],
      ["pricing.price.deleted", { id: "price_1" }],
      ["link-product-variant-price-set.attached", { id: "pvps_1" }],
      ["api_key.updated", { id: "apk_1" }],
    ].map(([name, data]) => {
      const operation = name.split(".").pop()
      const entityType = name.split(".").slice(-2).shift()
      const items = Array.isArray(data) ? data : [data]
      const tags = []
      for (const item of items) {
        for (const id of Array.isArray(item.id) ? item.id : [item.id]) {
          const { toCamelCase, upperCaseFirst } = req("@medusajs/framework/utils")
          tags.push(...tagsOf([{ type: upperCaseFirst(toCamelCase(entityType)), id }], operation))
        }
      }
      return { name, data, tags }
    }),
  }
  fs.writeFileSync(process.env.CACHING_VECTORS_OUT, JSON.stringify(out, null, 2) + "\n")
}

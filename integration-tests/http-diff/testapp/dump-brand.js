// Dumps what upstream derives from the brand test module, for the Go unit
// tests of integration-tests/http-diff/testapp/brand (testdata/upstream.json):
// the DML definitions, the MikroORM metadata, the module joiner config and
// linkable, and the joiner configs / relation map of the loaded application.
//
//   MEDUSA_TEST_BRAND=true medusa exec ./dump-brand.js <out.json>
const fs = require("fs")
const path = require("path")

exports.default = async function dumpBrand({ args }) {
  const req = require("module").createRequire(path.join(process.cwd(), "package.json"))
  const { toMikroORMEntity, GraphQLUtils } = req("@medusajs/framework/utils")
  const { MikroORM } = req("@medusajs/framework/mikro-orm/core")
  const { PostgreSqlDriver } = req("@medusajs/framework/mikro-orm/postgresql")
  const { MedusaModule } = req("@medusajs/framework/modules-sdk")

  const models = ["brand", "brand-logo", "brand-profile", "brand-stat"].map(
    (f) => require(path.join(process.cwd(), "src/modules/brand/models", f)).default
  )
  const orm = await MikroORM.init({
    driver: PostgreSqlDriver,
    entities: models.map((e) => toMikroORMEntity(e)),
    dbName: "unused",
    connect: false,
    discovery: { warnWhenNoEntities: false },
  })
  const metas = models.map((m) => {
    const meta = orm.getMetadata().find(toMikroORMEntity(m).name)
    const nameOf = (v) => (typeof v === "function" ? v().name : v?.className ?? v)
    return {
      className: meta.className,
      collection: meta.collection,
      primaryKeys: meta.primaryKeys,
      properties: Object.values(meta.properties).map((p) => ({
        name: p.name,
        kind: p.kind,
        type: typeof p.type === "string" ? p.type : p.type?.name,
        columnType: p.columnTypes?.length === 1 ? p.columnTypes[0] : p.columnType,
        fieldNames: p.fieldNames,
        nullable: !!p.nullable,
        primary: p.primary,
        persist: p.persist === false ? false : undefined,
        autoincrement: p.autoincrement,
        default: p.default,
        defaultRaw: p.defaultRaw,
        items: Array.isArray(p.items) ? p.items : undefined,
        entity: p.kind === "scalar" ? undefined : p.type,
        mappedBy: p.mappedBy,
        inversedBy: p.inversedBy,
        owner: p.owner,
        mapToPk: p.mapToPk,
        deleteRule: p.deleteRule,
        pivotTable: p.pivotTable,
        pivotEntity: p.pivotEntity ? nameOf(p.pivotEntity) : undefined,
        joinColumns: p.kind === "m:n" ? p.joinColumns : undefined,
        inverseJoinColumns: p.kind === "m:n" ? p.inverseJoinColumns : undefined,
        referencedColumnNames: p.kind === "scalar" ? undefined : p.referencedColumnNames,
      })),
    }
  })
  await orm.close(true)

  const mod = require(path.join(process.cwd(), "src/modules/brand")).default
  const linkable = {}
  for (const [k, v] of Object.entries(mod.linkable)) {
    linkable[k] = Object.fromEntries(Object.entries(v).filter(([n]) => n !== "toJSON"))
  }

  const joinerConfigs = []
  for (const m of MedusaModule.getLoadedModules()) {
    const loaded = Object.values(m)[0]
    if (!loaded?.__definition?.isQueryable) {
      continue
    }
    const { databaseClientUrl, ...config } = loaded.__joinerConfig
    joinerConfigs.push(config)
  }
  const loadedSchema = MedusaModule.getAllJoinerConfigs().map((c) => c?.schema ?? "").join("\n")
  const { schema: cleaned } = GraphQLUtils.cleanGraphQLSchema(
    "\n    scalar DateTime\n    scalar JSON\n    directive @enumValue(value: String) on ENUM_VALUE\n  " + loadedSchema
  )
  const schema = GraphQLUtils.makeExecutableSchema({ typeDefs: GraphQLUtils.mergeTypeDefs(cleaned) })
  const relations = {}
  for (const [entity, props] of GraphQLUtils.extractRelationsFromGQL(schema.getTypeMap())) {
    relations[entity] = Object.fromEntries(props)
  }

  fs.writeFileSync(
    args[0],
    JSON.stringify({ metas, joinerConfig: mod.service.prototype.__joinerConfig(), linkable, joinerConfigs, relationMap: relations }, null, 2)
  )
}

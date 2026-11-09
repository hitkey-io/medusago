// Dumps what the upstream Query is built from in a running application: the
// joiner configs of the queryable loaded modules (link modules included) and
// the relation map extracted from the merged GraphQL schema.
//
// Run from the reference application with `medusa exec`:
//   medusa exec <this file> <out.json>
// then `node scripts/extract-query-config.mjs <out.json>` generates Go code.

const fs = require("fs")

exports.default = async function extractQueryConfig({ args }) {
  // Resolve the packages of the application being executed.
  const appRequire = require("module").createRequire(process.cwd() + "/package.json")
  const { MedusaModule } = appRequire("@medusajs/framework/modules-sdk")
  const { GraphQLUtils } = appRequire("@medusajs/framework/utils")

  const joinerConfigs = []
  for (const mod of MedusaModule.getLoadedModules()) {
    const loaded = Object.values(mod)[0]
    if (!loaded?.__definition?.isQueryable) {
      continue
    }
    const { schema, ...config } = loaded.__joinerConfig
    joinerConfigs.push(config)
  }

  const loadedSchema = MedusaModule.getAllJoinerConfigs()
    .map((joinerConfig) => joinerConfig?.schema ?? "")
    .join("\n")
  const { schema: cleaned } = GraphQLUtils.cleanGraphQLSchema(`
    scalar DateTime
    scalar JSON
    directive @enumValue(value: String) on ENUM_VALUE
  ` + loadedSchema)
  const schema = GraphQLUtils.makeExecutableSchema({ typeDefs: GraphQLUtils.mergeTypeDefs(cleaned) })
  const relationMap = GraphQLUtils.extractRelationsFromGQL(schema.getTypeMap())

  const relations = {}
  for (const [entity, props] of relationMap) {
    relations[entity] = Object.fromEntries(props)
  }

  fs.writeFileSync(args[0], JSON.stringify({ joinerConfigs, relationMap: relations }, null, 2))
}

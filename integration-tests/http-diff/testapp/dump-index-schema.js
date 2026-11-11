// Dumps the schema object representation @medusajs/index builds
// (buildSchemaObjectRepresentation) for the default schema and for the
// schemas of testapp/index-schemas: the golden vectors of the Go port
// (packages/modules/index/src/utils/testdata).
//
// Run from the reference application with the index module enabled:
//   MEDUSA_FF_INDEX_ENGINE=true medusa exec ./dump-index-schema.js <out dir> [schema.graphql...]
// It writes <out dir>/default.json and <out dir>/<name>.json per schema
// ({ error } when upstream throws).
const fs = require("fs")
const path = require("path")

exports.default = async function dumpIndexSchema({ args }) {
  const req = require("module").createRequire(path.join(process.cwd(), "package.json"))
  const indexDir = path.dirname(req.resolve("@medusajs/index"))
  const { buildSchemaObjectRepresentation } = require(path.join(indexDir, "utils/build-config"))
  const { defaultSchema } = require(path.join(indexDir, "utils/default-schema"))
  const { baseGraphqlSchema } = require(path.join(indexDir, "utils/base-graphql-schema"))
  const { schemaObjectRepresentationPropertiesToOmit } = require(path.join(indexDir, "types"))

  const [outDir, ...files] = args
  fs.mkdirSync(outDir, { recursive: true })

  const dump = (schema) => {
    const { objectRepresentation, entitiesMap } = buildSchemaObjectRepresentation(baseGraphqlSchema + schema)
    const entities = []
    for (const [key, rep] of Object.entries(objectRepresentation)) {
      if (schemaObjectRepresentationPropertiesToOmit.includes(key)) {
        continue
      }
      entities.push({
        key,
        entity: rep.entity,
        alias: rep.alias ?? null,
        fields: rep.fields,
        listeners: rep.listeners,
        moduleConfig: rep.moduleConfig && {
          serviceName: rep.moduleConfig.serviceName,
          isLink: !!rep.moduleConfig.isLink,
          primaryKeys: rep.moduleConfig.primaryKeys,
          relationships: (rep.moduleConfig.isLink ? rep.moduleConfig.relationships ?? [] : []).map((r) => ({
            serviceName: r.serviceName,
            foreignKey: r.foreignKey,
          })),
        },
        parents: rep.parents.map((p) => ({
          ref: p.ref.entity,
          inSchemaRef: p.inSchemaRef?.entity ?? null,
          targetProp: p.targetProp,
          inverseSideProp: p.inverseSideProp ?? null,
          isList: p.isList ?? null,
          isInverse: p.isInverse ?? null,
        })),
      })
    }
    const propertiesMap = Object.entries(objectRepresentation._schemaPropertiesMap).map(([p, v]) => ({
      path: p,
      ref: v.ref.entity,
      shortCutOf: v.shortCutOf ?? null,
      isInverse: v.isInverse ?? null,
      isList: v.isList ?? null,
    }))
    const linkableKeys = {}
    for (const [service, config] of Object.entries(objectRepresentation._serviceNameModuleConfigMap)) {
      linkableKeys[service] = config.linkableKeys ?? {}
    }
    const types = {}
    for (const [name, type] of Object.entries(entitiesMap)) {
      if (!type.astNode || type.astNode.kind !== "ObjectTypeDefinition" || !type.getFields) {
        continue
      }
      types[name] = {}
      for (const [field, ref] of Object.entries(type.getFields())) {
        types[name][field] = ref.type.toString()
      }
    }
    return { entities, propertiesMap, linkableKeys, types }
  }

  const write = (name, value) => {
    fs.writeFileSync(path.join(outDir, name + ".json"), JSON.stringify(value, null, 2) + "\n")
  }

  write("default", dump(defaultSchema))
  for (const file of files) {
    const name = path.basename(file).replace(/\.graphql$/, "")
    try {
      write(name, dump(fs.readFileSync(file, "utf8")))
    } catch (e) {
      write(name, { error: e.message })
    }
  }
}

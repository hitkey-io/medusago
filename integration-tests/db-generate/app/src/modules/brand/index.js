const fs = require("fs")
const path = require("path")
const { Module, MedusaService, DmlEntity } = require("@medusajs/framework/utils")

// The service is generated over every model of the current version
// (generate.sh copies a version of the models into ./models).
const models = {}
const dir = path.join(__dirname, "models")
for (const file of fs.readdirSync(dir).sort()) {
  for (const value of Object.values(require(path.join(dir, file)))) {
    if (DmlEntity.isDmlEntity(value)) {
      models[value.name] = value
    }
  }
}

class BrandModuleService extends MedusaService(models) {}

exports.default = Module("brand", { service: BrandModuleService })

const { Module } = require("@medusajs/framework/utils")
const BrandModuleService = require("./service").default

exports.default = Module("brand", { service: BrandModuleService })

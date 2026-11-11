const { MedusaService } = require("@medusajs/framework/utils")
const Brand = require("./models/brand").default
const BrandLogo = require("./models/brand-logo").default
const BrandProfile = require("./models/brand-profile").default
const BrandStat = require("./models/brand-stat").default

class BrandModuleService extends MedusaService({ Brand, BrandLogo, BrandProfile, BrandStat }) {
  // composite primary key upsert through the internal service
  async upsertBrandStats(data) {
    const stats = await this.brandStatService_.upsert(data)
    return await this.baseRepository_.serialize(stats)
  }
}

exports.default = BrandModuleService

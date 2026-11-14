package brand

import utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"

// region: service
// BrandModuleService is the main service of the Brand Module. Embedding
// MedusaService gives it the generated methods of the module's data models
// (createBrands, retrieveBrand, listBrands, deleteBrands, ...), which you
// call by name with Call.
type BrandModuleService struct {
	*utilsmodulessdk.MedusaService
}

// endregion: service

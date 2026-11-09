package steps

// DeleteProductsStep mirrors deleteProductsStep (softDeleteProducts /
// restoreProducts).
var DeleteProductsStep = softDeleteStep("delete-products", "Product")

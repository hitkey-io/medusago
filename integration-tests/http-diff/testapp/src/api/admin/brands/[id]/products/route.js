// POST /admin/brands/:id/products { add: [product ids], remove: [product ids] }
// links products to the brand (custom product <-> brand link).
exports.POST = async (req, res) => {
  const link = req.scope.resolve("link")
  const add = (req.body.add ?? []).map((product_id) => ({
    product: { product_id },
    brand: { brand_id: req.params.id },
  }))
  const remove = (req.body.remove ?? []).map((product_id) => ({
    product: { product_id },
    brand: { brand_id: req.params.id },
  }))
  const created = add.length ? await link.create(add) : []
  const dismissed = remove.length ? await link.dismiss(remove) : []
  res.json({ created, dismissed })
}

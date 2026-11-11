// POST /admin/brands/:id/sales-channels { links: [{ sales_channel_id, data }] }
// links sales channels with the link's extra columns.
exports.POST = async (req, res) => {
  const link = req.scope.resolve("link")
  const created = await link.create(
    (req.body.links ?? []).map(({ sales_channel_id, data }) => ({
      brand: { brand_id: req.params.id },
      sales_channel: { sales_channel_id },
      ...(data ? { data } : {}),
    }))
  )
  res.json({ created })
}

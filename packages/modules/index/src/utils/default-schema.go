package utils

import modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"

// DefaultSchema mirrors defaultSchema: the index schema used when the
// module options have no `schema`.
var DefaultSchema = `
  type Product @Listeners(values: ["` + modulessdk.PRODUCT + `.product.created", "` + modulessdk.PRODUCT + `.product.updated", "` + modulessdk.PRODUCT + `.product.deleted", "` + modulessdk.PRODUCT + `.product.restored"]) {
    id: ID
    title: String
    handle: String
    status: String
    type_id: String
    collection_id: String
    is_giftcard: Boolean
    external_id: String
    created_at: DateTime
    updated_at: DateTime

    variants: [ProductVariant]
    sales_channels: [SalesChannel]
  }

  type ProductVariant @Listeners(values: ["` + modulessdk.PRODUCT + `.product-variant.created", "` + modulessdk.PRODUCT + `.product-variant.updated", "` + modulessdk.PRODUCT + `.product-variant.deleted", "` + modulessdk.PRODUCT + `.product-variant.restored"]) {
    id: ID
    product_id: String
    sku: String

    prices: [Price]
  }
  
  type Price @Listeners(values: ["` + modulessdk.PRICING + `.price.created", "` + modulessdk.PRICING + `.price.updated", "` + modulessdk.PRICING + `.price.deleted", "` + modulessdk.PRICING + `.price.restored"]) {
    id: ID
    amount: Float
    currency_code: String
    price_list_id: String
  }

  type SalesChannel @Listeners(values: ["` + modulessdk.SALES_CHANNEL + `.sales-channel.created", "` + modulessdk.SALES_CHANNEL + `.sales-channel.updated", "` + modulessdk.SALES_CHANNEL + `.sales-channel.deleted", "` + modulessdk.SALES_CHANNEL + `.sales-channel.restored"]) {
    id: ID
    is_disabled: Boolean
  }
`

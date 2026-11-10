// Package orders mirrors api/admin/orders.
package orders

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminOrderFields mirrors defaultAdminOrderFields.
var DefaultAdminOrderFields = []string{"id", "display_id", "custom_display_id", "status", "version", "summary", "total", "metadata", "locale", "created_at", "updated_at"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminOrderFields, DefaultLimit: 20, IsList: true, Entity: "order"}

// DefaultAdminRetrieveOrderFields mirrors defaultAdminRetrieveOrderFields.
var DefaultAdminRetrieveOrderFields = append(append([]string{}, DefaultAdminOrderFields...), "region_id", "total", "subtotal", "tax_total", "discount_total", "discount_tax_total", "original_total", "original_subtotal", "original_tax_total", "item_total", "item_subtotal", "item_tax_total", "original_item_total", "original_item_subtotal", "original_item_tax_total", "shipping_total", "shipping_subtotal", "shipping_tax_total", "original_shipping_tax_total", "original_shipping_subtotal", "original_shipping_total", "credit_line_total", "credit_line_subtotal", "credit_line_tax_total", "*items", "*credit_lines", "*items.tax_lines", "*items.adjustments", "*items.variant", "*items.variant.product", "*items.detail", "*shipping_address", "*billing_address", "*shipping_methods", "*shipping_methods.tax_lines", "*shipping_methods.adjustments", "*payment_collections", "*payment_collections.payments", "*payment_collections.payments.refunds", "*payment_collections.payments.captures")

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminRetrieveOrderFields, Entity: "order"}

// DefaultAdminRetrieveOrderChangesFields mirrors defaultAdminRetrieveOrderChangesFields.
var DefaultAdminRetrieveOrderChangesFields = []string{"id", "order_id", "return_id", "claim_id", "exchange_id", "version", "change_type", "*actions", "description", "status", "internal_note", "created_by", "requested_by", "requested_at", "confirmed_by", "confirmed_at", "declined_by", "declined_reason", "metadata", "declined_at", "canceled_by", "canceled_at", "created_at", "updated_at", "carry_over_promotions", "no_notification"}

// DefaultAdminOrderItemsFields mirrors defaultAdminOrderItemsFields.
var DefaultAdminOrderItemsFields = []string{"id", "order_id", "item_id", "version", "*item"}

// RetrieveOrderChangesTransformQueryConfig mirrors retrieveOrderChangesTransformQueryConfig.
var RetrieveOrderChangesTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminRetrieveOrderChangesFields}

// ListOrderItemsQueryConfig mirrors listOrderItemsQueryConfig.
var ListOrderItemsQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminOrderItemsFields, DefaultLimit: 100, IsList: true}

// ListShippingOptionsQueryConfig mirrors listShippingOptionsQueryConfig.
var ListShippingOptionsQueryConfig = httputils.QueryConfig{DefaultLimit: 100, IsList: true}

// DefaultAdminExportOrderFields mirrors defaultAdminExportOrderFields.
var DefaultAdminExportOrderFields = []string{"id", "display_id", "status", "created_at", "updated_at", "email", "currency_code", "region_id", "subtotal", "tax_total", "shipping_total", "discount_total", "gift_card_total", "total", "*customer", "*shipping_address", "*billing_address", "*sales_channel", "*items", "*shipping_methods", "*payment_collections", "*fulfillments"}

// ExportTransformQueryConfig mirrors exportTransformQueryConfig.
var ExportTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminExportOrderFields, IsList: true, Entity: "order"}

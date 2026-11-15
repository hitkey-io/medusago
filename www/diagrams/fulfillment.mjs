import { model, edge, parentWithTwo, providerFlow, providerModel, linkDiagram } from "./lib.mjs"
import { orderFulfillment, shippingOptionPriceSet, locationFulfillmentSet, locationFulfillmentProvider } from "./shared.mjs"

const serviceZone = (id, at, n, name, setId = "fuset_1") => model(id, at, "ServiceZone", `id = serzo_${n}`, `name = “${name}”`, `fulfillment_set_id = ${setId}`)

export const pages = {
  "resources/commerce-modules/fulfillment/concepts": [
    {
      nodes: [
        model("fs", [0, 0.5], "FulfillmentSet", "id = fuset_1"),
        serviceZone("s1", [1, 0], 1, "US"),
        model("g1", [2, 0], "GeoZone", "id = fgz_1", "type = “country”", "country_code = “us”", "service_zone_id = serzo_1"),
        serviceZone("s2", [1, 1], 2, "Canada"),
        model("g2", [2, 1], "GeoZone", "id = fgz_2", "type = “country”", "country_code = “ca”", "service_zone_id = serzo_2"),
      ],
      edges: [edge("s1", "fs", "", { fromSide: "l", toSide: "t" }), edge("g1", "s1"), edge("s2", "fs", "", { fromSide: "l", toSide: "b" }), edge("g2", "s2")],
    },
  ],
  "resources/commerce-modules/fulfillment/fulfillment-provider": [providerFlow("Fulfillment Module Provider", "3rd Party Service"), providerModel("Fulfillment", "FulfillmentProvider", "fp_")],
  "resources/commerce-modules/fulfillment/item-fulfillment": [
    {
      nodes: [
        model("p", [0, 0.5], "FulfillmentProvider", "id = webshipper"),
        model("f", [1, 0], "Fulfillment", "id = ful_1", "provider_id = webshipper", "shipping_option_id = so_1"),
        model("s", [1, 1], "ShippingOption", "id = so_1", "provider_id = webshipper"),
      ],
      edges: [edge("f", "p", "", { fromSide: "l", toSide: "t" }), edge("f", "s", "", { fromSide: "b", toSide: "t" }), edge("s", "p", "", { fromSide: "l", toSide: "b" })],
    },
    parentWithTwo(["Fulfillment", "id = ful_1"], ["FulfillmentItem", "id = fulit_1", "quantity = 2", "fulfillment_id = ful_1"], ["FulfillmentItem", "id = fulit_2", "quantity = 1", "fulfillment_id = ful_1"]),
  ],
  "resources/commerce-modules/fulfillment/links-to-other-modules": [
    orderFulfillment(),
    linkDiagram({
      left: "Order",
      right: "Fulfillment",
      table: "return_fulfillment",
      rows: [{ left: ["Return", "id = return_1"], right: ["Fulfillment", "id = ful_1"], fields: ["return_id = return_1", "fulfillment_id = ful_1"] }],
    }),
    shippingOptionPriceSet(),
    locationFulfillmentSet(),
    locationFulfillmentProvider(),
  ],
  "resources/commerce-modules/fulfillment/shipping-option": [
    {
      nodes: [
        model("p", [0, 0.5], "FulfillmentProvider", "id = webshipper"),
        model("o1", [1, 0], "ShippingOption", "id = so_1", "name = Express Shipping", "service_zone_id = serzo_1", "provider_id = webshipper"),
        model("z1", [2, 0], "ServiceZone", "id = serzo_1", "name = “US”"),
        model("o2", [1, 1], "ShippingOption", "id = so_2", "name = Express Shipping", "provider_id = webshipper", "service_zone_id = serzo_2"),
        model("z2", [2, 1], "ServiceZone", "id = serzo_2", "name = “Canada”"),
      ],
      edges: [edge("o1", "p", "", { fromSide: "l", toSide: "t" }), edge("o2", "p", "", { fromSide: "l", toSide: "b" }), edge("o1", "z1"), edge("o2", "z2")],
    },
    {
      nodes: [
        model("o", [0, 0], "ShippingOption", "id = so_1", "name = Express Shipping", "service_zone_id = serzo_1", "provider_id = webshipper"),
        model("z", [1, 0], "ServiceZone", "id = serzo_1"),
        model("g", [2, 0], "GeoZone", "id = fgz_1", "city = “New York”", "country_code = “us”", "service_zone_id = serzo_1"),
      ],
      edges: [edge("o", "z"), edge("g", "z")],
    },
    {
      nodes: [
        model("o", [0, 0], "ShippingOption", "id = so_1", "name = Express Shipping"),
        model("r", [1, 0], "ShippingOptionRule", "id = sorul_1", "attribute = customer_group", "operator = eq", "value = vip", "shipping_option_id = so_1"),
      ],
      edges: [edge("r", "o")],
    },
    parentWithTwo(
      ["ShippingOption", "id = so_1", "name = Express Shipping"],
      ["ShippingOptionRule", "id = sorul_1", "attribute = customer_group", "operator = eq", "value = vip", "shipping_option_id = so_1"],
      ["ShippingOptionRule", "id = sorul_2", "attribute = total_weight", "operator = lt", "value = 2000", "shipping_option_id = so_1"],
    ),
  ],
}

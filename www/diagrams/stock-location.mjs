import { locationFulfillmentSet, locationFulfillmentProvider, salesChannelStockLocation } from "./shared.mjs"

export const pages = {
  "resources/commerce-modules/stock-location/links-to-other-modules": [locationFulfillmentSet(), locationFulfillmentProvider(), salesChannelStockLocation()],
}

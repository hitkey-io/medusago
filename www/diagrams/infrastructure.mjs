import { start, step, model, edge, sequence } from "./lib.mjs"

// Pictures of the infrastructure module pages (resources/infrastructure-modules).
const cachingOverview = (provider, service) => ({
  nodes: [
    step("req", [1, 0], "GET /store/products"),
    model("app", [1, 1], "MedusaGo App", "Caching Module", `↳ ${provider} Provider`),
    start("svc", [1, 2], service),
  ],
  edges: [edge("req", "app", "", { dir: "<->" }), edge("app", "svc", "", { dir: "<->" })],
  gap: [70, 60],
})

export const pages = {
  "resources/infrastructure-modules/analytics": [
    sequence(
      "Analytics Module: tracking an event",
      ["Storefront", "MedusaGo Server", "PostHog"],
      [
        [0, 1, "Place order"],
        [1, 1, "Trigger order.placed event"],
        [1, 0, "Return order"],
        [1, 2, "Track event in PostHog"],
      ],
    ),
  ],
  "resources/infrastructure-modules/caching": [cachingOverview("Redis", "Redis")],
  "resources/infrastructure-modules/caching/guides/memcached": [cachingOverview("Memcached", "Memcached")],
  "resources/infrastructure-modules/caching/migrate-cache": [
    {
      nodes: [
        model("b", [0, 0], "MedusaGo App", "Redis Cache Module"),
        start("br", [1, 0], "Redis"),
        model("a", [0, 2], "MedusaGo App", "Caching Module", "↳ Redis Caching Module Provider", "↳ Memcached Caching Module Provider"),
        start("ar", [1, 1.6], "Redis"),
        start("am", [1, 2.4], "Memcached"),
      ],
      edges: [edge("b", "br", "", { dir: "<->" }), edge("a", "ar", "", { dir: "<->" }), edge("a", "am", "", { dir: "<->" })],
      groups: [
        { label: "Before", nodes: ["b", "br"], dashed: true },
        { label: "After", nodes: ["a", "ar", "am"], dashed: true },
      ],
      gap: [90, 50],
    },
  ],
}

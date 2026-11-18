import { step, edge } from "./lib.mjs"

// Pictures of the admin development chapters (learn/fundamentals/admin).
export const pages = {
  "learn/fundamentals/admin/routing": [
    {
      title: "A UI route with and without a route loader",
      nodes: [
        step("w1", [0, 0], "Show UI route"),
        step("w2", [0, 1], "Fetch brand details"),
        step("w3", [1, 1], "Display brand details", "once they are fetched"),
        step("r1", [0, 3], "Fetch brand details"),
        step("r2", [1, 3], "Show UI route", "with brand details"),
      ],
      edges: [edge("w2", "w3"), edge("r1", "r2")],
      groups: [
        { label: "Without route loader", nodes: ["w1", "w2", "w3"], dashed: true },
        { label: "With route loader", nodes: ["r1", "r2"], dashed: true },
      ],
    },
  ],
}

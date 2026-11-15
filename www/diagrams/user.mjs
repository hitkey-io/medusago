import { edge, start, step } from "./lib.mjs"

export const pages = {
  "resources/commerce-modules/user/invite-user-subscriber": [
    {
      title: "Invite User Flow",
      nodes: [
        start("a", [0, 0], "Admin invites user"),
        step("b", [1, 0], "Create invite"),
        step("c", [2, 0], "Emit invite.created event"),
        step("d", [2, 1], "Execute subscriber"),
        step("e", [1, 1], "Send invite email"),
        start("f", [0, 1], "New user accepts invite"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d"), edge("d", "e"), edge("e", "f")],
    },
  ],
}

import { start, io, decision, step, edge, sequence } from "./lib.mjs"

export const pages = {
  "resources/commerce-modules/auth/auth-flows": [
    {
      title: "Basic Authentication Flow",
      nodes: [
        start("a", [0, 0], "Register user"),
        io("b", [1, 0], "Use the Auth Module’s main", "service’s register method"),
        decision("c", [2, 0], "is success = true?"),
        start("d", [3, 0], "Fail Registration"),
        start("e", [2, 1], "User Registered Successfully"),
        start("f", [0, 2], "Authenticate user"),
        io("g", [1, 2], "Use the Auth Module’s main", "service’s authenticate method"),
        decision("h", [2, 2], "is success = true?"),
        start("i", [3, 2], "Fail Authentication"),
        start("j", [2, 3], "User Authenticated Successfully"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d", "No"), edge("c", "e", "Yes"), edge("f", "g"), edge("g", "h"), edge("h", "i", "No"), edge("h", "j", "Yes")],
    },
    {
      title: "Third-Party Authentication Flow - Before Callback",
      nodes: [
        start("a", [0, 0], "Authenticate user"),
        io("b", [1, 0], "Use the Auth Module’s main", "service’s authenticate", "method"),
        decision("c", [2, 0], "is location defined?"),
        start("d", [3, 0], "Return location to frontend"),
        decision("e", [2, 1], "is success = true?"),
        start("f", [0, 1], "Authentication successful"),
        start("g", [3, 1], "Authentication failed"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d", "Yes"), edge("c", "e", "No"), edge("e", "f", "Yes"), edge("e", "g", "No")],
    },
    {
      title: "Third-Party Authentication Flow - Validate Callback",
      nodes: [
        start("a", [0, 0], "Redirect after user", "logs-in with", "third-party service"),
        io("b", [1, 0], "Use the Auth Module’s main", "service’s validateCallback", "method"),
        decision("c", [2, 0], "is success = true?"),
        start("d", [3, 0], "Fail Authentication"),
        start("e", [2, 1], "User Authenticated Successfully"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d", "No"), edge("c", "e", "Yes")],
    },
  ],
  "resources/commerce-modules/auth/authentication-route": [
    sequence("Basic Authentication Flow Using Routes", ["Frontend", "MedusaGo"], [
      [0, 1, "Register user with /auth/{actor_type}/{provider}/register"],
      [1, 0, "Return registration’s authentication token"],
      [0, 1, "Create user using respective route with token in header"],
      [1, 0, "Return created user"],
      [0, 1, "Authenticate user with /auth/{actor_type}/{provider}"],
    ]),
    sequence(
      "Third-Party Authentication Flow Using Routes",
      ["Frontend", "MedusaGo", "Third-Party Service"],
      [
        [0, 1, "Authenticate user with /auth/{actor_type}/{provider}"],
        [1, 0, "Return authentication location"],
        [0, 2, "Redirect to received location"],
        [2, 0, "After authentication, redirect back to frontend with received\nquery parameters, such as `code` and `state`."],
        [0, 1, "/auth/{actor_type}/{provider}/callback?code=123&state=456"],
        [1, 2, "Validate callback with third-party service"],
        [1, 0, "Return authentication token"],
        [0, 1, "Create user using respective route with token in header"],
        [1, 0, "Return created user"],
        [0, 1, "Refresh token (/auth/token/refresh)"],
        [1, 0, "Return refreshed authentication token"],
      ],
      { frames: [{ label: "If token doesn’t have actor_id", steps: [7, 10], actors: [0, 1] }] },
    ),
  ],
  "resources/commerce-modules/auth/reset-password": [
    {
      title: "Reset Password Flow",
      nodes: [
        start("a", [0, 0], "User requests to reset password"),
        step("b", [2, 0], "Emit auth.password_reset event"),
        step("c", [2, 1], "Execute subscriber"),
        step("d", [1, 1], "Send reset password email"),
        start("e", [0, 1], "User resets password"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d"), edge("d", "e")],
    },
  ],
}

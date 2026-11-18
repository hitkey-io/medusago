import { start, step, io, decision, edge, model, sequence } from "./lib.mjs"

// The checkout, customer and login flows of the storefront guides (resources/storefront-development).
export const pages = {
  "resources/storefront-development/checkout/payment": [
    {
      title: "Payment Step in Storefront",
      nodes: [
        start("a", [0, 0], "Start Payment Step"),
        step("b", [1, 0], "Retrieve available", "payment providers"),
        step("c", [2, 0], "Customer chooses", "payment provider"),
        step("d", [1, 1], "Create payment collection", "for cart"),
        decision("e", [2, 1], "Cart has", "payment collections?"),
        start("f", [0, 2], "Show UI/actions to complete", "payment with provider"),
        step("g", [2, 2], "Initialize payment session", "for selected provider"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "e"), edge("e", "d", "No"), edge("e", "g", "Yes"), edge("d", "g"), edge("g", "f")],
    },
  ],
  "resources/storefront-development/checkout/shipping": [
    {
      title: "Shipping Step in Storefront",
      nodes: [
        start("a", [0, 0], "Start shipping step", "in checkout"),
        step("b", [1, 0], "Retrieve cart’s", "shipping options"),
        step("c", [2, 0], "Loop over shipping options"),
        io("d", [0, 1], "Wait for customer to", "choose an option"),
        step("e", [1, 1], "Show shipping", "option"),
        decision("f", [2, 1], "Is price type", "`calculated`?"),
        start("g", [0, 2], "Add shipping method", "to the cart"),
        step("h", [2, 2], "Retrieve calculated", "price of option"),
      ],
      edges: [
        edge("a", "b"),
        edge("b", "c"),
        edge("c", "f"),
        edge("f", "e", "No"),
        edge("f", "h", "Yes"),
        edge("h", "e"),
        edge("e", "d"),
        edge("d", "g"),
      ],
    },
  ],
  "resources/storefront-development/customers/register": [
    {
      title: "Register Customer Flow",
      nodes: [
        start("a", [0, 0], "Show registration", "form"),
        io("b", [1, 0], "Wait for customer", "to submit"),
        step("c", [2, 0], "Retrieve registration", "JWT token"),
        decision("d", [2, 1], "Request failed?"),
        start("e", [1, 1], "Create customer"),
        step("f", [2, 2], "Try to login"),
        decision("g", [1, 2], "Request failed?"),
        start("h", [0, 2], "Show error"),
      ],
      edges: [
        edge("a", "b"),
        edge("b", "c"),
        edge("c", "d"),
        edge("d", "e", "No"),
        edge("d", "f", "Yes"),
        edge("f", "g"),
        edge("g", "h", "Yes"),
        edge("g", "e", "No"),
      ],
    },
  ],
  // pictures of the page in order: summary flow, the Google console (a screenshot of a third-party UI, not redrawn), the sequence
  "resources/storefront-development/customers/third-party-login": [
    {
      title: "Third-Party Login in Storefront",
      nodes: [
        start("a", [0, 0], "Login Page"),
        step("b", [1, 0], "Click on", "“Login with Google”"),
        step("c", [2, 0], "Redirect to", "Google Authentication Page"),
        step("d", [2, 1], "Redirect to", "callback page"),
        decision("e", [2, 2], "Authentication", "Successful?"),
        start("f", [1, 3], "Redirect to homepage"),
        start("g", [3, 3], "Show error message"),
      ],
      edges: [edge("a", "b"), edge("b", "c"), edge("c", "d"), edge("d", "e"), edge("e", "f", "Yes"), edge("e", "g", "No")],
    },
    null,
    sequence(
      "Third-Party Authentication Flow in the Storefront",
      ["Storefront", "MedusaGo", "Third-Party Service"],
      [
        [0, 1, "Authenticate customer with /auth/customer/{provider}"],
        [1, 0, "Return authentication location"],
        [0, 2, "Redirect to received location"],
        [2, 0, "After authentication, redirect back to the storefront with received\nquery parameters, such as `code` and `state`."],
        [0, 1, "/auth/customer/{provider}/callback?code=123&state=456"],
        [1, 2, "Validate callback with third-party service"],
        [1, 0, "Return authentication token"],
        [0, 1, "Create customer (/store/customers) passing token in the header"],
        [1, 0, "Return created customer"],
        [0, 1, "Refresh token (/auth/token/refresh)"],
        [1, 0, "Return refreshed authentication token"],
      ],
      { frames: [{ label: "If token doesn’t have actor_id", steps: [7, 10], actors: [0, 1] }] },
    ),
  ],
}

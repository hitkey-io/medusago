import { model, edge } from "./lib.mjs"
import { cartPromotion, orderPromotion } from "./shared.mjs"

const ruleWithValue = (table, fields, parent) => ({
  nodes: [
    model("promo", [0, 0], ...parent.promo),
    model("am", [0, 1], ...parent.am),
    model("t", [1, 1], table, ...fields),
    model("r", [2, 1], "PromotionRule", "id = prorul_1", "attribute = SKU", "operator = eq"),
    model("v", [2, 2], "PromotionRuleValue", "id = prorulval_1", "promotion_rule_id = prorul_1", "value = SHIRT"),
  ],
  edges: [edge("am", "promo", "", { fromSide: "t", toSide: "b" }), edge("t", "am"), edge("t", "r"), edge("v", "r", "", { fromSide: "t", toSide: "b" })],
})

const campaignPromos = () => [
  model("p1", [0, 2], "Promotion", "id = promo_1", "campaign_id = procamp_1", "code = 10%OFF"),
  model("p2", [2, 2], "Promotion", "id = promo_2", "campaign_id = procamp_1", "code = FIRSTORDER"),
]

export const pages = {
  "resources/commerce-modules/promotion/application-method": [
    ruleWithValue("application_method_target_rules", ["application_method_id = proappmet_1", "promotion_rule_id = prorul_1"], {
      promo: ["Promotion", "id = promo_1", "type = standard"],
      am: ["ApplicationMethod", "id = proappmet_1", "promotion_id = promo_1", "target_type = items"],
    }),
    ruleWithValue("application_method_buy_rules", ["application_method_id = proappmet_1", "promotion_rule_id = prorul_1"], {
      promo: ["Promotion", "id = promo_1", "type = buyget"],
      am: ["ApplicationMethod", "id = proappmet_1", "promotion_id = promo_1", "buy_rules_min_quantity = 2"],
    }),
  ],
  "resources/commerce-modules/promotion/campaign": [
    {
      nodes: [model("c", [1, 0], "Campaign", "id = procamp_1", "start_date = 2024-05-02", "end_date = 2024-07-20"), ...campaignPromos().map((n) => ({ ...n, at: [n.at[0], 1] }))],
      edges: [edge("p1", "c", "", { fromSide: "t", toSide: "l" }), edge("p2", "c", "", { fromSide: "t", toSide: "r" })],
    },
    {
      nodes: [
        model("b", [0, 0], "CampaignBudget", "id = probudg_1", "campaign_id = procamp_1", "type = usage", "limit = 10"),
        model("c", [1, 1], "Campaign", "id = procamp_1", "start_date = 2024-05-02", "end_date = 2024-07-20"),
        ...campaignPromos(),
      ],
      edges: [edge("b", "c", "", { fromSide: "r", toSide: "t" }), edge("p1", "c", "", { fromSide: "t", toSide: "l" }), edge("p2", "c", "", { fromSide: "t", toSide: "r" })],
    },
    {
      nodes: [
        model("u1", [0, 0], "CampaignBudgetUsage", "id = probudgus_1", "attribute_value = cus_123", "used = 1", "budget_id = probudg_1"),
        model("u2", [1, 0], "CampaignBudgetUsage", "id = probudgus_2", "attribute_value = cus_456", "used = 5", "budget_id = probudg_1"),
        model("b", [0.5, 1], "CampaignBudget", "id = probudg_1", "type = use_by_attribute", "attribute = customer_id", "limit = 5", "campaign_id = procamp_1"),
        model("c", [2, 1], "Campaign", "id = procamp_1"),
        model("p", [2, 2], "Promotion", "id = promo_1", "campaign_id = procamp_1"),
      ],
      edges: [edge("u1", "b", "", { fromSide: "b", toSide: "t" }), edge("u2", "b", "", { fromSide: "b", toSide: "t" }), edge("b", "c"), edge("p", "c", "", { fromSide: "t", toSide: "b" })],
    },
  ],
  "resources/commerce-modules/promotion/concepts": [
    {
      nodes: [
        model("p", [0, 0], "Promotion", "id = promo_1"),
        model("t", [1, 0], "promotion_promotion_rule", "promotion_id: promo_1", "promotion_rule_id: prorul_1"),
        model("r", [2, 0], "PromotionRule", "id = prorul_1", "attribute = customer_group_id", "operator = eq"),
        model("v", [2, 1], "PromotionRuleValue", "id = prorulval_1", "promotion_rule_id = prorul_1", "value = VIP"),
      ],
      edges: [edge("t", "p"), edge("t", "r"), edge("v", "r", "", { fromSide: "t", toSide: "b" })],
    },
    {
      nodes: [
        model("p", [0, 0], "Promotion", "id = promo_1"),
        model("t", [1, 0], "promotion_promotion_rule", "promotion_id: promo_1", "promotion_rule_id: prorul_1"),
        model("r", [1, 1], "PromotionRule", "id = prorul_1", "attribute = customer_group_id", "operator = in"),
        model("v1", [0, 2], "PromotionRuleValue", "id = prorulval_1", "promotion_rule_id = prorul_1", "value = VIP"),
        model("v2", [2, 2], "PromotionRuleValue", "id = prorulval_2", "promotion_rule_id = prorul_1", "value = B2B"),
      ],
      edges: [edge("t", "p"), edge("t", "r", "", { fromSide: "b", toSide: "t" }), edge("v1", "r", "", { fromSide: "t", toSide: "l" }), edge("v2", "r", "", { fromSide: "t", toSide: "r" })],
    },
  ],
  "resources/commerce-modules/promotion/links-to-other-modules": [cartPromotion(), orderPromotion()],
}

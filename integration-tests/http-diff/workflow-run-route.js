// Test-only workflow runs: POST /workflow-run runs one of the workflows
// below (a core workflow as a step followed by a failing step, so that the
// core workflow is compensated after it finished) with throwOnError false
// and answers the final state of the transaction and its first error.
const { MedusaError } = require("@medusajs/framework/utils")
const {
  createStep,
  createWorkflow,
  WorkflowResponse,
} = require("@medusajs/framework/workflows-sdk")
const {
  completeCartWorkflow,
  createOrderCreditLinesWorkflow,
  refreshCartItemsWorkflow,
} = require("@medusajs/medusa/core-flows")

const failStep = createStep("http-diff-fail", async (input) => {
  throw new MedusaError(
    MedusaError.Types.NOT_ALLOWED,
    `Forced failure after ${input.after}`
  )
})

const workflows = {
  "http-diff-complete-cart-then-fail": createWorkflow(
    "http-diff-complete-cart-then-fail",
    function (input) {
      completeCartWorkflow.runAsStep({ input: { id: input.id } })
      failStep({ after: "complete-cart" })
      return new WorkflowResponse(void 0)
    }
  ),
  "http-diff-credit-lines-then-fail": createWorkflow(
    "http-diff-credit-lines-then-fail",
    function (input) {
      createOrderCreditLinesWorkflow.runAsStep({
        input: { id: input.id, credit_lines: input.credit_lines },
      })
      failStep({ after: "create-order-credit-lines" })
      return new WorkflowResponse(void 0)
    }
  ),
  "http-diff-refresh-cart-then-fail": createWorkflow(
    "http-diff-refresh-cart-then-fail",
    function (input) {
      refreshCartItemsWorkflow.runAsStep({ input: { cart_id: input.cart_id } })
      failStep({ after: "refresh-cart-items" })
      return new WorkflowResponse(void 0)
    }
  ),
}

exports.POST = async (req, res) => {
  const { workflow_id, input } = req.body
  const workflow = workflows[workflow_id]
  if (!workflow) {
    return res.status(400).json({ error: `unknown workflow ${workflow_id}` })
  }
  const { errors, transaction } = await workflow(req.scope).run({
    input,
    throwOnError: false,
  })
  const error = errors?.[0]?.error
  return res.json({
    state: transaction.getFlow().state,
    error: error ? { message: error.message, type: error.type ?? null } : null,
  })
}

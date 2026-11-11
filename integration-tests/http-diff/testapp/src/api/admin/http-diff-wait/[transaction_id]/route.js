// http-diff: completes the waiting step of an http-diff-wait workflow
// through the workflow engine ({ workflow_id, action: "success" |
// "failure", response }). Mirrored by
// integration-tests/http-diff/redis-workflows_test.go.
const { Modules } = require("@medusajs/framework/utils")

exports.POST = async (req, res) => {
  const engine = req.scope.resolve(Modules.WORKFLOW_ENGINE)
  const idempotencyKey = {
    action: "invoke",
    transactionId: req.params.transaction_id,
    stepId: "http-diff-wait-step",
    workflowId: req.body.workflow_id ?? "http-diff-wait",
  }
  try {
    if (req.body.action === "failure") {
      await engine.setStepFailure({
        idempotencyKey,
        stepResponse: new Error(String(req.body.response)),
        options: { throwOnError: false },
      })
    } else {
      await engine.setStepSuccess({
        idempotencyKey,
        stepResponse: req.body.response,
        options: { throwOnError: false },
      })
    }
    res.json({ ok: true })
  } catch (e) {
    res.status(400).json({ type: e.type ?? null, message: e.message })
  }
}

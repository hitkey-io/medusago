// Test-only workflow engine operations: POST /workflow-engine-ops runs
// workflowEngine.getRunningTransaction / cancel / retryStep and answers a
// summary of the transaction flow (state and the state of every step).
const { Modules } = require("@medusajs/framework/utils")

function summary(transaction) {
  if (!transaction) {
    return null
  }
  const flow = transaction.getFlow()
  const steps = {}
  for (const id of Object.keys(flow.steps ?? {}).sort()) {
    const step = flow.steps[id]
    if (id === "_root") continue
    steps[id] = {
      invoke: step.invoke?.state,
      invoke_status: step.invoke?.status,
      compensate: step.compensate?.state,
    }
  }
  return {
    state: flow.state,
    cancelled: !!flow.cancelledAt,
    has_failed_steps: !!flow.hasFailedSteps,
    steps,
  }
}

exports.POST = async (req, res) => {
  const engine = req.scope.resolve(Modules.WORKFLOW_ENGINE)
  const { op, workflow_id, transaction_id, step_id, throw_on_error } = req.body
  try {
    if (op === "get") {
      const transaction = await engine.getRunningTransaction(workflow_id, transaction_id)
      return res.json({ transaction: summary(transaction) })
    }
    if (op === "cancel") {
      const ret = await engine.cancel(workflow_id, {
        transactionId: transaction_id,
        throwOnError: throw_on_error,
      })
      return res.json({ acknowledgement: ret.acknowledgement, transaction: summary(ret.transaction) })
    }
    if (op === "retry") {
      const ret = await engine.retryStep({
        idempotencyKey: { action: "invoke", transactionId: transaction_id, stepId: step_id, workflowId: workflow_id },
        options: { throwOnError: throw_on_error },
      })
      return res.json({ transaction: summary(ret.transaction) })
    }
    return res.status(400).json({ error: `unknown op ${op}` })
  } catch (e) {
    return res.status(400).json({ error: e.message, type: e.type ?? null })
  }
}

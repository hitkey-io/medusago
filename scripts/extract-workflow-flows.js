// Dumps the step graphs upstream composes for the registered workflows: the
// flow definition WorkflowManager holds for every workflow (the steps nested
// by "next", a list for parallel branches) and the workflow options.
//
// Run from the reference application with `medusa exec`:
//   medusa exec <this file> <out.json>
// then `node scripts/extract-workflow-flows.mjs <out.json>` generates Go code.

const fs = require("fs")
const { createRequire } = require("module")

exports.default = async function extractWorkflowFlows({ args }) {
  // Resolve the packages of the application being executed.
  const req = createRequire(process.cwd() + "/package.json")
  // Loading core-flows registers its workflows.
  req("@medusajs/medusa/core-flows")
  const { WorkflowManager } = req("@medusajs/framework/orchestration")
  const out = {}
  for (const [id, workflow] of WorkflowManager.getWorkflows()) {
    out[id] = { flow: workflow.flow_, options: workflow.options ?? {} }
  }
  fs.writeFileSync(args[0], JSON.stringify(out))
}

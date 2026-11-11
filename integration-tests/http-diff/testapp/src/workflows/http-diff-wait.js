// http-diff: stored workflows waiting for an async step (completed by
// POST /admin/http-diff-wait/:transaction_id) between two steps creating
// product tags; the first one is compensated (its tag soft deleted). The
// step timeout is 60 s (http-diff-wait) or 2 s (http-diff-wait-timeout).
// Mirrored by integration-tests/http-diff/redis-workflows_test.go.
const {
  createStep,
  createWorkflow,
  StepResponse,
  WorkflowResponse,
} = require("@medusajs/framework/workflows-sdk")

const beforeStep = createStep(
  "http-diff-wait-before",
  async (input, { container }) => {
    const [tag] = await container
      .resolve("product")
      .createProductTags([{ value: `${input.value}-before` }])
    return new StepResponse({ id: tag.id, value: tag.value }, tag.id)
  },
  async (id, { container }) => {
    if (!id) {
      return
    }
    await container.resolve("product").softDeleteProductTags([id])
  }
)

const waitStep = createStep(
  { name: "http-diff-wait-step", async: true, timeout: 60 },
  async () => {}
)

const shortWaitStep = createStep(
  { name: "http-diff-wait-step", async: true, timeout: 2 },
  async () => {}
)

const afterStep = createStep(
  "http-diff-wait-after",
  async (input, { container }) => {
    const [tag] = await container.resolve("product").createProductTags([
      {
        value: `${input.before.value}-after`,
        metadata: { response: input.response ?? null },
      },
    ])
    return new StepResponse({ id: tag.id, value: tag.value })
  }
)

const build = (name, wait) =>
  createWorkflow({ name, store: true, retentionTime: 3600 }, (input) => {
    const before = beforeStep(input)
    const response = wait()
    const after = afterStep({ before, response })
    return new WorkflowResponse({ before, after })
  })

exports.httpDiffWait = build("http-diff-wait", waitStep)
exports.httpDiffWaitTimeout = build("http-diff-wait-timeout", shortWaitStep)

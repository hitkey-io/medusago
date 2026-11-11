// Node.js side of the bullmq interop tests (bullmq from the upstream
// checkout): node interop.js <reference dir> <redis url> <action> <json>
//
// actions:
//   add      { queue, prefix, jobs: [{ name, data, opts }] } -> job ids
//   repeat   { queue, prefix, name, data, opts }             -> job id
//   consume  { queue, prefix, count, fail, updateData }      -> processed jobs
const { createRequire } = require("module")
const path = require("path")

const [reference, redisUrl, action, raw] = process.argv.slice(2)
const req = createRequire(path.join(reference, "node_modules/bullmq/package.json"))
const { Queue, Worker } = req("bullmq")
const Redis = req("ioredis")
const input = JSON.parse(raw)

async function main() {
  const connection = new Redis(redisUrl, { maxRetriesPerRequest: null })
  const out = {}
  if (action === "add") {
    const queue = new Queue(input.queue, { prefix: input.prefix, connection })
    const jobs = await queue.addBulk(input.jobs)
    out.ids = jobs.map((j) => j.id)
    await queue.close()
  } else if (action === "repeat") {
    const queue = new Queue(input.queue, { prefix: input.prefix, connection })
    const job = await queue.add(input.name, input.data, input.opts)
    out.id = job && job.id
    out.repeatable = await queue.getRepeatableJobs()
    await queue.close()
  } else if (action === "consume") {
    const processed = []
    await new Promise((resolve) => {
      const worker = new Worker(
        input.queue,
        async (job) => {
          processed.push({ id: job.id, name: job.name, data: job.data, attemptsMade: job.attemptsMade, opts: job.opts })
          if (input.updateData) {
            await job.updateData({ ...job.data, completedSubscriberIds: ["s1"] })
          }
          if (input.fail && job.attemptsMade === 0) {
            throw new Error("fail once")
          }
          if (processed.length >= input.count) {
            setTimeout(() => worker.close().then(resolve), 50)
          }
          return { ok: true }
        },
        { prefix: input.prefix, connection: new Redis(redisUrl, { maxRetriesPerRequest: null }) }
      )
    })
    out.processed = processed
  }
  connection.disconnect()
  process.stdout.write(JSON.stringify(out), () => process.exit(0))
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})

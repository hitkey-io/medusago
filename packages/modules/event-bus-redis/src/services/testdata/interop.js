// Node.js side of the event bus interop tests: the upstream
// RedisEventBusService (built @medusajs/event-bus-redis of the reference
// checkout). node interop.js <reference dir> <redis url> <action> <json>
//
//   emit     { messages, release }  emits (grouped ones are released when
//                                    release is set); no worker runs
//   stage    { messages }           emits grouped messages, prints the
//                                    staged list
//   consume  { events, count }      runs a worker with subscribers for the
//                                    events until count events arrived
const path = require("path")
const [reference, redisUrl, action, raw] = process.argv.slice(2)
const Redis = require(path.join(reference, "node_modules/ioredis"))
const RedisEventBusService = require(path.join(
  reference,
  "packages/modules/event-bus-redis/dist/services/event-bus-redis"
)).default
const input = JSON.parse(raw)
const logger = { info() {}, warn() {}, error: console.error, debug() {} }

async function main() {
  const connection = new Redis(redisUrl, { maxRetriesPerRequest: null })
  const service = new RedisEventBusService(
    {
      logger,
      eventBusRedisConnection: connection,
      eventBusRedisQueueName: "events-queue",
      eventBusRedisQueueOptions: {},
      eventBusRedisWorkerOptions: {},
      eventBusRedisJobOptions: {},
    },
    {},
    { worker_mode: action === "consume" ? "shared" : "server" }
  )
  const out = {}
  if (action === "emit" || action === "stage") {
    for (const m of input.messages) {
      service.subscribe(m.name, async () => {})
    }
    await service.emit(input.messages)
    if (action === "stage") {
      const gid = input.messages[0].metadata.eventGroupId
      out.staged = await connection.lrange(`staging:${gid}`, 0, -1)
    }
    if (input.release) {
      const groups = new Set(input.messages.map((m) => m.metadata && m.metadata.eventGroupId).filter(Boolean))
      for (const g of groups) await service.releaseGroupedEvents(g)
    }
  } else if (action === "consume") {
    const received = []
    await new Promise((resolve) => {
      for (const name of input.events) {
        service.subscribe(name, async (event) => {
          received.push({
            name: event.name,
            data: event.data,
            metadataKeys: Object.keys(event.metadata || {}).sort(),
            createdAtIsDate: event.metadata && event.metadata.created_at instanceof Date,
          })
          if (received.length >= input.count) setTimeout(resolve, 100)
        })
      }
      service.__hooks.onApplicationStart()
    })
    out.received = received
    await service.bullWorker_.close()
  }
  await service.queue_.close()
  process.stdout.write(JSON.stringify(out), () => process.exit(0))
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})

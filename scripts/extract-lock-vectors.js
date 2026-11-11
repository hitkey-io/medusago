// Captures the lock keys computed by the upstream locking providers (the
// Postgres advisory lock ids and the Redis key names) and the Lua commands
// of locking-redis, for the Go tests:
//
//   node scripts/extract-lock-vectors.js > packages/modules/locking/src/testdata/upstream-lock-keys.json
//
// Requires reference/medusa built.
const path = require("path")

const providers = path.join(__dirname, "../reference/medusa/packages/modules/providers")
const { PostgresAdvisoryLockProvider } = require(path.join(providers, "locking-postgres/dist/services/advisory-lock.js"))
const { RedisLockingProvider } = require(path.join(providers, "locking-redis/dist/services/redis-lock.js"))

const keys = [
  "",
  "a",
  "cart_01JABCDEF0123456789ABCDEFG",
  "order_01K0000000000000000000000Z",
  "iitem_01HZZZZZZZZZZZZZZZZZZZZZZZ",
  "index-module:create-partitions",
  "migration-scripts-running",
  "product",
  "search:seed:products",
  "äöü ñ",
  "ключ-блокировки",
  "emoji-\u{1F600}-key",
  "x".repeat(300),
]

const hash = PostgresAdvisoryLockProvider.prototype.hashStringToInt
const scripts = {}
const fakeClient = { defineCommand(name, def) { scripts[name] = def } }
const redisDefault = new RedisLockingProvider({ redisClient: fakeClient }, {})
const redisNs = new RedisLockingProvider({ redisClient: fakeClient, prefix: "custom:" }, {})

const out = keys.map((key) => ({
  key,
  advisory_lock_key: hash.call(null, key),
  redis_key: redisDefault.getKeyName(key),
  redis_key_custom_prefix: redisNs.getKeyName(key),
}))
process.stdout.write(JSON.stringify({ keys: out, scripts }, null, 2) + "\n")

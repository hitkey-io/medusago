package modules

import lockingredis "github.com/hitkey-io/medusago/packages/modules/providers/locking-redis/src"

// LockingRedis re-exports @medusajs/locking-redis.
var LockingRedis = lockingredis.Module

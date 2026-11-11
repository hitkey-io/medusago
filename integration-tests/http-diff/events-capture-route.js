// Test-only event capture: an event bus interceptor records every event
// published (after the event group release) in order.
const { Modules } = require("@medusajs/framework/utils")

const captured = []
let installed = false

function install(scope) {
  if (installed) return
  installed = true
  const bus = scope.resolve(Modules.EVENT_BUS)
  bus.addInterceptor((message) => {
    const { created_at, published_at, ...metadata } = message.metadata ?? {}
    captured.push({
      name: message.name,
      data: message.data,
      metadata,
      options: message.options ?? null,
    })
  })
}

// POST /events-capture: start capturing (clears the captured events)
exports.POST = async (req, res) => {
  install(req.scope)
  captured.length = 0
  res.json({ events: [] })
}

// GET /events-capture: the captured events, in publication order
exports.GET = async (req, res) => {
  install(req.scope)
  res.json({ events: captured })
}

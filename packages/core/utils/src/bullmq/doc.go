// Package bullmq is a port of the parts of bullmq (the version upstream
// Medusa depends on, see VERSION) used by the Medusa Redis modules: queues,
// jobs, workers (retries, backoff, delays, stalled jobs, locks) and
// repeatable jobs. It runs bullmq's own Lua commands (commands/, copied from
// the bullmq build by extract-commands.mjs; bullmq is MIT licensed, see
// LICENSE) with the same keys and arguments, so that Go and Node.js
// processes can share the same queues.
package bullmq

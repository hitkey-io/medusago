package bullmq

import (
	"context"
	"embed"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

//go:embed commands/*.lua
var commandFS embed.FS

// command is a bullmq Lua command (ioredis defineCommand with numberOfKeys).
type command struct {
	script  *redis.Script
	numKeys int
}

var commands = loadCommands()

func loadCommands() map[string]command {
	out := map[string]command{}
	entries, err := commandFS.ReadDir("commands")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		base := strings.TrimSuffix(e.Name(), ".lua")
		i := strings.LastIndex(base, "-")
		n, err := strconv.Atoi(base[i+1:])
		if err != nil {
			panic(err)
		}
		src, err := commandFS.ReadFile(path.Join("commands", e.Name()))
		if err != nil {
			panic(err)
		}
		out[base[:i]] = command{script: redis.NewScript(string(src)), numKeys: n}
	}
	return out
}

// splitArgs turns an ioredis-style argument list (keys first) into keys and
// arguments.
func splitArgs(name string, args []any) (command, []string, []any) {
	cmd, ok := commands[name]
	if !ok {
		panic(fmt.Sprintf("bullmq: unknown command %s", name))
	}
	keys := make([]string, cmd.numKeys)
	for i := 0; i < cmd.numKeys; i++ {
		keys[i] = argString(args[i])
	}
	rest := make([]any, 0, len(args)-cmd.numKeys)
	for _, a := range args[cmd.numKeys:] {
		rest = append(rest, redisArg(a))
	}
	return cmd, keys, rest
}

// runCommand runs a command (EVALSHA, falling back to EVAL).
func runCommand(ctx context.Context, c redis.Scripter, name string, args ...any) *redis.Cmd {
	cmd, keys, rest := splitArgs(name, args)
	return cmd.script.Run(ctx, c, keys, rest...)
}

// queueCommand adds a command to a pipeline or transaction (EVAL: a
// pipeline cannot fall back on NOSCRIPT).
func queueCommand(ctx context.Context, p redis.Pipeliner, name string, args ...any) *redis.Cmd {
	cmd, keys, rest := splitArgs(name, args)
	return cmd.script.Eval(ctx, p, keys, rest...)
}

// redisArg mirrors how ioredis sends an argument: null/undefined as an
// empty string, numbers as String(n).
func redisArg(v any) any {
	switch x := v.(type) {
	case nil:
		return ""
	case []byte:
		return x
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return jsString(v)
}

func argString(v any) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return redisArg(v).(string)
}

// replyInt reads an integer reply (Lua numbers are integers in replies).
func replyInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	}
	return 0, false
}

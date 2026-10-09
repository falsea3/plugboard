package redis

import (
	"context"
	"slices"
	"strings"
	"time"
)

type cmdFlags struct {
	known bool
	read  bool
}

var alwaysRead = []string{"ping", "echo", "select", "info", "time", "hello", "multi", "exec", "discard", "watch", "unwatch", "quit", "lastsave", "role"}

var containers = []string{"acl", "client", "cluster", "command", "config", "debug", "function", "latency", "memory", "module", "object", "pubsub", "script", "slowlog", "xinfo"}

var refused = map[string]string{
	"subscribe":  "SUBSCRIBE keeps the connection listening; use a separate client for Pub/Sub",
	"psubscribe": "PSUBSCRIBE keeps the connection listening; use a separate client for Pub/Sub",
	"ssubscribe": "SSUBSCRIBE keeps the connection listening; use a separate client for Pub/Sub",
	"monitor":    "MONITOR streams every command forever; use redis-cli for it",
	"sync":       "SYNC is for replicas",
	"psync":      "PSYNC is for replicas",
}

func commandKey(args []string) string {
	name := strings.ToLower(args[0])
	if len(args) > 1 && slices.Contains(containers, name) {
		return name + "|" + strings.ToLower(args[1])
	}
	return name
}

func readFlags(flags []string) bool {
	return slices.Contains(flags, "readonly") && !slices.Contains(flags, "write")
}

func (s *Session) isRead(ctx context.Context, args []string) bool {
	if len(args) == 0 {
		return true
	}
	if slices.Contains(alwaysRead, strings.ToLower(args[0])) {
		return true
	}
	key := commandKey(args)
	s.flagsMu.Lock()
	f, ok := s.flags[key]
	s.flagsMu.Unlock()
	if ok {
		return f.read
	}
	f = s.lookup(ctx, key)
	if f.known {
		s.flagsMu.Lock()
		s.flags[key] = f
		s.flagsMu.Unlock()
	}
	return f.read
}

func (s *Session) lookup(ctx context.Context, key string) cmdFlags {
	c, err := s.client(0, true)
	if err != nil {
		return cmdFlags{}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	v, err := c.Do(ctx, "COMMAND", "INFO", key).Result()
	if err != nil {
		return cmdFlags{}
	}
	list, _ := v.([]any)
	if len(list) == 0 {
		return cmdFlags{}
	}
	info, _ := list[0].([]any)
	if len(info) < 3 {
		return cmdFlags{known: true}
	}
	return cmdFlags{known: true, read: readFlags(strs(info[2]))}
}

func strs(v any) []string {
	var out []string
	switch v := v.(type) {
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, strings.ToLower(s))
			}
		}
	case map[any]bool:
		for x := range v {
			if s, ok := x.(string); ok {
				out = append(out, strings.ToLower(s))
			}
		}
	}
	return out
}

func (s *Session) WriteStatements(script string) []string {
	cmds, err := parseScript(script)
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out []string
	for _, c := range cmds {
		if !s.isRead(ctx, c.args) {
			out = append(out, c.text)
		}
	}
	return out
}

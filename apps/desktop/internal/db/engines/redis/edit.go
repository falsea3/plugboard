package redis

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var (
	errExists  = apperr.New("exists", "that name is already taken")
	errChanged = apperr.New("row_gone", "the value was changed by someone else — refresh and try again")
	errScore   = apperr.New("bad_value", "a score is a number, like 1.5 or -inf")
)

func (s *Session) EditKey(ctx context.Context, schema string, e model.KeyEdit) error {
	if s.conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "changes"}
	}
	c, err := s.clientFor(schema)
	if err != nil {
		return err
	}
	k, err := untoken(e.Key)
	if err != nil {
		return err
	}
	field, err := untoken(e.Field)
	if err != nil {
		return err
	}
	if kind, ok := strings.CutPrefix(e.Op, "create:"); ok {
		return classifyEdit(create(ctx, c, k, kind, e))
	}
	if n, err := c.Exists(ctx, k).Result(); err != nil {
		return classify(err)
	} else if n == 0 {
		return errKeyGone
	}
	return classifyEdit(apply(ctx, c, k, field, e))
}

func classifyEdit(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(err.Error(), "PLUGBOARD_EXISTS"):
		return errExists
	case strings.Contains(err.Error(), "PLUGBOARD_CHANGED"):
		return errChanged
	}
	return classify(err)
}

func score(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, errScore
	}
	return f, nil
}

func ok(n int64, err error) error {
	if err == nil && n == 0 {
		return errExists
	}
	return err
}

func apply(ctx context.Context, c *goredis.Client, k, field string, e model.KeyEdit) error {
	switch e.Op {
	case "set":
		return c.SetArgs(ctx, k, e.Value, goredis.SetArgs{KeepTTL: true}).Err()
	case "hset":
		return c.HSet(ctx, k, field, e.Value).Err()
	case "hadd":
		set, err := c.HSetNX(ctx, k, field, e.Value).Result()
		if err == nil && !set {
			return errExists
		}
		return err
	case "hdel":
		return c.HDel(ctx, k, field).Err()
	case "hrename":
		return renameField.Run(ctx, c, []string{k}, field, e.Value).Err()
	case "lset":
		return listSet.Run(ctx, c, []string{k}, e.Index, e.Old, e.Value).Err()
	case "lrem":
		return listRemove.Run(ctx, c, []string{k}, e.Index, e.Old, "\x00plugboard:"+strconv.FormatInt(time.Now().UnixNano(), 36)).Err()
	case "rpush":
		return c.RPush(ctx, k, e.Value).Err()
	case "lpush":
		return c.LPush(ctx, k, e.Value).Err()
	case "sadd":
		return ok(c.SAdd(ctx, k, e.Value).Result())
	case "srem":
		return c.SRem(ctx, k, field).Err()
	case "sreplace":
		return replaceMember.Run(ctx, c, []string{k}, field, e.Value).Err()
	case "zadd":
		f, err := score(e.Score)
		if err != nil {
			return err
		}
		return ok(c.ZAddNX(ctx, k, goredis.Z{Score: f, Member: e.Value}).Result())
	case "zscore":
		f, err := score(e.Score)
		if err != nil {
			return err
		}
		return c.ZAddXX(ctx, k, goredis.Z{Score: f, Member: field}).Err()
	case "zrem":
		return c.ZRem(ctx, k, field).Err()
	case "zrename":
		return renameMember.Run(ctx, c, []string{k}, field, e.Value).Err()
	case "xadd":
		values, err := streamFields(e.Value)
		if err != nil {
			return err
		}
		return c.XAdd(ctx, &goredis.XAddArgs{Stream: k, Values: values}).Err()
	case "xdel":
		return c.XDel(ctx, k, field).Err()
	case "expire":
		if e.Index <= 0 {
			return c.Persist(ctx, k).Err()
		}
		return c.Expire(ctx, k, time.Duration(e.Index)*time.Second).Err()
	case "rename":
		if strings.TrimSpace(e.Value) == "" {
			return apperr.New("bad_value", "give the key a name")
		}
		set, err := c.RenameNX(ctx, k, e.Value).Result()
		if err == nil && !set {
			return errExists
		}
		return err
	case "delete":
		err := c.Unlink(ctx, k).Err()
		if err != nil && strings.Contains(err.Error(), "unknown command") {
			err = c.Del(ctx, k).Err()
		}
		return err
	}
	return apperr.New("bad_value", "unknown change: "+e.Op)
}

func create(ctx context.Context, c *goredis.Client, k, kind string, e model.KeyEdit) error {
	if strings.TrimSpace(k) == "" {
		return apperr.New("bad_value", "give the key a name")
	}
	if n, err := c.Exists(ctx, k).Result(); err != nil {
		return err
	} else if n > 0 {
		return errExists
	}
	switch kind {
	case "string":
		set, err := c.SetNX(ctx, k, e.Value, 0).Result()
		if err == nil && !set {
			return errExists
		}
		return err
	case "hash":
		return c.HSet(ctx, k, e.Field, e.Value).Err()
	case "list":
		return c.RPush(ctx, k, e.Value).Err()
	case "set":
		return c.SAdd(ctx, k, e.Value).Err()
	case "zset":
		f, err := score(e.Score)
		if err != nil {
			return err
		}
		return c.ZAdd(ctx, k, goredis.Z{Score: f, Member: e.Value}).Err()
	case "stream":
		values, err := streamFields(e.Value)
		if err != nil {
			return err
		}
		return c.XAdd(ctx, &goredis.XAddArgs{Stream: k, Values: values}).Err()
	}
	return apperr.New("bad_value", "unknown key type: "+kind)
}

func streamFields(text string) ([]string, error) {
	var fields map[string]string
	if err := json.Unmarshal([]byte(text), &fields); err != nil || len(fields) == 0 {
		return nil, apperr.New("bad_value", `a stream entry is a JSON object of text fields, like {"event": "signup"}`)
	}
	out := make([]string, 0, 2*len(fields))
	for f, v := range fields {
		out = append(out, f, v)
	}
	return out, nil
}

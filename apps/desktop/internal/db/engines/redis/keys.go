package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const (
	itemPage = 200
	maxText  = 1 << 20
	maxScans = 50
)

var errKeyGone = apperr.New("key_gone", "this key no longer exists — it may have expired or been deleted")

func (s *Session) ScanKeys(ctx context.Context, schema, pattern, cursor string, count int) (model.KeyPage, error) {
	c, err := s.clientFor(schema)
	if err != nil {
		return model.KeyPage{}, err
	}
	if strings.TrimSpace(pattern) == "" {
		pattern = "*"
	}
	if count <= 0 || count > 5000 {
		count = 500
	}
	at, _ := strconv.ParseUint(cursor, 10, 64)
	var names []string
	for range maxScans {
		batch, next, err := c.Scan(ctx, at, pattern, int64(count)).Result()
		if err != nil {
			return model.KeyPage{}, classify(err)
		}
		names = append(names, batch...)
		at = next
		if at == 0 || len(names) >= count {
			break
		}
	}
	types := make([]*goredis.StatusCmd, len(names))
	_, err = c.Pipelined(ctx, func(p goredis.Pipeliner) error {
		for i, n := range names {
			types[i] = p.Type(ctx, n)
		}
		return nil
	})
	if err != nil {
		return model.KeyPage{}, classify(err)
	}
	page := model.KeyPage{Keys: make([]model.KeyInfo, 0, len(names)), Cursor: strconv.FormatUint(at, 10), Done: at == 0}
	for i, n := range names {
		kind := types[i].Val()
		if kind == "none" {
			continue
		}
		label, _ := display(n)
		page.Keys = append(page.Keys, model.KeyInfo{Name: label, Key: token(n), Type: kind})
	}
	return page, nil
}

func (s *Session) ReadKey(ctx context.Context, schema, key, cursor string) (model.KeyValue, error) {
	c, err := s.clientFor(schema)
	if err != nil {
		return model.KeyValue{}, err
	}
	k, err := untoken(key)
	if err != nil {
		return model.KeyValue{}, err
	}
	kind, err := c.Type(ctx, k).Result()
	if err != nil {
		return model.KeyValue{}, classify(err)
	}
	if kind == "none" {
		return model.KeyValue{}, errKeyGone
	}
	ttl, err := c.PTTL(ctx, k).Result()
	if err != nil {
		return model.KeyValue{}, classify(err)
	}
	v := model.KeyValue{Key: key, Type: kind, TTL: -1, Items: []model.KeyItem{}, Done: true}
	if ttl >= 0 {
		v.TTL = int64(ttl.Seconds())
	}
	switch kind {
	case "string":
		err = readString(ctx, c, k, &v)
	case "hash", "set":
		err = readScan(ctx, c, k, kind, cursor, &v)
	case "list", "zset":
		err = readRange(ctx, c, k, kind, cursor, &v)
	case "stream":
		err = readStream(ctx, c, k, cursor, &v)
	default:
		v.Text = "Values of type " + kind + " can't be shown here; use the console."
		v.Binary = true
	}
	if errors.Is(err, goredis.Nil) {
		return model.KeyValue{}, errKeyGone
	}
	if err != nil {
		return model.KeyValue{}, classify(err)
	}
	return v, nil
}

func readString(ctx context.Context, c *goredis.Client, k string, v *model.KeyValue) error {
	size, err := c.StrLen(ctx, k).Result()
	if err != nil {
		return err
	}
	text, err := c.GetRange(ctx, k, 0, maxText-1).Result()
	if err != nil {
		return err
	}
	v.Size = size
	v.Truncated = size > maxText
	v.Text, v.Binary = display(text)
	return nil
}

func item(field, value string) model.KeyItem {
	label, binField := display(field)
	shown, binValue := display(value)
	it := model.KeyItem{Field: token(field), Value: shown, Binary: binValue}
	if binField {
		it.Label = label
	}
	return it
}

func readScan(ctx context.Context, c *goredis.Client, k, kind, cursor string, v *model.KeyValue) error {
	at, _ := strconv.ParseUint(cursor, 10, 64)
	var (
		batch []string
		next  uint64
		err   error
	)
	if kind == "hash" {
		v.Size, err = c.HLen(ctx, k).Result()
		if err == nil {
			batch, next, err = c.HScan(ctx, k, at, "*", itemPage).Result()
		}
	} else {
		v.Size, err = c.SCard(ctx, k).Result()
		if err == nil {
			batch, next, err = c.SScan(ctx, k, at, "*", itemPage).Result()
		}
	}
	if err != nil {
		return err
	}
	if kind == "hash" {
		for i := 0; i+1 < len(batch); i += 2 {
			v.Items = append(v.Items, item(batch[i], batch[i+1]))
		}
	} else {
		for _, m := range batch {
			it := item(m, m)
			v.Items = append(v.Items, it)
		}
	}
	v.Cursor, v.Done = strconv.FormatUint(next, 10), next == 0
	return nil
}

func readRange(ctx context.Context, c *goredis.Client, k, kind, cursor string, v *model.KeyValue) error {
	from, _ := strconv.ParseInt(cursor, 10, 64)
	to := from + itemPage - 1
	var err error
	if kind == "list" {
		var vals []string
		if v.Size, err = c.LLen(ctx, k).Result(); err == nil {
			vals, err = c.LRange(ctx, k, from, to).Result()
		}
		for i, val := range vals {
			v.Items = append(v.Items, item(strconv.FormatInt(from+int64(i), 10), val))
		}
	} else {
		var zs []goredis.Z
		if v.Size, err = c.ZCard(ctx, k).Result(); err == nil {
			zs, err = c.ZRangeWithScores(ctx, k, from, to).Result()
		}
		for _, z := range zs {
			m, _ := z.Member.(string)
			it := item(m, m)
			it.Score = strconv.FormatFloat(z.Score, 'g', -1, 64)
			v.Items = append(v.Items, it)
		}
	}
	if err != nil {
		return err
	}
	next := from + int64(len(v.Items))
	v.Cursor, v.Done = strconv.FormatInt(next, 10), next >= v.Size
	return nil
}

func readStream(ctx context.Context, c *goredis.Client, k, cursor string, v *model.KeyValue) error {
	var err error
	if v.Size, err = c.XLen(ctx, k).Result(); err != nil {
		return err
	}
	start := "-"
	if cursor != "" {
		start = "(" + cursor
	}
	msgs, err := c.XRangeN(ctx, k, start, "+", itemPage).Result()
	if err != nil {
		return err
	}
	for _, m := range msgs {
		fields := map[string]string{}
		for f, x := range m.Values {
			fields[f], _ = display(textOf(x))
		}
		b, _ := json.Marshal(fields)
		v.Items = append(v.Items, model.KeyItem{Field: m.ID, Value: string(b)})
	}
	if len(msgs) > 0 {
		v.Cursor = msgs[len(msgs)-1].ID
	}
	v.Done = len(msgs) < itemPage
	return nil
}

func textOf(x any) string {
	if s, ok := x.(string); ok {
		return s
	}
	return text(x)
}

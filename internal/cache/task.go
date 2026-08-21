package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/RamDass1/test-api/internal/breaker"
	"github.com/RamDass1/test-api/internal/domain"
)

const (
	breakerThreshold = 5
	breakerCooldown  = 15 * time.Second
	callTimeout      = 300 * time.Millisecond
)

type TaskCache struct {
	rdb     *redis.Client
	ttl     time.Duration
	breaker *breaker.Breaker
	enabled bool
}

func NewTaskCache(rdb *redis.Client, ttl time.Duration) *TaskCache {
	return &TaskCache{
		rdb:     rdb,
		ttl:     ttl,
		breaker: breaker.New(breakerThreshold, breakerCooldown),
		enabled: rdb != nil && ttl > 0,
	}
}

func (c *TaskCache) Get(ctx context.Context, f domain.TaskFilter) (domain.TaskPage, bool) {
	if !c.enabled {
		return domain.TaskPage{}, false
	}

	var raw []byte
	err := c.breaker.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()

		gen, err := c.generation(ctx, f.TeamID)
		if err != nil {
			return err
		}
		raw, err = c.rdb.Get(ctx, listKey(f, gen)).Bytes()
		if errors.Is(err, redis.Nil) {
			raw = nil
			return nil
		}
		return err
	})
	if err != nil {
		c.report("get", err)
		return domain.TaskPage{}, false
	}
	if raw == nil {
		return domain.TaskPage{}, false
	}

	var page domain.TaskPage
	if err := json.Unmarshal(raw, &page); err != nil {
		slog.Warn("discarding a malformed cache entry", slog.String("error", err.Error()))
		return domain.TaskPage{}, false
	}
	return page, true
}

func (c *TaskCache) Set(ctx context.Context, f domain.TaskFilter, page domain.TaskPage) {
	if !c.enabled {
		return
	}

	payload, err := json.Marshal(page)
	if err != nil {
		slog.Warn("cannot encode a task page for the cache", slog.String("error", err.Error()))
		return
	}

	err = c.breaker.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()

		gen, err := c.generation(ctx, f.TeamID)
		if err != nil {
			return err
		}
		return c.rdb.Set(ctx, listKey(f, gen), payload, c.ttl).Err()
	})
	if err != nil {
		c.report("set", err)
	}
}

func (c *TaskCache) Invalidate(ctx context.Context, teamID int64) {
	if !c.enabled {
		return
	}

	err := c.breaker.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		return c.rdb.Incr(ctx, generationKey(teamID)).Err()
	})
	if err != nil {
		c.report("invalidate", err)
	}
}

func (c *TaskCache) generation(ctx context.Context, teamID int64) (int64, error) {
	gen, err := c.rdb.Get(ctx, generationKey(teamID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return gen, err
}

func (c *TaskCache) report(op string, err error) {
	if errors.Is(err, breaker.ErrOpen) {
		return
	}
	slog.Warn("cache unavailable, falling back to mysql",
		slog.String("op", op), slog.String("error", err.Error()))
}

func generationKey(teamID int64) string {
	return "tasks:gen:" + strconv.FormatInt(teamID, 10)
}

func listKey(f domain.TaskFilter, generation int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "tasks:%d:%d:", f.TeamID, generation)
	b.WriteString("status=")
	if f.Status != nil {
		b.WriteString(string(*f.Status))
	}
	b.WriteString(";assignee=")
	if f.AssigneeID != nil {
		b.WriteString(strconv.FormatInt(*f.AssigneeID, 10))
	}
	b.WriteString(";cursor=")
	if f.Cursor != nil {
		b.WriteString(strconv.FormatInt(*f.Cursor, 10))
	}
	fmt.Fprintf(&b, ";limit=%d;offset=%d", f.Limit, f.Offset)
	return b.String()
}

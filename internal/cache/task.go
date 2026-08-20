package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/RamDass1/test-api/internal/domain"
)

const callTimeout = 300 * time.Millisecond

type TaskCache struct {
	rdb     *redis.Client
	ttl     time.Duration
	enabled bool
}

func NewTaskCache(rdb *redis.Client, ttl time.Duration) *TaskCache {
	return &TaskCache{rdb: rdb, ttl: ttl, enabled: rdb != nil && ttl > 0}
}

func (c *TaskCache) Get(ctx context.Context, f domain.TaskFilter) (domain.TaskPage, bool) {
	if !c.enabled {
		return domain.TaskPage{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	gen, err := c.generation(ctx, f.TeamID)
	if err != nil {
		log.Printf("cache get: %v", err)
		return domain.TaskPage{}, false
	}
	raw, err := c.rdb.Get(ctx, listKey(f, gen)).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.TaskPage{}, false
	}
	if err != nil {
		log.Printf("cache get: %v", err)
		return domain.TaskPage{}, false
	}

	var page domain.TaskPage
	if err := json.Unmarshal(raw, &page); err != nil {
		log.Printf("cache get: discard malformed entry: %v", err)
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
		log.Printf("cache set: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	gen, err := c.generation(ctx, f.TeamID)
	if err != nil {
		log.Printf("cache set: %v", err)
		return
	}
	if err := c.rdb.Set(ctx, listKey(f, gen), payload, c.ttl).Err(); err != nil {
		log.Printf("cache set: %v", err)
	}
}

func (c *TaskCache) Invalidate(ctx context.Context, teamID int64) {
	if !c.enabled {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	if err := c.rdb.Incr(ctx, generationKey(teamID)).Err(); err != nil {
		log.Printf("cache invalidate: %v", err)
	}
}

func (c *TaskCache) generation(ctx context.Context, teamID int64) (int64, error) {
	gen, err := c.rdb.Get(ctx, generationKey(teamID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return gen, err
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
	fmt.Fprintf(&b, ";limit=%d;offset=%d", f.Limit, f.Offset)
	return b.String()
}

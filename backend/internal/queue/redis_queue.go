package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	QueueRegister = "grokforge:queue:register"
	QueueProbe    = "grokforge:queue:probe"
	QueueNoop     = "grokforge:queue:noop"
	QueueDefault  = "grokforge:queue:default"
	PubSubEvents  = "grokforge:pubsub:events"
	LockJobPrefix = "grokforge:lock:job:"
)

// Queue is a thin Redis list + pubsub wrapper.
type Queue struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

func (q *Queue) Ping(ctx context.Context) error {
	return q.rdb.Ping(ctx).Err()
}

func QueueNameForKind(kind string) string {
	switch kind {
	case "register":
		return QueueRegister
	case "probe":
		return QueueProbe
	case "noop":
		return QueueNoop
	default:
		return QueueDefault
	}
}

func (q *Queue) Enqueue(ctx context.Context, kind string, jobID int64) error {
	if err := q.Ping(ctx); err != nil {
		return fmt.Errorf("redis unavailable: %w", err)
	}
	key := QueueNameForKind(kind)
	return q.rdb.LPush(ctx, key, strconv.FormatInt(jobID, 10)).Err()
}

// BRPop blocks until a job id is available from any of the queues.
func (q *Queue) BRPop(ctx context.Context, timeout time.Duration, queues ...string) (queue string, jobID int64, err error) {
	if len(queues) == 0 {
		queues = []string{QueueNoop, QueueRegister, QueueProbe, QueueDefault}
	}
	res, err := q.rdb.BRPop(ctx, timeout, queues...).Result()
	if err != nil {
		return "", 0, err
	}
	if len(res) != 2 {
		return "", 0, fmt.Errorf("unexpected brpop result")
	}
	id, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		return res[0], 0, fmt.Errorf("bad job id %q: %w", res[1], err)
	}
	return res[0], id, nil
}

func (q *Queue) TryLock(ctx context.Context, jobID int64, ttl time.Duration) (bool, error) {
	ok, err := q.rdb.SetNX(ctx, LockJobPrefix+strconv.FormatInt(jobID, 10), "1", ttl).Result()
	return ok, err
}

func (q *Queue) Unlock(ctx context.Context, jobID int64) error {
	return q.rdb.Del(ctx, LockJobPrefix+strconv.FormatInt(jobID, 10)).Err()
}

func (q *Queue) Publish(ctx context.Context, payload string) error {
	return q.rdb.Publish(ctx, PubSubEvents, payload).Err()
}

func (q *Queue) Subscribe(ctx context.Context) *redis.PubSub {
	return q.rdb.Subscribe(ctx, PubSubEvents)
}

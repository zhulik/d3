package locker

import (
	"context"

	"github.com/zhulik/d3/internal/core"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidislock"
)

type Locker struct {
	Config *core.Config

	locker rueidislock.Locker
}

func (l *Locker) Init(_ context.Context) error {
	locker, err := rueidislock.NewLocker(rueidislock.LockerOption{
		ClientOption:   rueidis.ClientOption{InitAddress: []string{l.Config.RedisAddress}},
		KeyMajority:    1,
		NoLoopTracking: true,
	})
	if err != nil {
		return err
	}

	l.locker = locker

	return nil
}

func (l *Locker) Shutdown(_ context.Context) error {
	l.locker.Close()

	return nil
}

func (l *Locker) Lock(ctx context.Context, key string) (context.Context, context.CancelFunc, error) {
	return l.locker.WithContext(ctx, key)
}

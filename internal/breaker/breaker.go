package breaker

import (
	"errors"
	"sync"
	"time"
)

var ErrOpen = errors.New("circuit breaker is open")

type Breaker struct {
	threshold int
	cooldown  time.Duration
	now       func() time.Time

	mu        sync.Mutex
	failures  int
	openUntil time.Time
}

func New(threshold int, cooldown time.Duration) *Breaker {
	if threshold < 1 {
		threshold = 1
	}
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now}
}

func (b *Breaker) Do(fn func() error) error {
	if !b.allow() {
		return ErrOpen
	}
	if err := fn(); err != nil {
		b.record(false)
		return err
	}
	b.record(true)
	return nil
}

func (b *Breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.openUntil.IsZero() {
		return true
	}
	if b.now().Before(b.openUntil) {
		return false
	}
	b.openUntil = time.Time{}
	b.failures = b.threshold - 1
	return true
}

func (b *Breaker) record(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		b.failures = 0
		b.openUntil = time.Time{}
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = b.now().Add(b.cooldown)
	}
}

func (b *Breaker) Open() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.openUntil.IsZero() && b.now().Before(b.openUntil)
}

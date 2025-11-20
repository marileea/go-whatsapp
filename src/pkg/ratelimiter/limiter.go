package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	domainAPIKey "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
	"github.com/sirupsen/logrus"
)

const ResourceREST = "rest_api"

type AllowRequest struct {
	TenantID     string
	APIKeyID     string
	ResourceType string
	Limit        int
}

type AllowResult struct {
	Allowed    bool
	RetryAfter time.Duration
	Remaining  int
}

type RateLimiter interface {
	Allow(ctx context.Context, req *AllowRequest) (*AllowResult, error)
	Shutdown(ctx context.Context) error
}

type Limiter struct {
	repo          domainAPIKey.IAPIKeyRepository
	window        time.Duration
	windowSeconds int
	flushInterval time.Duration
	clock         func() time.Time

	mu      sync.RWMutex
	buckets map[string]*bucket

	stopCh    chan struct{}
	stoppedCh chan struct{}
}

type Option func(*Limiter)

func WithWindow(window time.Duration) Option {
	return func(l *Limiter) {
		if window > 0 {
			l.window = window
			l.windowSeconds = int(window.Seconds())
		}
	}
}

func WithFlushInterval(interval time.Duration) Option {
	return func(l *Limiter) {
		if interval > 0 {
			l.flushInterval = interval
		}
	}
}

func WithClock(clock func() time.Time) Option {
	return func(l *Limiter) {
		if clock != nil {
			l.clock = clock
		}
	}
}

func NewLimiter(repo domainAPIKey.IAPIKeyRepository, opts ...Option) *Limiter {
	if repo == nil {
		panic("ratelimiter: repository is required")
	}

	limiter := &Limiter{
		repo:          repo,
		window:        time.Minute,
		windowSeconds: 60,
		flushInterval: 5 * time.Second,
		clock:         time.Now,
		buckets:       make(map[string]*bucket),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(limiter)
	}

	if limiter.windowSeconds <= 0 {
		limiter.window = time.Minute
		limiter.windowSeconds = 60
	}
	if limiter.flushInterval <= 0 {
		limiter.flushInterval = 5 * time.Second
	}

	go limiter.flushLoop()
	return limiter
}

func (l *Limiter) Allow(ctx context.Context, req *AllowRequest) (*AllowResult, error) {
	if req == nil {
		return nil, fmt.Errorf("allow request cannot be nil")
	}
	if req.APIKeyID == "" {
		return nil, fmt.Errorf("api key id is required")
	}
	if req.ResourceType == "" {
		req.ResourceType = ResourceREST
	}
	if req.Limit <= 0 {
		return &AllowResult{Allowed: true, Remaining: -1}, nil
	}

	now := l.clock()
	bucketKey := l.bucketKey(req)

	bucket, err := l.getOrCreateBucket(ctx, bucketKey, req, now)
	if err != nil {
		return nil, err
	}

	allowed, remaining, retryAfter, err := l.consume(ctx, bucket, req, now)
	if err != nil {
		return nil, err
	}

	return &AllowResult{Allowed: allowed, Remaining: remaining, RetryAfter: retryAfter}, nil
}

func (l *Limiter) Shutdown(ctx context.Context) error {
	select {
	case <-l.stopCh:
	default:
		close(l.stopCh)
	}

	select {
	case <-l.stoppedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *Limiter) bucketKey(req *AllowRequest) string {
	return fmt.Sprintf("%s:%s", req.ResourceType, req.APIKeyID)
}

func (l *Limiter) getOrCreateBucket(ctx context.Context, key string, req *AllowRequest, now time.Time) (*bucket, error) {
	l.mu.RLock()
	b := l.buckets[key]
	l.mu.RUnlock()
	if b != nil {
		return b, nil
	}

	b = newBucket(req, l.window, l.windowSeconds, now)
	if err := l.syncWindowUsage(ctx, b); err != nil {
		return nil, err
	}

	l.mu.Lock()
	if existing, ok := l.buckets[key]; ok {
		l.mu.Unlock()
		return existing, nil
	}
	l.buckets[key] = b
	l.mu.Unlock()
	return b, nil
}

func (l *Limiter) consume(ctx context.Context, b *bucket, req *AllowRequest, now time.Time) (bool, int, time.Duration, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := l.rotateWindowIfNeeded(ctx, b, now); err != nil {
		return false, 0, 0, err
	}

	if req.Limit != b.limit {
		b.updateLimit(req.Limit)
	}

	b.refill(now)

	if b.tokens < 1 {
		retryAfter := b.windowStart.Add(b.windowDuration).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, 0, retryAfter, nil
	}

	b.tokens -= 1
	b.pending++
	remaining := int(b.tokens)
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, 0, nil
}

func (l *Limiter) rotateWindowIfNeeded(ctx context.Context, b *bucket, now time.Time) error {
	currentWindow := now.Truncate(b.windowDuration)
	if currentWindow.Equal(b.windowStart) {
		return nil
	}

	if err := l.persistLocked(ctx, b); err != nil {
		return err
	}

	b.windowStart = currentWindow
	b.pending = 0
	b.tokens = b.capacity
	b.lastRefill = now

	if err := l.syncWindowUsage(ctx, b); err != nil {
		return err
	}

	return nil
}

func (l *Limiter) syncWindowUsage(ctx context.Context, b *bucket) error {
	apiKeyID := b.apiKeyID
	counter, err := l.repo.GetRateLimitCounter(ctx, b.tenantID, &apiKeyID, b.resource, b.windowStart)
	if errors.Is(err, domainAPIKey.ErrRateLimitCounterNotFound) {
		b.tokens = b.capacity
		return nil
	}
	if err != nil {
		return err
	}

	remaining := b.capacity - float64(counter.RequestCount)
	if remaining < 0 {
		remaining = 0
	}
	b.tokens = remaining
	return nil
}

func (l *Limiter) persistLocked(ctx context.Context, b *bucket) error {
	if b.pending == 0 {
		return nil
	}

	apiKeyID := b.apiKeyID
	req := &domainAPIKey.IncrementRateLimitRequest{
		TenantID:              b.tenantID,
		APIKeyID:              &apiKeyID,
		ResourceType:          b.resource,
		WindowDurationSeconds: b.windowSeconds,
		LimitValue:            b.limit,
		IncrementBy:           b.pending,
		WindowStart:           &b.windowStart,
	}

	if _, _, err := l.repo.IncrementRateLimit(ctx, req); err != nil {
		return err
	}

	b.pending = 0
	return nil
}

func (l *Limiter) flushLoop() {
	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.flush(context.Background())
		case <-l.stopCh:
			l.flush(context.Background())
			close(l.stoppedCh)
			return
		}
	}
}

func (l *Limiter) flush(ctx context.Context) {
	l.mu.RLock()
	buckets := make([]*bucket, 0, len(l.buckets))
	for _, bucket := range l.buckets {
		buckets = append(buckets, bucket)
	}
	l.mu.RUnlock()

	for _, b := range buckets {
		b.mu.Lock()
		err := l.persistLocked(ctx, b)
		b.mu.Unlock()
		if err != nil {
			logrus.WithError(err).Warn("ratelimiter: failed to persist bucket state")
		}
	}
}

type bucket struct {
	tenantID       string
	apiKeyID       string
	resource       string
	limit          int
	capacity       float64
	tokens         float64
	refillRate     float64
	windowDuration time.Duration
	windowSeconds  int
	windowStart    time.Time
	lastRefill     time.Time
	pending        int

	mu sync.Mutex
}

func newBucket(req *AllowRequest, window time.Duration, windowSeconds int, now time.Time) *bucket {
	capacity := float64(req.Limit)
	return &bucket{
		tenantID:       req.TenantID,
		apiKeyID:       req.APIKeyID,
		resource:       req.ResourceType,
		limit:          req.Limit,
		capacity:       capacity,
		tokens:         capacity,
		refillRate:     capacity / window.Seconds(),
		windowDuration: window,
		windowSeconds:  windowSeconds,
		windowStart:    now.Truncate(window),
		lastRefill:     now,
	}
}

func (b *bucket) updateLimit(limit int) {
	if limit <= 0 {
		return
	}
	b.limit = limit
	b.capacity = float64(limit)
	b.refillRate = b.capacity / b.windowDuration.Seconds()
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
}

func (b *bucket) refill(now time.Time) {
	if now.Before(b.lastRefill) {
		return
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	tokensToAdd := elapsed * b.refillRate
	if tokensToAdd <= 0 {
		return
	}

	b.tokens += tokensToAdd
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	b.lastRefill = now
}

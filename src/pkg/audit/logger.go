package auditlogger

import (
    "context"
    "fmt"
    "sync"
    "time"

    domainAudit "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/audit"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

type RequestEntry struct {
    TenantID     string
    APIKeyID     string
    ServerNodeID string
    Method       string
    Path         string
    StatusCode   int
    Latency      time.Duration
    IP           string
    UserAgent    string
    RequestID    string
    ResourceType string
    Action       string
    Error        error
}

type Logger interface {
    Record(entry RequestEntry)
    Close(ctx context.Context) error
}

type Option func(*asyncLogger)

func WithQueueSize(size int) Option {
    return func(l *asyncLogger) {
        if size > 0 {
            l.queueSize = size
        }
    }
}

func WithWorkerCount(count int) Option {
    return func(l *asyncLogger) {
        if count > 0 {
            l.workers = count
        }
    }
}

func WithServerNodeID(id string) Option {
    return func(l *asyncLogger) {
        l.serverNodeID = id
    }
}

type asyncLogger struct {
    repo         domainAudit.IAuditRepository
    entries      chan RequestEntry
    queueSize    int
    workers      int
    serverNodeID string

    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}

func NewLogger(repo domainAudit.IAuditRepository, opts ...Option) Logger {
    if repo == nil {
        panic("auditlogger: repository is required")
    }

    logger := &asyncLogger{
        repo:      repo,
        queueSize: 256,
        workers:   1,
    }

    for _, opt := range opts {
        opt(logger)
    }

    logger.entries = make(chan RequestEntry, logger.queueSize)
    logger.ctx, logger.cancel = context.WithCancel(context.Background())

    for i := 0; i < logger.workers; i++ {
        logger.wg.Add(1)
        go logger.run()
    }

    return logger
}

func (l *asyncLogger) Record(entry RequestEntry) {
    select {
    case <-l.ctx.Done():
        return
    case l.entries <- entry:
    }
}

func (l *asyncLogger) Close(ctx context.Context) error {
    l.cancel()
    close(l.entries)

    done := make(chan struct{})
    go func() {
        l.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (l *asyncLogger) run() {
    defer l.wg.Done()

    for {
        select {
        case <-l.ctx.Done():
            return
        case entry, ok := <-l.entries:
            if !ok {
                return
            }
            if err := l.writeLog(entry); err != nil {
                logrus.WithError(err).Warn("failed to record audit log")
            }
        }
    }
}

func (l *asyncLogger) writeLog(entry RequestEntry) error {
    status := domainAudit.AuditStatusSuccess
    switch {
    case entry.Error != nil || entry.StatusCode >= 500:
        status = domainAudit.AuditStatusError
    case entry.StatusCode >= 400:
        status = domainAudit.AuditStatusFailure
    }

    action := entry.Action
    if action == "" {
        action = fmt.Sprintf("%s %s", entry.Method, entry.Path)
    }

    resourceType := entry.ResourceType
    if resourceType == "" {
        resourceType = "rest_api"
    }

    logEntry := &domainAudit.AuditLog{
        ID:           uuid.NewString(),
        Action:       action,
        ResourceType: resourceType,
        ActorType:    domainAudit.ActorTypeAPIKey,
        Status:       status,
        IPAddress:    optionalString(entry.IP),
        UserAgent:    optionalString(entry.UserAgent),
        RequestID:    optionalString(entry.RequestID),
        Details: map[string]interface{}{
            "latency_ms": entry.Latency.Milliseconds(),
            "status_code": entry.StatusCode,
            "method": entry.Method,
            "path": entry.Path,
        },
    }

    if ptr := pointerString(entry.TenantID); ptr != nil {
        logEntry.TenantID = ptr
    }
    if ptr := pointerString(entry.APIKeyID); ptr != nil {
        logEntry.APIKeyID = ptr
        logEntry.ActorID = ptr
    }

    if ptr := pointerString(entry.ServerNodeID); ptr != nil {
        logEntry.ServerNodeID = ptr
    } else if ptr := pointerString(l.serverNodeID); ptr != nil {
        logEntry.ServerNodeID = ptr
    }

    return l.repo.CreateLog(context.Background(), logEntry)
}

func optionalString(val string) *string {
    if val == "" {
        return nil
    }
    copied := val
    return &copied
}

func pointerString(val string) *string {
    if val == "" {
        return nil
    }
    copied := val
    return &copied
}

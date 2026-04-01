package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/raynaythegreat/octai-app/pkg/logger"
)

var (
	emailRegex      = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex      = regexp.MustCompile(`\+?[0-9]{1,3}[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`)
	creditCardRegex = regexp.MustCompile(`\b[0-9]{13,19}\b`)
	apiKeyRegex     = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|credential)["\s:=]+["']?[^"'\s,}]+["']?`)
)

type AuditLogger interface {
	Log(ctx context.Context, entry *AuditEntry) error
	LogBatch(ctx context.Context, entries []*AuditEntry) error
	Close() error
}

type StructuredLogger struct {
	store       AuditStore
	buffer      []*AuditLog
	bufferMu    sync.Mutex
	bufferSize  int
	flushTicker *time.Ticker
	done        chan struct{}
	locationSvc LocationService
	redactPII   bool
}

type LoggerConfig struct {
	BufferSize    int
	FlushInterval time.Duration
	RedactPII     bool
}

func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
		RedactPII:     true,
	}
}

type LocationService interface {
	Lookup(ip string) *Location
}

type NoopLocationService struct{}

func (s *NoopLocationService) Lookup(ip string) *Location {
	return nil
}

func NewStructuredLogger(store AuditStore, config LoggerConfig) *StructuredLogger {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	sl := &StructuredLogger{
		store:       store,
		buffer:      make([]*AuditLog, 0, config.BufferSize),
		bufferSize:  config.BufferSize,
		done:        make(chan struct{}),
		locationSvc: &NoopLocationService{},
		redactPII:   config.RedactPII,
	}

	sl.flushTicker = time.NewTicker(config.FlushInterval)
	go sl.flushLoop()

	return sl
}

func (l *StructuredLogger) SetLocationService(svc LocationService) {
	l.locationSvc = svc
}

func (l *StructuredLogger) Log(ctx context.Context, entry *AuditEntry) error {
	log := l.entryToLog(entry)

	l.bufferMu.Lock()
	l.buffer = append(l.buffer, log)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.bufferMu.Unlock()

	if shouldFlush {
		go l.flush()
	}

	return nil
}

func (l *StructuredLogger) LogBatch(ctx context.Context, entries []*AuditEntry) error {
	logs := make([]*AuditLog, 0, len(entries))
	for _, entry := range entries {
		logs = append(logs, l.entryToLog(entry))
	}

	l.bufferMu.Lock()
	l.buffer = append(l.buffer, logs...)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.bufferMu.Unlock()

	if shouldFlush {
		go l.flush()
	}

	return nil
}

func (l *StructuredLogger) entryToLog(entry *AuditEntry) *AuditLog {
	changes := entry.Changes
	if l.redactPII && changes != nil {
		changes = l.redactSensitiveData(changes)
	}

	log := &AuditLog{
		ID:             uuid.New().String(),
		OrganizationID: entry.OrganizationID,
		UserID:         entry.UserID,
		Action:         entry.Action,
		ResourceType:   entry.ResourceType,
		ResourceID:     entry.ResourceID,
		Changes:        changes,
		IPAddress:      entry.IPAddress,
		UserAgent:      entry.UserAgent,
		Status:         entry.Status,
		ErrorMessage:   entry.ErrorMessage,
		Metadata:       entry.Metadata,
		Timestamp:      time.Now(),
	}

	if entry.IPAddress != "" && l.locationSvc != nil {
		log.Location = l.locationSvc.Lookup(entry.IPAddress)
	}

	if log.Status == "" {
		log.Status = StatusSuccess
	}

	return log
}

func (l *StructuredLogger) redactSensitiveData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		lowerKey := strings.ToLower(k)

		if isSensitiveField(lowerKey) {
			result[k] = "[REDACTED]"
			continue
		}

		switch val := v.(type) {
		case string:
			result[k] = l.redactString(val)
		case map[string]interface{}:
			result[k] = l.redactSensitiveData(val)
		default:
			result[k] = v
		}
	}
	return result
}

func (l *StructuredLogger) redactString(s string) string {
	s = emailRegex.ReplaceAllString(s, "[EMAIL REDACTED]")
	s = phoneRegex.ReplaceAllString(s, "[PHONE REDACTED]")
	s = creditCardRegex.ReplaceAllStringFunc(s, func(match string) string {
		if len(match) >= 13 && len(match) <= 19 {
			return "[CARD REDACTED]"
		}
		return match
	})
	s = apiKeyRegex.ReplaceAllString(s, "$1[REDACTED]")
	return s
}

func isSensitiveField(key string) bool {
	sensitivePatterns := []string{
		"password", "passwd", "pwd",
		"secret", "token", "apikey", "api_key",
		"credential", "private", "auth",
		"ssn", "social",
		"credit", "card", "cvv", "cvc",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}

func (l *StructuredLogger) flushLoop() {
	for {
		select {
		case <-l.flushTicker.C:
			l.flush()
		case <-l.done:
			l.flush()
			return
		}
	}
}

func (l *StructuredLogger) flush() {
	l.bufferMu.Lock()
	if len(l.buffer) == 0 {
		l.bufferMu.Unlock()
		return
	}

	logs := l.buffer
	l.buffer = make([]*AuditLog, 0, l.bufferSize)
	l.bufferMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := l.store.Store(ctx, logs); err != nil {
		logger.ErrorC("audit", "failed to flush audit logs: "+err.Error())
	}
}

func (l *StructuredLogger) Close() error {
	close(l.done)
	l.flushTicker.Stop()
	return nil
}

func LogAction(ctx context.Context, logger AuditLogger, orgID, userID string, action Action, resourceType ResourceType, resourceID string, changes map[string]interface{}) error {
	return logger.Log(ctx, &AuditEntry{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Changes:        changes,
	})
}

func LogWithRequest(ctx context.Context, logger AuditLogger, r *http.Request, orgID, userID string, action Action, resourceType ResourceType, resourceID string, changes map[string]interface{}) error {
	ip := extractIPAddress(r)
	userAgent := r.UserAgent()

	return logger.Log(ctx, &AuditEntry{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Changes:        changes,
		IPAddress:      ip,
		UserAgent:      userAgent,
	})
}

func LogError(ctx context.Context, logger AuditLogger, r *http.Request, orgID, userID string, action Action, resourceType ResourceType, resourceID string, errMsg string) error {
	ip := extractIPAddress(r)
	userAgent := r.UserAgent()

	return logger.Log(ctx, &AuditEntry{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      ip,
		UserAgent:      userAgent,
		Status:         StatusFailure,
		ErrorMessage:   errMsg,
	})
}

func extractIPAddress(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := r.RemoteAddr
	if colonIdx := strings.LastIndex(addr, ":"); colonIdx != -1 {
		addr = addr[:colonIdx]
	}
	return addr
}

type NoopLogger struct{}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (l *NoopLogger) Log(ctx context.Context, entry *AuditEntry) error {
	return nil
}

func (l *NoopLogger) LogBatch(ctx context.Context, entries []*AuditEntry) error {
	return nil
}

func (l *NoopLogger) Close() error {
	return nil
}

func ChangesFromJSON(data []byte) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

func ChangesToJSON(changes map[string]interface{}) []byte {
	if changes == nil {
		return nil
	}
	data, _ := json.Marshal(changes)
	return data
}

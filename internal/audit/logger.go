package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"
)

// Decision values for cache outcomes
const (
	DecisionL1Hit     = "L1_hit"
	DecisionL2aHit    = "L2a_hit"
	DecisionL2bAccept = "L2b_accept"
	DecisionL2bReject = "L2b_reject"
	DecisionBackend   = "backend"
)

// LogEvent represents a single structured audit entry.
type LogEvent struct {
	RequestID  string    `json:"request_id"`
	TenantID   string    `json:"tenant_id"`
	QueryHash  string    `json:"query_hash"` // SHA-256 of normalized query
	Domain     string    `json:"domain"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	Confidence float32   `json:"confidence,omitempty"`
	Timestamp   time.Time `json:"ts"`
}

// Logger handles async audit logging.
type Logger struct {
	// Simple wrapper around standard log for mvp (JSON output to stdout)
}

func NewLogger() *Logger {
	return &Logger{}
}

// Log records a cache decision event.
func (l *Logger) Log(ev LogEvent) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	
	data, err := json.Marshal(ev)
	if err == nil {
		log.Println(string(data))
	}
}

// HashQuery generates a SHA-256 hash for privacy-preserving audit logs.
func HashQuery(normalized string) string {
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

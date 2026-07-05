package audit

import "log/slog"

// Level selects which operations are audited.
type Level string

const (
	LevelAlterOnly Level = "alter_only" // only state-changing operations (default)
	LevelAll       Level = "all"        // reads too
)

// Service records audit entries subject to the enabled flag and level. When
// disabled it is a no-op, so call sites are unconditional. Write failures are
// logged and never propagated: auditing must never fail or delay the operation.
type Service struct {
	enabled bool
	level   Level
	w       Writer
	log     *slog.Logger
	user    string
}

// NewService builds an audit service. A nil writer or logger is tolerated
// (logger falls back to slog.Default). Pass enabled=false for a no-op service.
func NewService(enabled bool, level Level, w Writer, log *slog.Logger) *Service {
	if level != LevelAll {
		level = LevelAlterOnly
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{enabled: enabled, level: level, w: w, log: log, user: ResolveUser()}
}

// Enabled reports whether the service records anything.
func (s *Service) Enabled() bool { return s != nil && s.enabled }

// Record stamps the record with timestamp/user and writes it, honoring the
// configured level. Read-only operations are skipped at alter_only level. Nil
// service or writer is a no-op.
func (s *Service) Record(rec Record) {
	if s == nil || !s.enabled || s.w == nil {
		return
	}
	if s.level == LevelAlterOnly && !rec.isAltering() {
		return
	}
	rec.Timestamp = nowISO()
	if rec.User == "" {
		rec.User = s.user
	}
	if err := s.w.Write(rec); err != nil {
		s.log.Error("audit write failed", "err", err, "operation", rec.Operation)
	}
}

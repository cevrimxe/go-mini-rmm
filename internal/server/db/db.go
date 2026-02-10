package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	d, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := d.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	slog.Info("database initialized", "path", dbPath)
	return &Store{db: d}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// ---- Agents ----

func (s *Store) UpsertAgent(a models.HeartbeatPayload) error {
	_, err := s.db.Exec(`
		INSERT INTO agents (id, hostname, os, ip, version, last_heartbeat, status)
		VALUES (?, ?, ?, ?, ?, ?, 'online')
		ON CONFLICT(id) DO UPDATE SET
			hostname=excluded.hostname,
			os=excluded.os,
			ip=excluded.ip,
			version=excluded.version,
			last_heartbeat=excluded.last_heartbeat,
			status='online'
	`, a.AgentID, a.Hostname, a.OS, a.IP, a.Version, time.Now().UTC())
	return err
}

func (s *Store) ListAgents() ([]models.Agent, error) {
	rows, err := s.db.Query(`SELECT id, hostname, os, ip, version, last_heartbeat, status, created_at FROM agents ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.Hostname, &a.OS, &a.IP, &a.Version, &a.LastHeartbeat, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (s *Store) GetAgent(id string) (*models.Agent, error) {
	var a models.Agent
	err := s.db.QueryRow(`SELECT id, hostname, os, ip, version, last_heartbeat, status, created_at FROM agents WHERE id=?`, id).
		Scan(&a.ID, &a.Hostname, &a.OS, &a.IP, &a.Version, &a.LastHeartbeat, &a.Status, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) MarkOfflineAgents(timeout time.Duration) (int64, error) {
	res, err := s.db.Exec(`UPDATE agents SET status='offline' WHERE status='online' AND last_heartbeat < ?`,
		time.Now().UTC().Add(-timeout))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) DeleteAgent(id string) error {
	_, err := s.db.Exec(`DELETE FROM agents WHERE id=?`, id)
	return err
}

// ---- Metrics ----

func (s *Store) InsertMetric(m models.Metric) error {
	_, err := s.db.Exec(`INSERT INTO metrics (agent_id, cpu_percent, memory_percent, disk_percent, timestamp) VALUES (?, ?, ?, ?, ?)`,
		m.AgentID, m.CPUPercent, m.MemoryPercent, m.DiskPercent, time.Now().UTC())
	return err
}

func (s *Store) GetLatestMetrics(agentID string, limit int) ([]models.Metric, error) {
	rows, err := s.db.Query(`SELECT id, agent_id, cpu_percent, memory_percent, disk_percent, timestamp FROM metrics WHERE agent_id=? ORDER BY timestamp DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.Metric
	for rows.Next() {
		var m models.Metric
		if err := rows.Scan(&m.ID, &m.AgentID, &m.CPUPercent, &m.MemoryPercent, &m.DiskPercent, &m.Timestamp); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (s *Store) GetLatestMetric(agentID string) (*models.Metric, error) {
	var m models.Metric
	err := s.db.QueryRow(`SELECT id, agent_id, cpu_percent, memory_percent, disk_percent, timestamp FROM metrics WHERE agent_id=? ORDER BY timestamp DESC LIMIT 1`, agentID).
		Scan(&m.ID, &m.AgentID, &m.CPUPercent, &m.MemoryPercent, &m.DiskPercent, &m.Timestamp)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ---- Commands ----

func (s *Store) CreateCommand(agentID, command string) (*models.Command, error) {
	res, err := s.db.Exec(`INSERT INTO commands (agent_id, command, status) VALUES (?, ?, 'pending')`, agentID, command)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Command{
		ID:        id,
		AgentID:   agentID,
		Command:   command,
		Status:    models.CommandPending,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *Store) UpdateCommandResult(id int64, stdout, stderr string, exitCode int) error {
	status := models.CommandDone
	if exitCode != 0 {
		status = models.CommandFailed
	}
	_, err := s.db.Exec(`UPDATE commands SET stdout=?, stderr=?, exit_code=?, status=? WHERE id=?`,
		stdout, stderr, exitCode, status, id)
	return err
}

func (s *Store) GetCommandsByAgent(agentID string, limit int) ([]models.Command, error) {
	rows, err := s.db.Query(`SELECT id, agent_id, command, stdout, stderr, exit_code, status, created_at FROM commands WHERE agent_id=? ORDER BY created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []models.Command
	for rows.Next() {
		var c models.Command
		if err := rows.Scan(&c.ID, &c.AgentID, &c.Command, &c.Stdout, &c.Stderr, &c.ExitCode, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	return cmds, rows.Err()
}

// ---- Alert Rules ----

func (s *Store) CreateAlertRule(r models.AlertRuleRequest) (*models.AlertRule, error) {
	res, err := s.db.Exec(`INSERT INTO alert_rules (metric, operator, threshold) VALUES (?, ?, ?)`,
		r.Metric, r.Operator, r.Threshold)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.AlertRule{
		ID:        id,
		Metric:    r.Metric,
		Operator:  r.Operator,
		Threshold: r.Threshold,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *Store) ListAlertRules() ([]models.AlertRule, error) {
	rows, err := s.db.Query(`SELECT id, metric, operator, threshold, created_at FROM alert_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.AlertRule
	for rows.Next() {
		var r models.AlertRule
		if err := rows.Scan(&r.ID, &r.Metric, &r.Operator, &r.Threshold, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *Store) DeleteAlertRule(id int64) error {
	_, err := s.db.Exec(`DELETE FROM alert_rules WHERE id=?`, id)
	return err
}

// ---- Alerts ----

func (s *Store) CreateAlert(ruleID int64, agentID, message string) error {
	_, err := s.db.Exec(`INSERT INTO alerts (rule_id, agent_id, message) VALUES (?, ?, ?)`,
		ruleID, agentID, message)
	return err
}

func (s *Store) ListAlerts(limit int) ([]models.Alert, error) {
	rows, err := s.db.Query(`SELECT id, rule_id, agent_id, message, resolved, created_at FROM alerts ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.RuleID, &a.AgentID, &a.Message, &a.Resolved, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (s *Store) ResolveAlert(id int64) error {
	_, err := s.db.Exec(`UPDATE alerts SET resolved=1 WHERE id=?`, id)
	return err
}

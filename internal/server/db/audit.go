package db

import (
	"time"

	"github.com/cevrimxe/go-mini-rmm/internal/models"
)

func (s *Store) InsertAuditLog(username, action, target, details string) error {
	_, err := s.db.Exec(`INSERT INTO audit_logs (username, action, target, details, created_at) VALUES (?, ?, ?, ?, ?)`,
		username, action, target, details, time.Now().UTC())
	return err
}

func (s *Store) GetAuditLogs(limit int) ([]models.AuditLog, error) {
	rows, err := s.db.Query(`SELECT id, username, action, target, details, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.Username, &l.Action, &l.Target, &l.Details, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

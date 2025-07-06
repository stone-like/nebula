package memory

import (
	"database/sql"
	"fmt"
)

// CreateSession creates a new session in the database
func (d *Database) CreateSession(session *Session) error {
	query := `
		INSERT INTO sessions (id, started_at, project_path, model_used)
		VALUES (?, ?, ?, ?)
	`
	_, err := d.db.Exec(query, session.ID, session.StartedAt, session.ProjectPath, session.ModelUsed)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// EndSession marks a session as ended
func (d *Database) EndSession(sessionID string) error {
	query := `UPDATE sessions SET ended_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID
func (d *Database) GetSession(sessionID string) (*Session, error) {
	query := `SELECT id, started_at, ended_at, project_path, model_used FROM sessions WHERE id = ?`
	row := d.db.QueryRow(query, sessionID)

	var session Session
	var endedAt sql.NullTime
	err := row.Scan(&session.ID, &session.StartedAt, &endedAt, &session.ProjectPath, &session.ModelUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}

	return &session, nil
}

// GetSessionsByProject retrieves sessions for a specific project path
func (d *Database) GetSessionsByProject(projectPath string, limit int) ([]*SessionSummary, error) {
	query := `
		SELECT s.id, s.started_at, s.ended_at, s.project_path, s.model_used,
			   COUNT(m.id) as message_count,
			   COALESCE(
				   (SELECT content FROM messages WHERE session_id = s.id ORDER BY timestamp DESC LIMIT 1),
				   ''
			   ) as last_message
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE s.project_path = ?
		GROUP BY s.id
		ORDER BY s.started_at DESC
		LIMIT ?
	`
	rows, err := d.db.Query(query, projectPath, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by project: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionSummary
	for rows.Next() {
		var summary SessionSummary
		var endedAt sql.NullTime
		err := rows.Scan(
			&summary.ID, &summary.StartedAt, &endedAt, &summary.ProjectPath,
			&summary.ModelUsed, &summary.MessageCount, &summary.LastMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session summary: %w", err)
		}

		if endedAt.Valid {
			summary.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, &summary)
	}

	return sessions, nil
}

// SaveMessage saves a message to the database
func (d *Database) SaveMessage(message *Message) error {
	query := `
		INSERT INTO messages (session_id, timestamp, role, content, tool_calls, tool_results)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query, message.SessionID, message.Timestamp, message.Role, message.Content, message.ToolCalls, message.ToolResults)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	message.ID = int(id)

	return nil
}

// GetSessionMessages retrieves all messages for a session
func (d *Database) GetSessionMessages(sessionID string) ([]*Message, error) {
	query := `
		SELECT id, session_id, timestamp, role, content, tool_calls, tool_results
		FROM messages
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`
	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var message Message
		var toolCalls, toolResults sql.NullString
		err := rows.Scan(
			&message.ID, &message.SessionID, &message.Timestamp,
			&message.Role, &message.Content, &toolCalls, &toolResults,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if toolCalls.Valid {
			message.ToolCalls = &toolCalls.String
		}
		if toolResults.Valid {
			message.ToolResults = &toolResults.String
		}

		messages = append(messages, &message)
	}

	return messages, nil
}

// GetRecentSessions retrieves the most recent sessions across all projects
func (d *Database) GetRecentSessions(limit int) ([]*SessionSummary, error) {
	query := `
		SELECT s.id, s.started_at, s.ended_at, s.project_path, s.model_used,
			   COUNT(m.id) as message_count,
			   COALESCE(
				   (SELECT content FROM messages WHERE session_id = s.id ORDER BY timestamp DESC LIMIT 1),
				   ''
			   ) as last_message
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		GROUP BY s.id
		ORDER BY s.started_at DESC
		LIMIT ?
	`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionSummary
	for rows.Next() {
		var summary SessionSummary
		var endedAt sql.NullTime
		err := rows.Scan(
			&summary.ID, &summary.StartedAt, &endedAt, &summary.ProjectPath,
			&summary.ModelUsed, &summary.MessageCount, &summary.LastMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session summary: %w", err)
		}

		if endedAt.Valid {
			summary.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, &summary)
	}

	return sessions, nil
}

// DeleteSession deletes a session and all its messages
func (d *Database) DeleteSession(sessionID string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete messages first
	if _, err := tx.Exec("DELETE FROM messages WHERE session_id = ?", sessionID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete session
	if _, err := tx.Exec("DELETE FROM sessions WHERE id = ?", sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
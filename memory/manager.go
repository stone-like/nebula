package memory

import (
	"fmt"
	"os"
	"time"
)

// Manager handles memory operations
type Manager struct {
	db             *Database
	currentSession *Session
}

// NewManager creates a new memory manager
func NewManager(dbPath string) (*Manager, error) {
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &Manager{
		db: db,
	}, nil
}

// Close closes the memory manager
func (m *Manager) Close() error {
	// End current session if active
	if m.currentSession != nil && m.currentSession.IsActive() {
		if err := m.EndSession(); err != nil {
			return fmt.Errorf("failed to end current session: %w", err)
		}
	}

	return m.db.Close()
}

// StartSession creates a new session or restores an existing one
func (m *Manager) StartSession(projectPath, modelUsed string) (*Session, error) {
	// Generate session ID based on timestamp
	sessionID := fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))

	session := &Session{
		ID:          sessionID,
		StartedAt:   time.Now(),
		ProjectPath: projectPath,
		ModelUsed:   modelUsed,
	}

	if err := m.db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.currentSession = session
	return session, nil
}

// RestoreSession restores an existing session
func (m *Manager) RestoreSession(sessionID string) (*Session, error) {
	session, err := m.db.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	m.currentSession = session
	return session, nil
}

// EndSession ends the current session
func (m *Manager) EndSession() error {
	if m.currentSession == nil {
		return nil
	}

	if err := m.db.EndSession(m.currentSession.ID); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Update local session
	now := time.Now()
	m.currentSession.EndedAt = &now
	m.currentSession = nil

	return nil
}

// GetCurrentSession returns the current session
func (m *Manager) GetCurrentSession() *Session {
	return m.currentSession
}

// SaveMessage saves a message to the current session
func (m *Manager) SaveMessage(role, content string, toolCalls, toolResults interface{}) error {
	if m.currentSession == nil {
		return nil
	}

	message := &Message{
		SessionID: m.currentSession.ID,
		Timestamp: time.Now(),
		Role:      role,
		Content:   content,
	}

	// Convert tool calls/results to JSON strings if provided
	if toolCalls != nil {
		if toolCallsJSON, ok := toolCalls.(string); ok {
			message.ToolCalls = &toolCallsJSON
		}
	}
	if toolResults != nil {
		if toolResultsJSON, ok := toolResults.(string); ok {
			message.ToolResults = &toolResultsJSON
		}
	}

	return m.db.SaveMessage(message)
}

// GetSessionsByProject returns sessions for the current project
func (m *Manager) GetSessionsByProject(projectPath string, limit int) ([]*SessionSummary, error) {
	return m.db.GetSessionsByProject(projectPath, limit)
}

// GetCurrentProjectSessions returns sessions for the current working directory
func (m *Manager) GetCurrentProjectSessions(limit int) ([]*SessionSummary, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return m.GetSessionsByProject(currentDir, limit)
}

// GetSessionMessages returns all messages for a session
func (m *Manager) GetSessionMessages(sessionID string) ([]*Message, error) {
	return m.db.GetSessionMessages(sessionID)
}

// GetRecentSessions returns recent sessions across all projects
func (m *Manager) GetRecentSessions(limit int) ([]*SessionSummary, error) {
	return m.db.GetRecentSessions(limit)
}

// DeleteSession deletes a session and all its messages
func (m *Manager) DeleteSession(sessionID string) error {
	// If deleting current session, clear it
	if m.currentSession != nil && m.currentSession.ID == sessionID {
		m.currentSession = nil
	}

	return m.db.DeleteSession(sessionID)
}
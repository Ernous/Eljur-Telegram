package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"school-diary-bot/bot/eljur"
)

// SessionData represents user session data for serverless environment
type SessionData struct {
	ChatID        int64             `json:"chat_id"`
	State         string            `json:"state"`
	AuthStep      int               `json:"auth_step"`
	TempLogin     string            `json:"temp_login"`
	TempPassword  string            `json:"temp_password"`
	TempRecipient string            `json:"temp_recipient"`
	CurrentWeek   string            `json:"current_week"`
	CurrentPeriod string            `json:"current_period"`
	GeminiAPIKey  string            `json:"gemini_api_key"`
	GeminiModel   string            `json:"gemini_model"`
	GeminiContext string            `json:"gemini_context"`
	CreatedAt     time.Time         `json:"created_at"`
	LastAccess    time.Time         `json:"last_access"`
	EljurAuth     *EljurAuthData    `json:"eljur_auth,omitempty"`
}

// EljurAuthData stores authentication information for Eljur
type EljurAuthData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// SessionManager manages user sessions in serverless environment
type SessionManager struct {
	sessions map[int64]*SessionData
	mutex    sync.RWMutex
}

// Global session manager instance
var globalSessionManager = &SessionManager{
	sessions: make(map[int64]*SessionData),
}

// GetUserStateServerless gets or creates user state for serverless environment
func (b *Bot) GetUserStateServerless(chatID int64) *UserState {
	sessionData := globalSessionManager.GetSession(chatID)
	
	// Create UserState from session data
	userState := &UserState{
		ChatID:        sessionData.ChatID,
		State:         sessionData.State,
		AuthStep:      sessionData.AuthStep,
		TempLogin:     sessionData.TempLogin,
		TempPassword:  sessionData.TempPassword,
		TempRecipient: sessionData.TempRecipient,
		Client:        eljur.NewClient(),
		CurrentWeek:   sessionData.CurrentWeek,
		CurrentPeriod: sessionData.CurrentPeriod,
		GeminiAPIKey:  sessionData.GeminiAPIKey,
		GeminiModel:   sessionData.GeminiModel,
		GeminiContext: sessionData.GeminiContext,
	}

	// Restore Eljur authentication if available
	if sessionData.EljurAuth != nil && sessionData.EljurAuth.Login != "" {
		// Restore authentication without exposing sensitive data
		if err := userState.Client.RestoreSession(sessionData.EljurAuth.Login, sessionData.EljurAuth.Token); err != nil {
			log.Printf("Failed to restore Eljur session for user %d: %v", chatID, err)
			// Clear invalid auth data
			sessionData.EljurAuth = nil
		}
	}

	return userState
}

// SaveUserStateServerless saves user state for serverless environment
func (b *Bot) SaveUserStateServerless(userState *UserState) {
	sessionData := &SessionData{
		ChatID:        userState.ChatID,
		State:         userState.State,
		AuthStep:      userState.AuthStep,
		TempLogin:     userState.TempLogin,
		TempPassword:  userState.TempPassword,
		TempRecipient: userState.TempRecipient,
		CurrentWeek:   userState.CurrentWeek,
		CurrentPeriod: userState.CurrentPeriod,
		GeminiAPIKey:  userState.GeminiAPIKey,
		GeminiModel:   userState.GeminiModel,
		GeminiContext: userState.GeminiContext,
		LastAccess:    time.Now(),
	}

	// Save Eljur authentication if available
	if userState.Client.IsAuthenticated() {
		sessionData.EljurAuth = &EljurAuthData{
			Login: userState.Client.GetLogin(),
			Token: userState.Client.GetToken(),
		}
	}

	globalSessionManager.SaveSession(sessionData)
}

// GetSession gets session data for a user
func (sm *SessionManager) GetSession(chatID int64) *SessionData {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if session, exists := sm.sessions[chatID]; exists {
		// Update last access time
		session.LastAccess = time.Now()
		return session
	}

	// Create new session
	newSession := &SessionData{
		ChatID:     chatID,
		State:      "idle",
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}

	sm.mutex.RUnlock()
	sm.mutex.Lock()
	sm.sessions[chatID] = newSession
	sm.mutex.Unlock()
	sm.mutex.RLock()

	return newSession
}

// SaveSession saves session data for a user
func (sm *SessionManager) SaveSession(sessionData *SessionData) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionData.LastAccess = time.Now()
	sm.sessions[sessionData.ChatID] = sessionData

	// Clean up old sessions (older than 24 hours)
	sm.cleanupOldSessions()
}

// cleanupOldSessions removes sessions older than 24 hours
func (sm *SessionManager) cleanupOldSessions() {
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for chatID, session := range sm.sessions {
		if session.LastAccess.Before(cutoff) {
			delete(sm.sessions, chatID)
			log.Printf("Cleaned up old session for user %d", chatID)
		}
	}
}

// ClearSession removes session data for a user
func (sm *SessionManager) ClearSession(chatID int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	delete(sm.sessions, chatID)
}

// GetSessionJSON returns session data as JSON string for debugging
func (sm *SessionManager) GetSessionJSON(chatID int64) string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if session, exists := sm.sessions[chatID]; exists {
		if jsonData, err := json.MarshalIndent(session, "", "  "); err == nil {
			return string(jsonData)
		}
	}
	
	return fmt.Sprintf(`{"error": "Session not found for chat ID %d"}`, chatID)
}

// GetStats returns session statistics
func (sm *SessionManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	activeSessions := 0
	authenticatedSessions := 0
	
	for _, session := range sm.sessions {
		activeSessions++
		if session.EljurAuth != nil && session.EljurAuth.Token != "" {
			authenticatedSessions++
		}
	}

	return map[string]interface{}{
		"total_sessions":         activeSessions,
		"authenticated_sessions": authenticatedSessions,
		"last_cleanup":          time.Now().Format(time.RFC3339),
	}
}
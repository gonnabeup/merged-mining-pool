package pool

import "sync"

type sessionMap map[string]*stratumClient

var (
    sessions     sessionMap
    sessionsMutex sync.RWMutex  // Changed from sessionMutex to sessionsMutex
)

func initiateSessions() {
    sessions = make(sessionMap)
}

func addSession(client *stratumClient) {
    sessionsMutex.Lock()
    defer sessionsMutex.Unlock()
    sessions[client.sessionID] = client
}

func removeSession(client *stratumClient) {
    sessionsMutex.Lock()
    defer sessionsMutex.Unlock()
    delete(sessions, client.sessionID)
}

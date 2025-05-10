package pool

import "sync"

type sessionMap map[string]*stratumClient

var (
    sessions     sessionMap
    sessionMutex sync.RWMutex
)

func initiateSessions() {
    sessions = make(sessionMap)
}

func addSession(client *stratumClient) {
    sessionMutex.Lock()
    defer sessionMutex.Unlock()
    sessions[client.sessionID] = client
}

func removeSession(client *stratumClient) {
    sessionMutex.Lock()
    defer sessionMutex.Unlock()
    delete(sessions, client.sessionID)
}

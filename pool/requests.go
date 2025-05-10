package pool

import (
	"encoding/json"

	"designs.capital/dogepool/bitcoin"
)

type stratumRequest struct {
	Id     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func miningNotify(work bitcoin.Work) stratumRequest {
	var request stratumRequest

	params, err := json.Marshal(work)
	logOnError(err)

	request.Method = "mining.notify"
	request.Params = params

	return request
}

func miningSetExtranonce(extranonce string) stratumRequest {
	var request stratumRequest

	// TODO build request; I need a better example

	return request
}

func miningSetDifficulty(difficulty float64) stratumRequest {
	var request stratumRequest

	request.Method = "mining.set_difficulty"

	diff := []float64{difficulty}

	var err error
	request.Params, err = json.Marshal(diff)
	logOnError(err)

	return request
}

type stratumResponse struct {
    ID     json.RawMessage     `json:"id"`
    Result interface{}         `json:"result"`
    Error  interface{}         `json:"error"`
}

// Remove the stratumResponse struct definition

func (p *PoolServer) handleStratumRequest(client *stratumClient, req stratumRequest) error {
    switch req.Method {
    case "mining.configure":
        response := stratumResponse{
            ID: req.Id,    // Changed from Id to ID
            Result: map[string]interface{}{
                "version-rolling": false,
                "minimum-difficulty": true,
                "subscribe-extranonce": true,
            },
            Error: nil,
        }
        return sendPacket(response, client)
    }
    return nil
}

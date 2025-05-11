package pool

import (
	"designs.capital/dogepool/bitcoin"
	"log"
	"math/big"
)

const (
	shareInvalid = iota
	shareValid
	shareBlock
	primaryCandidate
	aux1Candidate
	dualCandidate
)

var statusMap = map[int]string{
	2: "Primary",
	3: "Aux1",
	4: "Dual",
}

func validateAndWeighShare(primaryBlockTemplate *bitcoin.BitcoinBlock, auxBlock *bitcoin.AuxBlock, poolDifficulty float64) (int, float64) {
    // Add debug logging
    log.Printf("Validating share - Pool Difficulty: %f", poolDifficulty)
    
    // Add header hash logging
    headerHash, err := primaryBlockTemplate.HeaderDigest()
    if err != nil {
        log.Printf("Error calculating header digest: %v", err)
        return shareInvalid, 0
    }
    log.Printf("Header hash: %s", headerHash)
    
    // Calculate share difficulty
    hashBig := new(big.Int).SetBytes([]byte(headerHash))
    shareDifficulty := bitcoin.CalculateDifficulty(hashBig)
    
    log.Printf("Share difficulty: %f, Pool difficulty: %f", shareDifficulty, poolDifficulty)
    
    // Validate against pool difficulty
    if shareDifficulty < poolDifficulty {
        log.Printf("Share rejected - Difficulty too low (share: %f < pool: %f)", 
                  shareDifficulty, poolDifficulty)
        return shareInvalid, shareDifficulty
    }

    // Check if this could be a block
    if shareDifficulty >= primaryBlockTemplate.NetworkDifficulty() {
        return shareBlock, shareDifficulty
    }

    return shareValid, shareDifficulty
}

package pool

import (
	"designs.capital/dogepool/bitcoin"
	"log"
	"math/big"
	"encoding/hex"
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
    log.Printf("Validating share - Pool Difficulty: %f", poolDifficulty)
    
    // Get header hash using chain's digest function
    headerHash, err := primaryBlockTemplate.chain.HeaderDigest(primaryBlockTemplate.header)
    if err != nil {
        log.Printf("Error calculating header digest: %v", err)
        return shareInvalid, 0
    }
    log.Printf("Header hash: %s", headerHash)
    
    // Convert hex string to bytes
    hashBytes, err := hex.DecodeString(headerHash)
    if err != nil {
        log.Printf("Error decoding hash: %v", err)
        return shareInvalid, 0
    }
    
    // Calculate difficulty from target bits
    target := bitcoin.Target(primaryBlockTemplate.Template.Bits)
    shareDifficulty, _ := target.ToDifficulty()
    
    log.Printf("Share difficulty: %f, Pool difficulty: %f", shareDifficulty, poolDifficulty)
    
    if shareDifficulty < poolDifficulty {
        log.Printf("Share rejected - Difficulty too low (share: %f < pool: %f)", 
                  shareDifficulty, poolDifficulty)
        return shareInvalid, shareDifficulty
    }

    // Get network difficulty from chain
    networkDiff := primaryBlockTemplate.chain.NetworkDifficulty()
    if shareDifficulty >= networkDiff {
        return shareBlock, shareDifficulty
    }

    return shareValid, shareDifficulty
}

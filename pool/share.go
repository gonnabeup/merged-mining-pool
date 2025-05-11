package pool

import (
    "designs.capital/dogepool/bitcoin"
    "designs.capital/dogepool/config"
    "log"
    "time"
    "sync"
)

// Share status constants
const (
    shareInvalid = iota
    shareValid
    shareBlock
    primaryCandidate
    aux1Candidate
    dualCandidate
)

type minerDifficulty struct {
    difficulty     float64
    lastShareTime  time.Time
    lastRetarget   time.Time
    shareCount     int
}

var minerDiffs = make(map[string]*minerDifficulty)
var diffLock sync.RWMutex

// VarDiff configuration from pool config
var (
    varDiffEnabled    bool
    targetShareTime   int
    variancePercent   float64
    retargetTime      int
    minDifficulty     float64
    maxDifficulty     float64
)

func InitVarDiff(config *config.Config) {
    varDiffEnabled = config.VarDiff.Enabled
    targetShareTime = config.VarDiff.TargetTime
    variancePercent = config.VarDiff.VariancePercent
    retargetTime = config.VarDiff.RetargetTime
    minDifficulty = config.VarDiff.MinDiff
    maxDifficulty = config.VarDiff.MaxDiff

    // Set default minimum difficulty if not configured
    if minDifficulty == 0 {
        minDifficulty = 200000 // Default minimum difficulty for S19+ miners
    }
}

// Update function signature to remove poolDifficulty parameter
func validateAndWeighShare(primaryBlockTemplate *bitcoin.BitcoinBlock, auxBlock *bitcoin.AuxBlock, minerAddr string) (int, float64) {
    // Get header hash using HeaderHashed method
    headerHash, err := primaryBlockTemplate.HeaderHashed()
    if err != nil {
        log.Printf("Error calculating header digest: %v", err)
        return shareInvalid, 0
    }
    log.Printf("Header hash: %s", headerHash)
    
    // Calculate difficulty from target bits
    target := bitcoin.Target(primaryBlockTemplate.Template.Bits)
    shareDifficulty, _ := target.ToDifficulty()
    
    // Get updated difficulty for this miner
    currentDiff := getUpdatedDifficulty(minerAddr, shareDifficulty)
    
    log.Printf("Share difficulty: %f, Current miner difficulty: %f", shareDifficulty, currentDiff)
    
    if shareDifficulty < currentDiff {
        log.Printf("Share rejected - Difficulty too low (share: %f < required: %f)", 
                  shareDifficulty, currentDiff)
        return shareInvalid, shareDifficulty
    }

    // Get network difficulty from target bits
    networkTarget := bitcoin.Target(primaryBlockTemplate.Template.Bits)
    networkDiff, _ := networkTarget.ToDifficulty()
    if shareDifficulty >= networkDiff {
        return shareBlock, shareDifficulty
    }

    return shareValid, shareDifficulty
}

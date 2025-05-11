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
    shareInvalid = iota  // 0
    shareValid           // 1
    shareBlock           // 2
    primaryCandidate     // 3
    aux1Candidate        // 4
    dualCandidate        // 5
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
    shareDifficulty, _ := target.ToDifficulty()  // Remove unused accuracy variable
    if shareDifficulty == 0 {
        log.Printf("Error: Invalid share difficulty calculated from bits %s", primaryBlockTemplate.Template.Bits)
        return shareInvalid, 0
    }
    
    // Get updated difficulty for this miner
    currentDiff := getUpdatedDifficulty(minerAddr, shareDifficulty)
    if currentDiff == 0 {
        currentDiff = minDifficulty // Ensure we never have 0 difficulty
    }
    
    log.Printf("Share difficulty: %f, Current miner difficulty: %f", shareDifficulty, currentDiff)
    
    if shareDifficulty < currentDiff {
        log.Printf("Share rejected - Difficulty too low (share: %f < required: %f)", 
                  shareDifficulty, currentDiff)
        return shareInvalid, shareDifficulty
    }

    // Get network difficulty from target bits
    networkTarget := bitcoin.Target(primaryBlockTemplate.Template.Bits)
    networkDiff, _ := networkTarget.ToDifficulty()
    
    // Check if this is a block candidate
    if shareDifficulty >= networkDiff {
        if auxBlock != nil {
            // Check aux chain target
            auxTarget := bitcoin.Target(auxBlock.Target)
            auxDiff, _ := auxTarget.ToDifficulty()  // Remove unused auxAccuracy variable
            if auxDiff == 0 {
                log.Printf("Warning: Invalid aux difficulty calculated from target %s", auxBlock.Target)
                return primaryCandidate, shareDifficulty
            }
            if shareDifficulty >= auxDiff {
                return dualCandidate, shareDifficulty
            }
            return primaryCandidate, shareDifficulty
        }
        return primaryCandidate, shareDifficulty
    }

    return shareValid, shareDifficulty
}

func getUpdatedDifficulty(minerAddr string, shareDiff float64) float64 {
    if !varDiffEnabled {
        return minDifficulty
    }

    diffLock.Lock()
    defer diffLock.Unlock()

    now := time.Now()
    diff, exists := minerDiffs[minerAddr]
    if !exists {
        minerDiffs[minerAddr] = &minerDifficulty{
            difficulty:    minDifficulty,
            lastShareTime: now,
            lastRetarget:  now,
            shareCount:    1,
        }
        return minDifficulty
    }

    diff.shareCount++
    
    // Check if it's time to retarget
    if now.Sub(diff.lastRetarget).Seconds() >= float64(retargetTime) {
        averageShareTime := now.Sub(diff.lastRetarget).Seconds() / float64(diff.shareCount)
        
        // Adjust difficulty based on share time
        if averageShareTime < float64(targetShareTime)*(1-variancePercent/100.0) {
            diff.difficulty *= 2
        } else if averageShareTime > float64(targetShareTime)*(1+variancePercent/100.0) {
            diff.difficulty /= 2
        }

        // Clamp difficulty
        if diff.difficulty < minDifficulty {
            diff.difficulty = minDifficulty
        } else if diff.difficulty > maxDifficulty {
            diff.difficulty = maxDifficulty
        }

        // Reset counters
        diff.lastRetarget = now
        diff.shareCount = 0
    }

    diff.lastShareTime = now
    return diff.difficulty
}

package pool

import (
    "designs.capital/dogepool/bitcoin"
    "log"
    "time"
    "sync"
)

// VarDiff configuration
const (
    targetShareTime = 15              // Target time between shares in seconds
    variancePercent = 30              // Allow variance of ±30%
    retargetTime    = 120             // Check to retarget every 120 seconds
    minDifficulty   = 200000          // Minimum difficulty
    maxDifficulty   = 4294967296.0    // Maximum difficulty
    targetShares    = 4               // Target number of shares per retarget time
)

type minerDifficulty struct {
    difficulty     float64
    lastShareTime  time.Time
    lastRetarget   time.Time
    shareCount     int
}

var minerDiffs = make(map[string]*minerDifficulty)
var diffLock sync.RWMutex

func getUpdatedDifficulty(minerAddr string, shareDiff float64) float64 {
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
    if now.Sub(diff.lastRetarget).Seconds() >= retargetTime {
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
        
        log.Printf("Adjusted difficulty for miner %s to %f (avg share time: %.2fs)", 
                  minerAddr, diff.difficulty, averageShareTime)
    }

    diff.lastShareTime = now
    return diff.difficulty
}

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

func validateAndWeighShare(primaryBlockTemplate *bitcoin.BitcoinBlock, auxBlock *bitcoin.AuxBlock, minerAddr string, poolDifficulty float64) (int, float64) {
    log.Printf("Validating share - Pool Difficulty: %f", poolDifficulty)
    
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

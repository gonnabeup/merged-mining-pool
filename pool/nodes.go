package pool

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/rpc"
	"github.com/go-zeromq/zmq4"
)

type BlockChainNodesMap map[string]blockChainNode // "blockChainName" => activeNode

type blockChainNode struct {
	NotifyURL          string
	RPC                *rpc.RPCClient
	ChainName          string
	Network            string
	RewardPubScriptKey string // TODO - this is very bitcoin specific.  Abstract to interface.
	RewardTo           string
	NetworkDifficulty  float64
}

func (p *PoolServer) GetPrimaryNode() blockChainNode {
	return p.activeNodes[p.config.GetPrimary()]
}

func (p *PoolServer) GetAux1Node() blockChainNode {
	return p.activeNodes[p.config.GetAux1()]
}

type hashblockCounterMap map[string]uint32 // "blockChainName" => hashblock msg counter

func (pool *PoolServer) loadBlockchainNodes() {
	pool.activeNodes = make(BlockChainNodesMap)
	for _, blockChainName := range pool.config.BlockChainOrder {
		rpcManager, exists := pool.rpcManagers[blockChainName]
		if !exists {
			panic("Blockchain not found for: " + blockChainName)
		}
		rpcClient := rpcManager.GetActiveClient()
		nodeConfig := pool.config.BlockchainNodes[blockChainName][rpcManager.GetIndex()]

		chainInfo, err := rpcClient.GetBlockChainInfo()
		logFatalOnError(err)

		address, err := rpcClient.ValidateAddress(nodeConfig.RewardTo)
		logFatalOnError(err)

		// TODO this is wayy to bitcoin specific.  Move this to the coin package.
		rewardPubScriptKey := address.ScriptPubKey

		newNode := blockChainNode{
			NotifyURL:          nodeConfig.NotifyURL,
			RPC:                rpcClient,
			Network:            chainInfo.Chain,
			RewardPubScriptKey: rewardPubScriptKey,
			RewardTo:           nodeConfig.RewardTo,
			NetworkDifficulty:  chainInfo.NetworkDifficulty,
			ChainName:          blockChainName,
		}
		pool.activeNodes[blockChainName] = newNode
	}
}

func (pool *PoolServer) listenForBlockNotifications() error {
	notifyChannel := make(chan hashBlockResponse)
	hashblockCounterMap := make(hashblockCounterMap)

	for blockChainName := range pool.activeNodes {
		subscription, err := pool.createZMQSubscriptionToHashBlock(blockChainName, notifyChannel)
		if err != nil {
			return err
		}
		defer subscription.Close()
	}

	for {
		msg := <-notifyChannel
		chainName := msg.blockChainName
		prevCount := hashblockCounterMap[chainName]
		newCount := msg.blockHashCounter
		prevBlockHash := msg.previousBlockHash

		m := "**New %v block: %v - %v**"
		log.Printf(m, chainName, newCount, prevBlockHash)

		if prevCount != 0 && (prevCount+1) != newCount {
			m = "We missed a %v block notification, previous count: %v current count: %v"
			log.Printf(m, chainName, prevCount, newCount)
		}

		hashblockCounterMap[chainName] = newCount

		err := pool.fetchRpcBlockTemplatesAndCacheWork()
		logOnError(err)
		work, err := pool.generateWorkFromCache(true)
		logOnError(err)
		pool.broadcastWork(work)
	}
}

// Ultimate program OUTPUT
func (p *PoolServer) submitBlockToChain(block *bitcoin.BitcoinBlock) error {
    primaryNode := p.GetPrimaryNode()
    blockHex := block.ToHex()
    
    response, err := primaryNode.RPC.SubmitBlock(blockHex)
    if err != nil {
        if strings.Contains(err.Error(), "high-hash") {
            log.Printf("Block rejected due to high hash value: %s", err)
            return nil // Don't treat high-hash as an error, it's expected for some shares
        }
        return fmt.Errorf("error submitting block: %v", err)
    }
    
    log.Printf("Successfully submitted block to chain: %s", response)
    return nil
}

func (p *PoolServer) submitAuxBlock(primaryBlock bitcoin.BitcoinBlock, aux1Block bitcoin.AuxBlock) error {
	auxpow := bitcoin.MakeAuxPow(primaryBlock)
	success, err := p.GetAux1Node().RPC.SubmitAuxBlock(aux1Block.Hash, auxpow.Serialize())
	if !success {
		nodeName := p.GetAux1Node().ChainName
		m := "⚠️  %v node failed to submit aux block: %v"
		m = fmt.Sprintf(m, nodeName, err.Error())
		return errors.New(m)
	}
	return err
}

type hashBlockResponse struct {
	blockChainName    string
	previousBlockHash string
	blockHashCounter  uint32
}

func (p *PoolServer) createZMQSubscriptionToHashBlock(blockChainName string, hashBlockChannel chan hashBlockResponse) (zmq4.Socket, error) {
	sub := zmq4.NewSub(context.Background())

	url := p.activeNodes[blockChainName].NotifyURL
	err := sub.Dial(url)
	if err != nil {
		return sub, err
	}

	err = sub.SetOption(zmq4.OptionSubscribe, "hashblock")
	if err != nil {
		return sub, err
	}

	logErr := func(msg zmq4.Msg, err error) zmq4.Msg {
		if err != nil {
			log.Println(err)
		}

		return msg
	}

	go func() {
		for {
			msg := logErr(sub.Recv())

			if len(msg.Frames) > 2 {
				var blockHashCounter uint32
				blockHashCounter |= uint32(msg.Frames[2][0])
				blockHashCounter |= uint32(msg.Frames[2][1]) << 8
				blockHashCounter |= uint32(msg.Frames[2][2]) << 16
				blockHashCounter |= uint32(msg.Frames[2][3]) << 24

				hashBlockChannel <- hashBlockResponse{
					blockChainName:    blockChainName,
					previousBlockHash: hex.EncodeToString(msg.Frames[1]),
					blockHashCounter:  blockHashCounter,
				}
			}

		}
	}()

	return sub, nil
}

func (p *PoolServer) CheckAndRecoverRPCs() error {
	var err error
	for coin, manager := range p.rpcManagers {
		err = manager.CheckAndRecoverRPCs()
		if err != nil {
			coinError := errors.New(coin)
			return errors.Join(coinError, err)
		}
	}
	return nil
}

package pool

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"designs.capital/dogepool/bitcoin"
	"designs.capital/dogepool/config"
	"designs.capital/dogepool/persistence"
	"designs.capital/dogepool/rpc"
)

type PoolServer struct {
	sync.RWMutex
	config            *config.Config
	activeNodes       BlockChainNodesMap
	rpcManagers       map[string]*rpc.Manager
	connectionTimeout time.Duration
	templates         Pair
	workCache         bitcoin.Work
	shareBuffer       []persistence.Share
}

func NewServer(cfg *config.Config, rpcManagers map[string]*rpc.Manager) *PoolServer {
	if len(cfg.PoolName) < 1 {
		log.Println("Pool must have a name")
	}
	if len(cfg.BlockchainNodes) < 1 {
		log.Println("Pool must have at least 1 blockchain node to work from")
	}
	if len(cfg.BlockChainOrder) < 1 {
		log.Println("Pool must have a blockchain order to tell primary vs aux")
	}

	pool := &PoolServer{
		config:      cfg,
		rpcManagers: rpcManagers,
	}

	return pool
}

func (pool *PoolServer) Start() {
	initiateSessions()
	pool.loadBlockchainNodes()
	pool.startBufferManager()

	// Add logging for initialization
	log.Printf("Pool server starting with config: %+v", pool.config)
	log.Printf("Active nodes: %+v", pool.activeNodes)

	amountOfChains := len(pool.config.BlockChainOrder) - 1
	pool.templates.AuxBlocks = make([]bitcoin.AuxBlock, amountOfChains)

	// Initial work creation
	log.Println("Creating initial work template...")
	panicOnError(pool.fetchRpcBlockTemplatesAndCacheWork())
	work, err := pool.generateWorkFromCache(false)
	panicOnError(err)
	log.Printf("Initial work template created: %+v", work)

	go pool.listenForConnections()
	pool.broadcastWork(work)

	// There after..
	panicOnError(pool.listenForBlockNotifications())
}

func (pool *PoolServer) handleConnection(conn net.Conn) {
	// Add connection details logging
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New connection details - Local: %v, Remote: %v", 
	    conn.LocalAddr().String(), remoteAddr)

	client := &stratumClient{
		connection: conn,
		ip:         strings.Split(remoteAddr, ":")[0],
	}

	// Generate unique extranonce for the client
	client.extranonce1 = generateExtranonce()
	log.Printf("Generated extranonce1 for client %s: %s", client.ip, client.extranonce1)

	// Add error handling and logging for client setup
	if err := pool.setupClient(client); err != nil {
		log.Printf("Error setting up client %s: %v", client.ip, err)
		conn.Close()
		return
	}

	go pool.handleClient(client)
}

func (pool *PoolServer) handleClient(client *stratumClient) {
	defer func() {
		log.Printf("Client disconnecting - IP: %s, Worker: %s, Session: %s", 
		    client.ip, client.login, client.sessionID)
		removeSession(client)
		client.connection.Close()
	}()

	reader := bufio.NewReader(client.connection)
	for {
		message, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client %s: %v", client.ip, err)
			}
			return
		}

		// Log raw message for debugging
		log.Printf("Received message from %s: %s", client.ip, string(message))

		err = pool.respondToStratumClient(client, message)
		if err != nil {
			log.Printf("Error handling message from %s: %v", client.ip, err)
			return
		}
	}
}

func (pool *PoolServer) broadcastWork(work bitcoin.Work) {
	request := miningNotify(work)
	err := notifyAllSessions(request)
	logOnError(err)
}

func (p *PoolServer) fetchAllBlockTemplatesFromRPC() (bitcoin.Template, *bitcoin.AuxBlock, error) {
	var template bitcoin.Template
	var err error
	response, err := p.GetPrimaryNode().RPC.GetBlockTemplate()
	if err != nil {
		return template, nil, errors.New("RPC error: " + err.Error())
	}

	err = json.Unmarshal(response, &template)
	if err != nil {
		return template, nil, err
	}

	var auxBlock bitcoin.AuxBlock

	if p.config.GetAux1() != "" {
		response, err = p.GetAux1Node().RPC.CreateAuxBlock(p.GetAux1Node().RewardTo)
		if err != nil {
			log.Println("No aux block found: " + err.Error())
			return template, nil, nil
		}

		err = json.Unmarshal(response, &auxBlock)
		if err != nil {
			return template, nil, err
		}
	}

	return template, &auxBlock, nil
}

func notifyAllSessions(request stratumRequest) error {
	for _, client := range sessions {
		err := sendPacket(request, client)
		logOnError(err)
	}
	log.Printf("Sent work to %v client(s)", len(sessions))
	return nil
}

func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

func logOnError(e error) {
	if e != nil {
		log.Println(e)
	}
}

func logFatalOnError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

package bitcoin

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

type BlockGenerator interface {
	MakeHeader(extranonce, nonce string) (string, error) // On aux generation, on work verfication, and possibily even work submission
	Header() string
	Sum() (*big.Int, error)  // On work verification, many, more than than header generation
	Submit() (string, error) // On submission
}

var jobCounter int

func GenerateWork(template *Template, auxBlock *AuxBlock, chainName, arbitrary, poolPayoutPubScriptKey string, reservedArbitraryByteLength int) (*BitcoinBlock, Work, error) { // On trigger
	if template == nil {
		return nil, nil, errors.New("Template cannot be null")
	}

	var err error
	block := BitcoinBlock{}

	block.init(GetChain(chainName))
	block.Template = template

	block.ReversePrevBlockHash, err = reverseHex4Bytes(block.Template.PrevBlockHash)
	if err != nil {
		m := "invalid previous block hash hex: " + err.Error()
		return nil, nil, errors.New(m)
	}

	arbitraryBytes := bytesWithLengthHeader([]byte(arbitrary))
	arbitraryByteLength := uint(len(arbitraryBytes) + reservedArbitraryByteLength)
	arbitraryHex := hex.EncodeToString(arbitraryBytes)

	block.CoinbaseInitial = block.Template.CoinbaseInitial(arbitraryByteLength).Serialize()
	block.CoinbaseFinal = arbitraryHex + block.Template.CoinbaseFinal(poolPayoutPubScriptKey).Serialize()
	block.MerkleSteps, err = block.Template.MerkleSteps()
	if err != nil {
		return nil, nil, err
	}

	work := make(Work, 8)
	work[0] = fmt.Sprintf("%08x", jobCounter) // Job ID
	work[1] = block.ReversePrevBlockHash
	work[2] = block.CoinbaseInitial
	work[3] = block.CoinbaseFinal
	work[4] = block.MerkleSteps
	work[5] = fmt.Sprintf("%08x", block.Template.Version)
	work[6] = block.Template.Bits
	work[7] = fmt.Sprintf("%x", block.Template.CurrentTime)

	jobCounter++

	return &block, work, nil
}

func (b *BitcoinBlock) MakeHeader(extranonce, nonce, nonceTime string) (string, error) {
	if b.Template == nil {
		return "", errors.New("generate work first")
	}

	var err error
	coinbase := Coinbase{
		CoinbaseInital: b.CoinbaseInitial,
		Arbitrary:      extranonce,
		CoinbaseFinal:  b.CoinbaseFinal,
	}

	b.Coinbase = coinbase.Serialize()
	coinbaseHashed, err := b.CoinbaseHashed()
	if err != nil {
		return "", err
	}

	merkleRoot, err := makeHeaderMerkleRoot(coinbaseHashed, b.MerkleSteps)
	if err != nil {
		return "", err
	}

	t := b.Template

	b.Header, err = blockHeader(uint(t.Version), t.PrevBlockHash, merkleRoot, nonceTime, t.Bits, nonce)

	if err != nil {
		return "", err
	}

	return b.Header, nil
}

func (b *BitcoinBlock) HeaderHashed() (string, error) {
	// TODO - break out headerdigest vs blockdigest
	header, err := b.Chain.CoinbaseDigest(b.Header)
	if err != nil {
		return "", err
	}
	// Not sure if this is for litecoin only, but..
	header, err = reverseHexBytes(header)
	if err != nil {
		return "", err
	}
	return header, nil
}

func (b *BitcoinBlock) CoinbaseHashed() (string, error) {
	return b.Chain.CoinbaseDigest(b.Coinbase)
}

func (b *BitcoinBlock) Sum() (*big.Int, error) {
	if b.Chain == nil {
		return nil, errors.New("calculateSum: Missing blockchain interface")
	}
	if b.Header == "" {
		return nil, errors.New("generate header first")
	}

	digest, err := b.Chain.HeaderDigest(b.Header)
	if err != nil {
		return nil, err
	}

	digest, err = reverseHexBytes(digest)
	if err != nil {
		return nil, err
	}

	b.Hash = digest  // Changed from b.hash to b.Hash

	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(digestBytes), nil
}

func (b *BitcoinBlock) Submit() (string, error) {
	if b.Header == "" {  // Changed from b.header to b.Header
		return "", errors.New("generate header first")
	}

	transactionPool := make([]string, len(b.Template.Transactions))
	for i, transaction := range b.Template.Transactions {
		transactionPool[i] = transaction.Data
	}

	submission := b.createSubmissionHex()

	if b.Template.MimbleWimble != "" {
		submission = submission + "01" + b.Template.MimbleWimble
	}

	return submission, nil
}

func debugMerkleSteps(block BitcoinBlock) {
	fmt.Println()
	fmt.Println("Transaction IDs")
	for i, transaction := range block.Template.Transactions {
		fmt.Println(i+1, transaction.ID, " : ", transaction.Data)
	}

	fmt.Println()
	fmt.Println("Steps")
	for i, step := range block.MerkleSteps {  // Changed from block.merkleSteps to block.MerkleSteps
		fmt.Println(i+1, step)
	}
	fmt.Println()
}

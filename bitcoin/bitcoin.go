package bitcoin

import (
	"log"
)

type BitcoinBlock struct {
	Template             *Template
	ReversePrevBlockHash string
	CoinbaseInitial      string
	CoinbaseFinal        string
	MerkleSteps          []string
	Coinbase             string
	Header               string
	Hash                 string
	Chain                Blockchain
}

func (b BitcoinBlock) ChainName() string {
	if b.Chain == nil {
		panic("Chain needs to be set")
	}
	return b.Chain.ChainName()
}

func (b *BitcoinBlock) init(chain Blockchain) {
	if chain == nil {
		panic("Chain cannot be null")
	}
	b.Chain = chain
}

func (b *BitcoinBlock) ToHex() string {
    submission, err := b.Submit()
    if err != nil {
        log.Printf("Error converting block to hex: %v", err)
        return ""
    }
    return submission
}

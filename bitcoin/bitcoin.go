import (
	"log"
)
package bitcoin

type BitcoinBlock struct {
	Template             *Template
	reversePrevBlockHash string
	coinbaseInitial      string
	coinbaseFinal        string
	merkleSteps          []string
	coinbase             string
	header               string
	hash                 string
	chain                Blockchain
}

func (b BitcoinBlock) ChainName() string {
	if b.chain == nil {
		panic("Chain needs to be set")
	}
	return b.chain.ChainName()
}

func (b *BitcoinBlock) init(chain Blockchain) {
	if chain == nil {
		panic("Chain cannot be null")
	}
	b.chain = chain
}

func (b *BitcoinBlock) ToHex() string {
    submission, err := b.Submit()
    if err != nil {
        log.Printf("Error converting block to hex: %v", err)
        return ""
    }
    return submission
}

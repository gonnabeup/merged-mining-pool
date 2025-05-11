package bitcoin

import "fmt"

const (
	mergedMiningHeader  = "fabe6d6d"
	mergedMiningTrailer = "010000000000000000002632"
)

type AuxBlock struct {
	Hash              string `json:"hash"`
	ChainID           int    `json:"chainid"`
	PreviousBlockHash string `json:"previousblockhash"`
	CoinbaseHash      string `json:"coinbasehash"`
	CoinbaseValue     uint   `json:"coinbasevalue"`
	Bits              string `json:"bits"`
	Height            uint64 `json:"height"`
	Target            string `json:"target"`
}

func (b *AuxBlock) GetWork() string {
	return mergedMiningHeader + b.Hash + mergedMiningTrailer
}

type AuxPow struct {
	ParentCoinbase   string
	ParentHeaderHash string
	ParentMerkleBranch
	auxMerkleBranch      AuxMerkleBranch
	ParentHeaderUnhashed string
}

func MakeAuxPow(parentBlock BitcoinBlock) AuxPow {
	if parentBlock.Hash == "" {
		panic("Set parent block hash first")
	}

	return AuxPow{
		ParentCoinbase:       parentBlock.Coinbase,
		ParentHeaderHash:     parentBlock.Hash,
		ParentMerkleBranch:   makeParentMerkleBranch(parentBlock.MerkleSteps),
		auxMerkleBranch:      makeAuxChainMerkleBranch(),
		ParentHeaderUnhashed: parentBlock.Header,
	}
}

func (p *AuxPow) Serialize() string {
	return p.ParentCoinbase +
		p.ParentHeaderHash +
		p.ParentMerkleBranch.Serialize() +
		p.auxMerkleBranch.Serialize() +
		p.ParentHeaderUnhashed
}

type ParentMerkleBranch struct {
	Length uint
	Items  []string
	mask   string
}

func makeParentMerkleBranch(items []string) ParentMerkleBranch {
	length := uint(len(items))
	return ParentMerkleBranch{
		Length: length,
		Items:  items,
		mask:   "00000000",
	}
}

func (pm *ParentMerkleBranch) Serialize() string {
	items := ""
	for _, item := range pm.Items {
		items = items + item
	}
	return varUint(pm.Length) + items + pm.mask
}

type AuxMerkleBranch struct {
	numberOfBranches string
	mask             string
}

func makeAuxChainMerkleBranch() AuxMerkleBranch {
	return AuxMerkleBranch{
		numberOfBranches: "00",
		mask:             "00000000",
	}
}

func (am *AuxMerkleBranch) Serialize() string {
	return am.numberOfBranches + am.mask
}

func debugAuxPow(parentBlock BitcoinBlock, parentMerkle ParentMerkleBranch, auxchainMerkle AuxMerkleBranch) {
	fmt.Println()
	fmt.Println("coinbase", parentBlock.Coinbase)
	fmt.Println("hash", parentBlock.Hash)
	fmt.Println("merkleSteps", parentBlock.MerkleSteps)
	fmt.Println("merkleDigested", parentMerkle.Serialize())
	fmt.Println("chainmerklebranch", auxchainMerkle.Serialize())
	fmt.Println("header", parentBlock.Header)
	fmt.Println()
}

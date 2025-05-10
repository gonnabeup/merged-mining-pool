package bitcoin

import (
	"regexp"
)

type Digibyte struct{}

func (Digibyte) ChainName() string {
	return "digibyte"
}

func (Digibyte) CoinbaseDigest(coinbase string) (string, error) {
	return DoubleSha256(coinbase)
}

func (Digibyte) HeaderDigest(header string) (string, error) {
	return DoubleSha256(header)
}

func (Digibyte) ShareMultiplier() float64 {
	return 65536
}

func (Digibyte) ValidMainnetAddress(address string) bool {
	return regexp.MustCompile("^(D|S)[A-Za-z0-9]{33}$|^dgb1[0-9A-Za-z]{39}$").MatchString(address)
}

func (Digibyte) ValidTestnetAddress(address string) bool {
	return regexp.MustCompile("^(n|m|t)[a-km-zA-HJ-NP-Z1-9]{33}$").MatchString(address)
}

func (Digibyte) MinimumConfirmations() uint {
	return uint(100)
}
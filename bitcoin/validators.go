package bitcoin

func (b BitcoinBlock) ValidateMainnetAddress(address string) bool {
    return b.Chain.ValidMainnetAddress(address)
}

func (b BitcoinBlock) ValidateTestnetAddress(address string) bool {
    return b.Chain.ValidTestnetAddress(address)
}

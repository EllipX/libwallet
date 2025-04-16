package wlttest

import (
	"github.com/EllipX/libwallet/wltnet"
	"time"
)

func getTestNetwork() *wltnet.Network {
	return &wltnet.Network{
		Id:               nil,
		Type:             "evm",
		ChainId:          "1",
		Name:             "Ethereum Mainnet",
		RPC:              "",
		CurrencySymbol:   "ETH",
		CurrencyDecimals: 0,
		BlockExplorer:    "",
		TestNet:          false,
		Priority:         0,
		Created:          time.Now(),
		Updated:          time.Now(),
	}
}

type MockAddressProvider struct {
	MockAddress string
}

func (m *MockAddressProvider) GetAddress() string {
	return m.MockAddress
}

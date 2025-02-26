package wlttoken

import (
	"fmt"
	"git.atonline.com/ellipx/libwallet/wltnet"
	"testing"
	"time"
)

func getNetwork() *wltnet.Network {
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

func TestDiscoverToken(t *testing.T) {
	n := getNetwork()
	// USDT
	token, err := DiscoverToken(n, "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err != nil {
		t.Fatalf("Error DiscoverToken USDT: %v", err)
	}
	fmt.Println(token)
	// BNB
	token, err = DiscoverToken(n, "0xB8c77482e45F1F44dE1745F52C74426C631bDD52")
	if err != nil {
		t.Fatalf("Error DiscoverToken BNB: %v", err)
	}
	fmt.Println(token)
	// SHIBA INU (SHIB)
	token, err = DiscoverToken(n, "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE")
	if err != nil {
		t.Fatalf("Error DiscoverToken SHIBA INU: %v", err)
	}
	fmt.Println(token)
	// SEI (SEI)
	token, err = DiscoverToken(n, "0xbdF43ecAdC5ceF51B7D1772F722E40596BC1788B")
	if err != nil {
		t.Fatalf("Error DiscoverToken SEI: %v", err)
	}
	fmt.Println(token)

	fmt.Println("SUCCESSSSSS")
}

package wlttest

import (
	"git.atonline.com/ellipx/libwallet/wltnet"
	"testing"
)

func TestDiscoverToken(t *testing.T) {
	n := getTestNetwork()
	// USDT
	_, err := wltnet.DiscoverToken(n, "0xdAC17F958D2ee523a2206206994597C13D831ec7")
	if err != nil {
		t.Fatalf("Error DiscoverToken USDT: %v", err)
	}
	// BNB
	_, err = wltnet.DiscoverToken(n, "0xB8c77482e45F1F44dE1745F52C74426C631bDD52")
	if err != nil {
		t.Fatalf("Error DiscoverToken BNB: %v", err)
	}
	// SHIBA INU (SHIB)
	_, err = wltnet.DiscoverToken(n, "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE")
	if err != nil {
		t.Fatalf("Error DiscoverToken SHIBA INU: %v", err)
	}
	// SEI (SEI)
	_, err = wltnet.DiscoverToken(n, "0xbdF43ecAdC5ceF51B7D1772F722E40596BC1788B")
	if err != nil {
		t.Fatalf("Error DiscoverToken SEI: %v", err)
	}
	fmt.Println(token)

	fmt.Println("SUCCESSSSSS")
}

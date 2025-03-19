package wlttest

import (
	"fmt"
	"testing"

	"github.com/EllipX/libwallet/wltbase"
	"github.com/EllipX/libwallet/wltintf"
)

func TestNftMetadata(t *testing.T) {
	v, err := wltbase.InitEnv("test")
	if err != nil {
		t.Fatal(err)
	}

	env, ok := v.(wltintf.Env)
	if !ok {
		fmt.Println("Failed to cast to wltintf.Env")
		return
	}

	network := getTestNetwork()

	// this address uses HTTP token URI
	_, err = network.NftMetadata(env, "0x3E34ff1790BF0a13EfD7d77e75870Cb525687338", "1")
	if err != nil {
		t.Fatalf("Error getting NFT list from 1 : %v", err)
	}

	// this address uses IPFS token URI
	_, err = network.NftMetadata(env, "0xBd3531dA5CF5857e7CfAA92426877b022e612cf8", "1")
	if err != nil {
		t.Fatalf("Error getting NFT list from 2 : %v", err)
	}
}

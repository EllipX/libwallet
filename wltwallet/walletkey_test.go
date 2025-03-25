package wltwallet

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/KarpelesLab/spotlib"
)

func TestWalletPeers(t *testing.T) {
	log.Printf("connecting to spot...")
	spot, err := spotlib.New()
	if err != nil {
		t.Fatalf("failed to get: %s", err)
		return
	}
	spot.WaitOnline(context.Background())
	log.Printf("spot is now online")
	time.Sleep(time.Second / 2)

	res, err := selectPeer(context.Background(), spot)
	if err != nil {
		t.Errorf("failed to select peer: %s", err)
	}
	log.Printf("peer = %+v", res)
}

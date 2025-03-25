package wltacct

import (
	"log"
	"time"

	"github.com/EllipX/libwallet/wltintf"
	"github.com/EllipX/libwallet/wltwallet"
	"github.com/KarpelesLab/emitter"
	"github.com/KarpelesLab/xuid"
)

func Init(e wltintf.Env) {
	go handleWalletDelete(e, e.Emitter().On("wallet:delete"))
	go handleWalletRestore(e, e.Emitter().On("wallet:restored"))
}

func handleWalletRestore(e wltintf.Env, ch <-chan *emitter.Event) {
	for ev := range ch {
		// create an account for this new wallet
		wallet, err := emitter.Arg[*wltwallet.Wallet](ev, 0)
		if err != nil {
			log.Printf("failed to fetch wallet in wallet:restored: %s", err)
			continue
		}
		newAcct := &Account{
			Id:        xuid.New("acct"),
			Name:      "Restored Account",
			Chaincode: wallet.Chaincode,
			Wallet:    wallet.Id,
			Type:      "ethereum",
			Created:   time.Now(),
		}
		err = newAcct.init(wallet)
		if err != nil {
			log.Printf("failed to init account: %s", err)
			continue
		}

		err = newAcct.save(e)
		if err != nil {
			log.Printf("failed to save account: %s", err)
		}

	}
}

func handleWalletDelete(e wltintf.Env, ch <-chan *emitter.Event) {
	for ev := range ch {
		// delete each account
		var accts []*Account
		e.Find(&accts, map[string]any{"Wallet": ev.Args[0]})

		for _, acct := range accts {
			err := acct.accountDelete(e)
			if err != nil {
				log.Printf("failed to cascade delete account: %s", err)
			}
		}
	}
}

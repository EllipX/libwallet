package wltwallet

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/EllipX/libwallet/wltcrash"
	"github.com/EllipX/libwallet/wltintf"
	"github.com/EllipX/libwallet/wltsign"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/xuid"
	"github.com/ModChain/secp256k1"
	"github.com/ModChain/tss-lib/v2/common"
	"github.com/ModChain/tss-lib/v2/ecdsa/keygen"
	"github.com/ModChain/tss-lib/v2/ecdsa/signing"
	"github.com/ModChain/tss-lib/v2/tss"
)

// Wallet represents a multi-signature wallet with threshold signature scheme (TSS) support
// It can contain multiple keys with a configurable threshold for signatures
type Wallet struct {
	Id        *xuid.XUID   `gorm:"primaryKey"` // Unique identifier for the wallet
	Name      string       // User-friendly name
	Curve     string       // Elliptic curve used (e.g., "secp256k1")
	Threshold int          // Minimum number of keys required for signing
	Keys      []*WalletKey `gorm:"-:all"` // Associated keys (not stored in database)
	Pubkey    string       // Base64 encoded public key
	Chaincode string       // Base64 encoded chaincode for HD wallet derivation
	Created   time.Time    `gorm:"autoCreateTime"` // Creation timestamp
	Modified  time.Time    `gorm:"autoUpdateTime"` // Last modification timestamp
}

// save persists the wallet and all its keys to the database
// Returns any error encountered during the operation
func (w *Wallet) save(e wltintf.Env) error {
	for _, wk := range w.Keys {
		err := wk.save(e)
		if err != nil {
			return err
		}
	}
	return e.Save(w)
}

// ApiUpdate handles API requests to update wallet properties
// Currently supports updating the wallet name
// Returns nil if no updates were made or any error encountered during saving
func (w *Wallet) ApiUpdate(ctx *apirouter.Context) error {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return errors.New("failed to get env")
	}

	updated := false

	if v, ok := apirouter.GetParam[string](ctx, "Name"); ok {
		w.Name = v
		updated = true
	}
	if !updated {
		return nil
	}
	w.Modified = time.Now()
	return w.save(e)
}

// ApiDelete handles API requests to delete a wallet
// Emits a "wallet:deleted" event and removes the wallet and its keys from the database
// Returns any error encountered during the deletion
func (w *Wallet) ApiDelete(ctx *apirouter.Context) error {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return errors.New("failed to get env")
	}

	e.Emitter().Emit(ctx, "wallet:deleted", w.Id.String())

	// delete Wallet/Key entries
	e.DeleteWhere(&WalletKey{}, map[string]any{"Wallet": w.Id.String()})
	//e.sql.Where(map[string]any{"Wallet": w.Id.String()}).Delete(&WalletKey{})
	return e.Delete(w)
}

// initializeWallet creates a new wallet with the specified key descriptions
// Implements Threshold Signature Scheme (TSS) for distributed key generation
// Parameters:
//   - ctx: context for progress reporting and cancellation
//   - kDesc: array of key descriptions for wallet creation
//
// Returns any error encountered during wallet initialization
func (w *Wallet) initializeWallet(ctx context.Context, kDesc []*wltsign.KeyDescription) error {
	if w.Threshold == 0 {
		w.Threshold = 1
	}
	nk := len(kDesc)
	w.Keys = make([]*WalletKey, nk)

	if nk == 0 {
		return errors.New("at least one key is required")
	}
	if w.Threshold >= nk {
		return errors.New("threshold too high")
	}
	if w.Threshold < 0 {
		return errors.New("threshold too low")
	}

	// Create wallet keys for each key description
	for i, kInfo := range kDesc {
		switch kInfo.Type {
		case "StoreKey", "Plain", "RemoteKey", "Password":
			// OK
		default:
			return fmt.Errorf("unsupported key type %s for key #%d", kInfo.Type, i+1)
		}
		log.Printf("generating key %d/%d", i, nk)
		apirouter.Progress(ctx, map[string]any{"count": nk + 1, "running": i + 1})

		k, err := w.createWalletKey(ctx, kInfo.Type)
		if err != nil {
			return err
		}
		w.Keys[i] = k
	}

	log.Printf("producing final")

	// Perform final operation (actual key generation)
	apirouter.Progress(ctx, map[string]any{"count": nk + 1, "running": nk + 1})

	// Set up TSS parties for distributed key generation
	var ids tss.UnSortedPartyIDs
	m := make(map[string]tssPartyUpdateOnly)
	idmap := make(map[int]*tss.PartyID)
	for n, p := range w.Keys {
		key := new(big.Int).SetBytes(p.Id.UUID[:])
		id := tss.NewPartyID(p.Id.String(), p.Id.String(), key)
		ids = append(ids, id)
		idmap[n] = id
	}
	sids := tss.SortPartyIDs(ids)

	curve := tss.EC()
	tssctx := tss.NewPeerContext(sids)

	// Create channels for TSS communication
	outCh := make(chan tss.Message)
	defer close(outCh)
	var wg sync.WaitGroup
	wg.Add(len(w.Keys))

	// Start TSS key generation for each party
	for n, p := range w.Keys {
		endCh := make(chan *keygen.LocalPartySaveData)
		params := tss.NewParameters(curve, tssctx, idmap[n], nk, w.Threshold)
		party := keygen.NewLocalParty(params, outCh, endCh, *p.pre)
		m[p.Id.String()] = party
		go func(p *WalletKey) {
			defer wg.Done()
			err := party.Start()
			if err != nil {
				log.Printf("err = %s", err)
			}
			p.sdata = <-endCh
		}(p)
	}
	go tssRouter(ctx, m, outCh)

	// Generate random chaincode for HD wallet derivation
	chaincode := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, chaincode)
	if err != nil {
		return err
	}

	// Wait for all key generation to complete
	wg.Wait()

	// Set wallet properties from generated keys
	pk := w.Keys[0].sdata.ECDSAPub.ToSecp256k1PubKey()
	w.Pubkey = base64.RawURLEncoding.EncodeToString(pk.SerializeCompressed())
	w.Chaincode = base64.RawURLEncoding.EncodeToString(chaincode)
	w.Curve = curve.Params().Name

	// Encrypt keys with their respective key descriptions
	for i, kInfo := range kDesc {
		err = w.Keys[i].encrypt(kInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

// getKey retrieves a WalletKey by its ID string
// Returns the key if found, or nil if not found
func (w *Wallet) getKey(id string) *WalletKey {
	for _, k := range w.Keys {
		if k.Id.String() == id {
			return k
		}
	}
	return nil
}

// Sign the digest using the wallet, returning a DER encoded signature
// Implements the crypto.Signer interface
// Parameters:
//   - rand: random source (not used in TSS signatures)
//   - digest: the hash or message to sign
//   - opts: must be *wltsign.Opts containing context and key information
//
// Returns the signature and any error encountered
// Has panic recovery to prevent crashes during signature generation
func (w *Wallet) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (dat []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			// TODO might want to find a way to get the crash log
			if aopt, ok := opts.(*wltsign.Opts); ok {
				id := wltcrash.Log(aopt.Context, e, "signature main thread")
				log.Printf("panic: %s", e)
				err = fmt.Errorf("panic during signature generation, please contact support (crash id %s)", id)
			}
		}
	}()
	dat, err = w.subSign(rand, digest, opts)
	return
}

// subSign performs the actual distributed signature operation using TSS
// This is called by Sign after setting up panic recovery
// Parameters:
//   - rand: random source (not used in TSS signatures)
//   - digest: the hash or message to sign
//   - opts: must be *wltsign.Opts containing context, key information, and IL (intermediate value)
//
// Returns the DER-encoded signature and any error encountered
func (w *Wallet) subSign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if w.Threshold == 0 {
		w.Threshold = 1
	}
	aopt, ok := opts.(*wltsign.Opts)
	if !ok {
		return nil, errors.New("sign requires appropriate options")
	}
	msg := new(big.Int).SetBytes(digest)
	keys := aopt.Keys

	// Prepare party IDs for TSS signing
	var ids tss.UnSortedPartyIDs
	m := make(map[string]tssPartyUpdateOnly)
	idmap := make(map[int]*tss.PartyID)
	for n, kd := range keys {
		p := w.getKey(kd.Id)
		if p == nil {
			return nil, fmt.Errorf("could not find key id=%s", kd.Id)
		}
		key := new(big.Int).SetBytes(p.Id.UUID[:])
		id := tss.NewPartyID(p.Id.String(), p.Id.String(), key)
		ids = append(ids, id)
		idmap[n] = id
	}
	sids := tss.SortPartyIDs(ids)

	// Get the correct curve for the wallet
	curve, ok := tss.GetCurveByName(tss.CurveName(w.Curve))
	if !ok {
		return nil, fmt.Errorf("unknown curve %s", w.Curve)
	}
	tssctx := tss.NewPeerContext(sids)

	// Create channels for TSS communication
	outCh := make(chan tss.Message)
	defer close(outCh)
	res := make(chan any, len(keys))

	// Start TSS signing parties
	for n, kd := range keys {
		p := w.getKey(kd.Id)
		if p == nil {
			return nil, fmt.Errorf("could not find key id=%s", kd.Id)
		}
		endCh := make(chan *common.SignatureData)
		params := tss.NewParameters(curve, tssctx, idmap[n], len(keys), w.Threshold)
		sdata, err := p.decrypt(kd, keySignPurpose)
		if err != nil {
			return nil, err
		}
		party := signing.NewLocalPartyWithAutoKDD(msg, params, *sdata, aopt.IL, outCh, endCh, len(digest))
		m[p.Id.String()] = party
		go func(p *WalletKey) {
			defer func() {
				wltcrash.Log(aopt.Context, recover(), "signing party thread")
			}()
			err := party.Start()
			if err != nil {
				log.Printf("err = %s", err)
				res <- err
			}
			sig := <-endCh
			res <- sig.GetSignatureObject().Serialize()
		}(p)
	}
	go tssRouter(aopt.Context, m, outCh)

	// Set a timeout for the signing operation
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	// Wait for result or timeout
	select {
	case final := <-res:
		switch v := final.(type) {
		case error:
			return nil, v
		case []byte:
			return v, nil
		default:
			return nil, fmt.Errorf("invalid data type %T", v)
		}
	case <-timer.C:
		return nil, fmt.Errorf("signature operation timed out")
	}
}

// GetPubkey returns the wallet's public key as a secp256k1.PublicKey object
// Decodes the base64-encoded public key stored in the wallet
// Returns the public key and any error encountered during decoding
func (w *Wallet) GetPubkey() (*secp256k1.PublicKey, error) {
	dat, err := base64.RawURLEncoding.DecodeString(w.Pubkey)
	if err != nil {
		return nil, err
	}
	return secp256k1.ParsePubKey(dat)
}

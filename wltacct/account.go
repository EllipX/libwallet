package wltacct

import (
	"context"
	"crypto"
	"encoding/base64"
	"errors"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/EllipX/libwallet/wltintf"
	"github.com/EllipX/libwallet/wltnet"
	"github.com/EllipX/libwallet/wltsign"
	"github.com/EllipX/libwallet/wltwallet"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/pobj"
	"github.com/KarpelesLab/xuid"
	"github.com/ModChain/outscript"
	"github.com/ModChain/secp256k1"
	"github.com/ModChain/secp256k1/ecckd"
)

// standard for derivation path:
// m / purpose' / coin_type' / account' / change / index

// Account represents a blockchain account derived from a wallet's public key
// Using hierarchical deterministic (HD) derivation to generate addresses for different chains
type Account struct {
	Id        *xuid.XUID `gorm:"primaryKey"` // Unique identifier for the account
	Wallet    *xuid.XUID // Parent wallet ID
	Name      string     // User-friendly name
	Index     int        // Account index, starts at zero
	Type      string     // "ethereum", "bitcoin", etc *deprecated* we don't care about the account type, only the wallet curve
	Path      string     // Derivation path, e.g. m/44/60/0/0 (note: no hardened keys since we only have public keys)
	Address   string     // Blockchain address in the appropriate format
	URI       string     // URI for sending to this account (e.g. ethereum:0x...)
	Pubkey    string     // Base64 encoded public key
	Chaincode string     // Base64 encoded chaincode for HD derivation
	IL        *big.Int   `json:"IL,string" gorm:"serializer:json"` // Intermediate value used in derivation
	Created   time.Time  `gorm:"autoCreateTime"`                   // Creation timestamp
	Updated   time.Time  `gorm:"autoUpdateTime"`                   // Last update timestamp
}

// save persists the account to the database
// Returns any error encountered during the operation
func (a *Account) save(e wltintf.Env) error {
	return e.Save(a)
}

// check ensures the account has proper chaincode and updates address/URI based on the current network
// Handles different network types (evm, bitcoin) and formats addresses accordingly
// Returns any error encountered during the operation
func (a *Account) check(e wltintf.Env) error {
	// Ensure chaincode is present, get from wallet if missing
	if a.Chaincode == "" {
		wlt, err := a.getWallet(e)
		if err != nil {
			return err
		}
		a.Chaincode = wlt.Chaincode
		a.save(e)
	}

	// Get current network for proper address formatting
	net, err := wltnet.CurrentNetwork(e)
	if err != nil {
		return err
	}

	switch net.Type {
	case "evm":
		// Format Ethereum address
		addr, err := outscript.New(a.PublicKey()).Out("eth").Address()
		if err != nil {
			return err
		}
		a.Address = addr
		a.URI = "ethereum:" + a.Address
		return nil
	case "bitcoin":
		// For Bitcoin-based chains, derive a child key at m/0
		pub, err := a.DerivePublic("m/0")
		if err != nil {
			return err
		}
		s := outscript.New(pub)

		// Format address based on specific blockchain
		switch net.ChainId {
		case "bitcoin":
			// Bitcoin uses SegWit (p2wpkh)
			addr, err := s.Out("p2wpkh").Address("bitcoin")
			if err != nil {
				return err
			}
			a.Address = addr
			a.URI = "bitcoin:" + addr
			return nil
		case "bitcoin-cash":
			// Bitcoin Cash uses legacy format (p2pkh)
			addr, err := s.Out("p2pkh").Address("bitcoincash")
			if err != nil {
				return err
			}
			a.Address = addr
			a.URI = addr
			return nil
		case "dogecoin":
			// Dogecoin uses legacy format (p2pkh)
			addr, err := s.Out("p2pkh").Address("dogecoin")
			if err != nil {
				return err
			}
			a.Address = addr
			a.URI = "dogecoin:" + addr
			return nil
		case "litecoin":
			// Litecoin uses SegWit (p2wpkh)
			addr, err := s.Out("p2wpkh").Address("litecoin")
			if err != nil {
				return err
			}
			a.Address = addr
			a.URI = "litecoin:" + addr
			return nil
		}
		fallthrough
	default:
		// Default case for unsupported networks
		a.Address = "N/A"
		a.URI = ""
		return nil
	}
}

// setCurrent sets this account as the current active account in the environment
// Returns any error encountered during the operation
func (a *Account) setCurrent(e wltintf.Env) error {
	return e.SetCurrent("account", a.Id.String())
}

// init initializes a new account with a specified wallet and index
// Derives the account's public key and addresses from the wallet's master key
// Uses the BIP44 path format: m/44/60/0/{index} (Ethereum-like for now)
// Returns any error encountered during the initialization
func (a *Account) init(wallet *wltwallet.Wallet) error {
	// Set up derivation path for Ethereum (hardcoded for now)
	a.Path = "m/44/60/0/" + strconv.Itoa(a.Index)
	a.Chaincode = wallet.Chaincode

	// Get the wallet's master public key
	wpubkey, err := wallet.GetPubkey()
	if err != nil {
		return err
	}

	// Decode chaincode from base64
	chainCode, err := base64.RawURLEncoding.DecodeString(wallet.Chaincode)
	if err != nil {
		return err
	}

	// Derive the account's public key using HD wallet derivation
	IL, pubkey, err := DerivePublicKey(wpubkey, chainCode, a.Path)
	if err != nil {
		return err
	}

	// Store the IL (intermediate value) and public key
	a.IL = IL
	a.Pubkey = base64.RawURLEncoding.EncodeToString(pubkey.SerializeCompressed())

	// Default to Ethereum address format
	s := outscript.New(pubkey)
	addr, err := s.Out("eth").Address()
	if err != nil {
		return err
	}
	a.Address = addr
	a.URI = "ethereum:" + a.Address
	return nil
}

// getWallet retrieves the parent wallet of this account
// Returns the wallet object and any error encountered
func (a *Account) getWallet(e wltintf.Env) (*wltwallet.Wallet, error) {
	return wltwallet.WalletById(e, a.Wallet)
}

// PublicKey returns the account's public key as a secp256k1.PublicKey object
// Decodes the base64-encoded public key stored in the account
// Returns nil if there's an error during decoding
func (a *Account) PublicKey() *secp256k1.PublicKey {
	k, err := base64.RawURLEncoding.DecodeString(a.Pubkey)
	if err != nil {
		return nil
	}
	obj, err := secp256k1.ParsePubKey(k)
	if err != nil {
		return nil
	}
	return obj
}

// Public implements the crypto.Signer interface
// Returns the account's public key as a crypto.PublicKey interface
func (a *Account) Public() crypto.PublicKey {
	return a.PublicKey()
}

// DerivePublic derives a child public key from this account's public key
// Uses HD wallet derivation with the provided subpath
// Returns the derived public key and any error encountered
func (a *Account) DerivePublic(subpath string) (*secp256k1.PublicKey, error) {
	if a.Chaincode == "" {
		return nil, errors.New("need chaincode")
	}
	chainCode, err := base64.RawURLEncoding.DecodeString(a.Chaincode)
	if err != nil {
		return nil, err
	}
	_, newpub, err := DerivePublicKey(a.PublicKey(), chainCode, subpath)
	return newpub, err
}

// GetAddress returns the account's blockchain address
func (a *Account) GetAddress() string {
	return a.Address
}

// DerivePublicKey takes a public key and a path, and returns a derived public key
// Implements BIP32 HD wallet derivation for public keys (no hardened derivation)
// Parameters:
//   - pubkey: The parent public key
//   - chainCode: The chain code for derivation
//   - path: Derivation path in format "m/a/b/c" (only normal, non-hardened derivation supported)
//
// Returns:
//   - The intermediate value (IL) used in derivation
//   - The derived public key
//   - Any error encountered during derivation
func DerivePublicKey(pubkey *secp256k1.PublicKey, chainCode []byte, path string) (*big.Int, *secp256k1.PublicKey, error) {
	// Parse the derivation path
	pathA := strings.Split(path, "/")
	if pathA[0] != "m" {
		return nil, nil, errors.New("path must start with m/")
	}
	pathA = pathA[1:]
	if len(pathA) < 1 {
		return nil, nil, errors.New("path cannot be empty, must have at least a derivation")
	}

	// Convert path segments to integers
	pathInt := make([]uint32, len(pathA))
	for n, v := range pathA {
		x, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, nil, err
		}
		if x >= 0x80000000 {
			return nil, nil, errors.New("hardened keys not supported in here")
		}
		pathInt[n] = uint32(x)
	}

	// Create extended key from public key and chain code
	ek, err := ecckd.FromPublicKey(pubkey.ToECDSA(), chainCode)
	if err != nil {
		return nil, nil, err
	}

	// Derive child key and get intermediate value (IL)
	il, ek, err := ek.DeriveWithIL(pathInt)
	if err != nil {
		return nil, nil, err
	}

	// Convert to secp256k1.PublicKey format
	pub, err := secp256k1.ParsePubKey(ek.KeyData)
	if err != nil {
		return nil, nil, err
	}

	return il, pub, nil
}

// ApiUpdate handles API requests to update account properties
// Currently supports updating the account name
// Returns nil if no updates were made or any error encountered during saving
func (a *Account) ApiUpdate(ctx *apirouter.Context) error {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return errors.New("failed to get env")
	}

	updated := false

	if v, ok := apirouter.GetParam[string](ctx, "Name"); ok {
		a.Name = v
		updated = true
	}
	if !updated {
		return nil
	}
	return a.save(e)
}

// ApiDelete handles API requests to delete an account
// Returns any error encountered during the deletion
func (a *Account) ApiDelete(ctx *apirouter.Context) error {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return errors.New("failed to get env")
	}

	return a.accountDelete(e)
}

// accountDelete removes an account and related data from the database
// Emits an "account:delete" event and deletes the account record
// TODO: Also delete transactions and connected sites once implemented
// Returns any error encountered during deletion
func (a *Account) accountDelete(e wltintf.Env) error {
	// delete any transaction from this account
	e.Emitter().Emit(context.Background(), "account:delete", a.Id.String())
	//TODO e.sql.Where(map[string]any{"From": a.Id.String()}).Delete(&Transaction{})
	//TODO e.sql.Where(map[string]any{"Account": a.Id.String()}).Delete(&connectedSite{})

	return e.Delete(a)
}

// Sign signs a digest using the account's parent wallet
// Implements the crypto.Signer interface
// Parameters:
//   - rand: random source (passed to wallet's sign method)
//   - digest: the hash or message to sign
//   - opts: must be *wltsign.Opts containing context and key information
//
// Returns the signature and any error encountered
func (a *Account) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	aopt, ok := opts.(*wltsign.Opts)
	if !ok {
		return nil, errors.New("sign requires appropriate options")
	}
	// Add the account's IL (intermediate value) to the options
	aopt.IL = a.IL

	// Get the parent wallet
	w, err := pobj.ById[wltwallet.Wallet](aopt.Context, a.Wallet.String())
	if err != nil {
		return nil, err
	}

	// Delegate signing to the wallet
	return w.Sign(rand, digest, aopt)
}

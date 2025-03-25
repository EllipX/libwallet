package wltwallet

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/EllipX/libwallet/wltintf"
	"github.com/EllipX/libwallet/wltsign"
	"github.com/KarpelesLab/cryptutil"
	"github.com/KarpelesLab/rest"
	"github.com/KarpelesLab/spotlib"
	"github.com/KarpelesLab/xuid"
	"github.com/ModChain/tss-lib/v2/ecdsa/keygen"
	"github.com/fxamacker/cbor/v2"
)

type WalletKey struct {
	Id     *xuid.XUID `gorm:"primaryKey"`
	Wallet *xuid.XUID
	Type   string
	Key    string `json:"Key,omitempty"` // (public) key used for encryption
	Data   []byte `json:",protect"`
	pre    *keygen.LocalPreParams
	sdata  *keygen.LocalPartySaveData
}

func (wk *WalletKey) save(e wltintf.Env) error {
	return e.Save(wk)
}

func (w *Wallet) createWalletKey(ctx context.Context, typ string) (*WalletKey, error) {
	// generate key
	preParams, err := keygen.GeneratePreParamsWithContext(ctx)
	if err != nil {
		return nil, err
	}
	final := &WalletKey{
		Id:     xuid.New("wkey"),
		Wallet: w.Id,
		Type:   typ,
		pre:    preParams,
	}
	return final, nil
}

// encrypt stores wk.sdata into wk.Data
func (wk *WalletKey) encrypt(kd *wltsign.KeyDescription) error {
	res, err := cryptutil.MarshalJson(wk.sdata)
	if err != nil {
		return err
	}

	wk.Type = kd.Type

	switch kd.Type {
	case "StoreKey":
		// encrypt
		pubKey, err := storeKeyReadPublic(kd.Key)
		if err != nil {
			return err
		}
		pubKeyB, err := x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return err
		}
		wk.Key = base64.RawURLEncoding.EncodeToString(pubKeyB)
		// encrypt for our key
		err = res.Encrypt(rand.Reader, pubKey)
		if err != nil {
			return err
		}
	case "RemoteKey":
		// store on remote server
		// First, get keys of machines that will need to be able to decrypt this
		var ids []string
		err = rest.Apply(context.Background(), "EllipX/WalletSign:keys", "GET", nil, &ids)
		if err != nil {
			err = rest.Apply(context.Background(), "EllipX/WalletSign:keys", "GET", nil, &ids)
			if err != nil {
				return err
			}
		}
		var keys []crypto.PublicKey
		for _, idStr := range ids {
			idC := &cryptutil.IDCard{}
			idBin, err := base64.RawURLEncoding.DecodeString(idStr)
			if err != nil {
				return err
			}
			err = idC.UnmarshalBinary(idBin)
			if err != nil {
				return err
			}
			keys = append(keys, idC.GetKeys("decrypt")...)
		}
		// encrypt bottle
		err = res.Encrypt(rand.Reader, keys...)
		if err != nil {
			return err
		}
	case "Plain":
		// do nothing
	case "Password":
		pk, err := passwordToEd25519(kd.Key, wk.Id.UUID[:])
		if err != nil {
			return err
		}
		pubKey := pk.Public()
		pubKeyB, err := x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return err
		}
		wk.Key = base64.RawURLEncoding.EncodeToString(pubKeyB)
		// encrypt for our key
		err = res.Encrypt(rand.Reader, pubKey)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported key type %s", kd.Type)
	}

	buf, err := cbor.Marshal(res)
	if err != nil {
		return err
	}
	if kd.Type == "RemoteKey" {
		// upload bottle
		_, err = rest.Do(context.Background(), "EllipX/WalletSign:setGeneratedKey", "POST", rest.Param{"data": base64.RawURLEncoding.EncodeToString(buf), "key": kd.Key})
		if err != nil {
			return err
		}
		wk.Key = kd.Key
	}
	wk.Data = buf
	return nil
}

func (wk *WalletKey) decrypt(kd *wltsign.KeyDescription, purpose keyUsagePurpose) (*keygen.LocalPartySaveData, error) {
	bottle := cryptutil.AsCborBottle(wk.Data)

	op := cryptutil.EmptyOpener

	switch wk.Type {
	case "StoreKey":
		k, err := storeKeyToEd25519(kd.Key)
		if err != nil {
			return nil, err
		}
		pkBin, err := x509.MarshalPKIXPublicKey(k.Public())
		if err != nil {
			return nil, err
		}
		curPkBin, err := base64.RawURLEncoding.DecodeString(wk.Key)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(pkBin, curPkBin) {
			return nil, ErrBadStoreKey
		}
		op, err = cryptutil.NewOpener(k)
		if err != nil {
			return nil, err
		}
	case "Password":
		pk, err := passwordToEd25519(kd.Key, wk.Id.UUID[:])
		if err != nil {
			return nil, err
		}
		pkBin, err := x509.MarshalPKIXPublicKey(pk.Public())
		if err != nil {
			return nil, err
		}
		curPkBin, err := base64.RawURLEncoding.DecodeString(wk.Key)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(pkBin, curPkBin) {
			return nil, ErrBadPassword
		}
		op, err = cryptutil.NewOpener(pk)
		if err != nil {
			return nil, err
		}
	case "Plain":
		// do nothing
	default:
		return nil, fmt.Errorf("cannot open keys of type %s", wk.Type)
	}

	var final *keygen.LocalPartySaveData
	_, err := op.Unmarshal(bottle, &final)
	if err != nil {
		return nil, fmt.Errorf("while decrypting key %s: %w", wk.Id, err)
	}
	return final, err
}

func selectPeer(ctx context.Context, spot *spotlib.Client) (string, error) {
	var ids []string
	err := rest.Apply(ctx, "EllipX/WalletSign:keys", "GET", nil, &ids)
	if err != nil {
		err = rest.Apply(ctx, "EllipX/WalletSign:keys", "GET", nil, &ids)
		if err != nil {
			return "", err
		}
	}
	var keys []string
	for _, idStr := range ids {
		idC := &cryptutil.IDCard{}
		idBin, err := base64.RawURLEncoding.DecodeString(idStr)
		if err != nil {
			return "", err
		}
		err = idC.UnmarshalBinary(idBin)
		if err != nil {
			log.Printf("failed to parse peer ID: %s", err)
			continue
		}

		key := "k." + base64.RawURLEncoding.EncodeToString(cryptutil.Hash(idC.Self, sha256.New))
		keys = append(keys, key)
	}

	// let's try to ping
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	res := make(chan string, 1)

	for _, k := range keys {
		go func(k string) {
			pingBuf := make([]byte, 32)
			io.ReadFull(rand.Reader, pingBuf)
			x, err := spot.Query(ctx, k+"/ping", pingBuf)
			if err != nil {
				log.Printf("failed to read from %s: %s", k, err)
				return
			}
			if !bytes.Equal(pingBuf, x) {
				log.Printf("bad buffer from %s", k)
				return
			}
			select {
			case res <- k:
			default:
			}
		}(k)
	}

	select {
	case v := <-res:
		return v, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

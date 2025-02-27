package wltnet

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ModChain/ethrpc"
	"github.com/ModChain/outscript"
	"math/big"
	"strings"
	"time"

	"git.atonline.com/ellipx/libwallet/wltintf"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/pobj"
	"github.com/KarpelesLab/xuid"
)

type Token struct {
	Id          *xuid.XUID `gorm:"primaryKey"`
	Network     *xuid.XUID
	Type        string    `json:"type"` // ethereum | bitcoin
	Address     string    `json:"address"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	Decimals    string    `json:"decimals"`
	TotalSupply string    `json:"totalSupply"`
	Created     time.Time `gorm:"autoCreateTime"`
	Updated     time.Time `gorm:"autoUpdateTime"`
}

func init() {
	pobj.RegisterActions[Token]("Token",
		&pobj.ObjectActions{
			List:   pobj.Static(apiListToken),
			Create: pobj.Static(apiCreateToken),
			Fetch:  pobj.Static(apiFetchToken),
		},
	)
	pobj.RegisterStatic("Token:discoverToken", apiDiscoverToken)
}

func (t *Token) save(e wltintf.Env) error {
	return e.Save(t)
}

func (t *Token) validate() error {
	if t.Network == nil {
		return errors.New("missing network")
	}
	if t.Name == "" {
		return errors.New("missing name")
	}
	if t.Symbol == "" {
		return errors.New("missing symbol")
	}
	if t.Decimals == "" {
		return errors.New("missing decimals")
	}
	if t.TotalSupply == "" {
		return errors.New("missing total supply")
	}
	if t.Address == "" {
		return errors.New("missing address")
	}
	switch t.Type {
	case "ethereum":
		// check if address is valid
		addr, err := outscript.ParseEvmAddress(t.Address)
		if err != nil {
			return fmt.Errorf("failed to parse address: %s", err)
		}
		t.Address, err = addr.Address() // re-output address to guarantee proper formatting
		return nil
	case "bitcoin":
		addr, err := outscript.ParseBitcoinAddress(t.Address)
		if err != nil {
			return fmt.Errorf("failed to parse address: %s", err)
		}
		t.Address, err = addr.Address() // will use the initial parsing address, so we should get back the same stuff
		return nil
	default:
		return fmt.Errorf("unsupported Token type %s", t.Type)
	}
}

func apiDiscoverToken(ctx context.Context) (any, error) {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	n := apirouter.GetObject[Network](ctx, "Network")
	if n == nil {
		// if no network passed, take current net
		var err error
		n, err = CurrentNetwork(e)
		if err != nil {
			return nil, err
		}
	}

	address, okaddress := apirouter.GetParam[string](ctx, "address")
	if !okaddress {
		return nil, errors.New("missing field address")
	}

	return DiscoverToken(n, address)
}

func apiListToken(ctx *apirouter.Context) (any, error) {
	return wltintf.ListHelper[Token](ctx, "Name ASC", "Name", "Address", "Network")
}

func apiCreateToken(ctx *apirouter.Context, t *Token) (any, error) {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	if t.Network == nil {
		n, err := CurrentNetwork(e)
		if err != nil {
			return nil, err
		}
		t.Network = n.Id
	}
	err := t.validate()
	if err != nil {
		return nil, err
	}

	t.Id, err = xuid.NewRandom("tk")
	if err != nil {
		return nil, err
	}

	err = e.Save(t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func DiscoverToken(n *Network, address string) (any, error) {
	// ERC20 function selectors.
	selectors := map[string]string{
		"name":        "0x06fdde03",
		"symbol":      "0x95d89b41",
		"decimals":    "0x313ce567",
		"totalSupply": "0x18160ddd",
	}

	param := map[string]string{
		"data": selectors["name"],
	}

	call := func(key string) (string, error) {
		param = map[string]string{
			"data": selectors[key],
			"to":   address,
		}

		if key == "name" || key == "symbol" {
			str, err := doEthCallAndDecodeString(n, param)
			if err != nil {
				return "", err
			}
			return str, nil
		} else {
			str, err := doEthCallUint256AndDecodeString(n, param)
			if err != nil {
				return "", err
			}
			return str, nil
		}
	}

	// Retrieve token name.
	name, err := call("name")
	if err != nil {
		return nil, errors.New("Cannot retrieve token name: " + err.Error())
	}

	// Retrieve token symbol.
	symbol, err := call("symbol")
	if err != nil {
		return nil, errors.New("Cannot retrieve token symbol: " + err.Error())
	}

	// Retrieve token decimals.
	decimals, err := call("decimals")
	if err != nil {
		return nil, errors.New("Cannot retrieve token decimals: " + err.Error())
	}

	// Retrieve token totalSupply.
	totalSupply, err := call("totalSupply")
	if err != nil {
		return nil, errors.New("Cannot retrieve token totalSupply: " + err.Error())
	}

	token := map[string]any{
		"address":     address,
		"name":        name,
		"symbol":      symbol,
		"decimals":    decimals,
		"totalSupply": totalSupply,
	}

	return token, nil
}

func doEthCallAndDecodeString(n *Network, param map[string]string) (string, error) {
	hexStr, err := ethrpc.ReadString(n.DoRPC("eth_call", param, "latest"))
	if err != nil {
		fmt.Printf("error RPC %v", err)
		return "", err
	}
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}

	if len(hexStr) == 0 {
		return "", nil
	}

	// Decode hex string to readable text
	buf, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %w", err)
	}
	ln := new(big.Int).SetBytes(buf[32:64]).Int64()
	if len(buf) < 64+int(ln) {
		return "", fmt.Errorf("buffer too small")
	}
	return string(buf[64 : 64+ln]), nil
}

func doEthCallUint256AndDecodeString(n *Network, param map[string]string) (string, error) {
	num, err := ethrpc.ReadBigInt(n.DoRPC("eth_call", param, "latest"))
	if err != nil {
		fmt.Printf("error RPC %v", err)
		return "", err
	}

	return num.String(), nil
}

func apiFetchToken(ctx *apirouter.Context, in struct{ Id string }) (any, error) {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	id, err := xuid.Parse(in.Id)
	if err != nil {
		return nil, err
	}

	return TokenById(e, id)
}

func TokenById(e wltintf.Env, id *xuid.XUID) (*Token, error) {
	if id.Prefix != "tk" {
		return nil, fmt.Errorf("invalid key for token: %s", id.Prefix)
	}
	return wltintf.ByPrimaryKey[Token](e, id)
}

func (t *Token) ApiDelete(ctx *apirouter.Context) error {
	e := wltintf.GetEnv(ctx)
	if e == nil {
		return errors.New("failed to get env")
	}

	return e.Delete(t)
}

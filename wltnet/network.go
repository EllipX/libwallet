package wltnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EllipX/ellipxobj"
	"github.com/EllipX/libwallet/chains"
	"github.com/EllipX/libwallet/wltasset"
	"github.com/EllipX/libwallet/wltintf"
	"github.com/EllipX/libwallet/wltnft"
	"github.com/EllipX/libwallet/wltutil"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/xuid"
	"github.com/ModChain/ethrpc"
)

var (
	networkCache   map[xuid.XUID]*Network
	networkCacheLk sync.Mutex
)

const ModChainApiKey = "crapi-nx4p6j-ifez-cjli-p5wj-uml43cte"

type Network struct {
	Id               *xuid.XUID     `gorm:"primaryKey"`
	Type             string         `gorm:"index:typeChain,unique"` // evm | bitcoin
	ChainId          string         `gorm:"index:typeChain,unique"` // for Type=evm, the chain id from chainlist. For Type=bitcoin, chain key is included here
	Name             string         // name, automatic if empty
	RPC              string         // rpc url, automatic if empty
	validRPC         ethrpc.Handler // valid RPC servers
	CurrencySymbol   string         // currency symbol, automatic if empty
	CurrencyDecimals int            // decimals, automatic if zero
	BlockExplorer    string         // explorer, automatic if empty
	TestNet          bool           // is this a testnet?
	Priority         int            // display priority
	Created          time.Time      `gorm:"autoCreateTime"`
	Updated          time.Time      `gorm:"autoUpdateTime"`
}

type NativeCurrencyObject struct {
	Decimals int    `json:"decimals"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
}

type AddEthereumChainParameter struct {
	ChainId           string               `json:"chainId"` // The chain ID as a 0x-prefixed hexadecimal string
	BlockExplorerUrls []string             `json:"blockExplorerUrls"`
	ChainName         string               `json:"chainName"`
	IconUrls          []string             `json:"iconUrls"`
	NativeCurrency    NativeCurrencyObject `json:"nativeCurrency"`
	RPCUrls           []string             `json:"rpcUrls"`
}

func (a *AddEthereumChainParameter) Validate() error {
	if !strings.HasPrefix(a.ChainId, "0x") {
		return &apirouter.Error{Code: -32602, Message: "Expected 0x-prefixed, unpadded, non-zero hexadecimal string 'chainId'."}
	}
	bigV, ok := new(big.Int).SetString(a.ChainId, 0)
	if !ok || "0x"+bigV.Text(16) != a.ChainId {
		return &apirouter.Error{Code: -32602, Message: "Expected 0x-prefixed, unpadded, non-zero hexadecimal string 'chainId'."}
	}
	if len(a.ChainName) < 3 {
		return &apirouter.Error{Code: -32602, Message: "Expected chainName"}
	}
	if n := len(a.NativeCurrency.Symbol); n < 2 || n > 6 {
		return &apirouter.Error{Code: -32602, Message: "Expected 2-6 character string 'nativeCurrency.symbol'."}
	}
	// TODO add more checks?

	return nil
}

func (a *AddEthereumChainParameter) AsNetwork() *Network {
	bigV, _ := new(big.Int).SetString(a.ChainId, 0)
	if len(a.RPCUrls) == 0 {
		a.RPCUrls = []string{"auto"}
	}

	net := &Network{
		Id:               NetworkIdForTypeAndChainId("evm", bigV.Text(10)),
		Type:             "evm",
		ChainId:          bigV.Text(10),
		Name:             a.ChainName,
		RPC:              "auto",
		CurrencySymbol:   a.NativeCurrency.Symbol,
		CurrencyDecimals: a.NativeCurrency.Decimals,
		BlockExplorer:    "auto",
	}
	if len(a.RPCUrls) > 0 {
		net.RPC = a.RPCUrls[0]
	}
	if len(a.BlockExplorerUrls) > 0 {
		net.BlockExplorer = a.BlockExplorerUrls[0]
	}
	return net
}

func (n *Network) check() error {
	// check network status and fill anything missing
	switch n.Type {
	case "evm":
		// ok
	case "bitcoin":
		switch n.ChainId {
		case "bitcoin":
			if n.Name == "" {
				n.Name = "Bitcoin"
			}
		case "bitcoin-cash":
			if n.Name == "" {
				n.Name = "Bitcoin Cash"
			}
		case "litecoin":
			if n.Name == "" {
				n.Name = "Litecoin"
			}
		case "dogecoin":
			if n.Name == "" {
				n.Name = "Dogecoin"
			}
		default:
			return fmt.Errorf("invalid network type %s/%s", n.Type, n.ChainId)
		}
	default:
		return fmt.Errorf("invalid network type %s", n.Type)
	}
	info, err := n.GetChainInfo()
	if err != nil {
		// ignore fetch failed but do not input anything either
		return nil
	}
	if info.ChainId == 137 && n.CurrencySymbol == "MATIC" {
		n.CurrencySymbol = "POL"
	}
	if n.Name == "" {
		n.Name = info.Name
	}
	if n.RPC == "" {
		n.RPC = "auto"
	}
	if n.CurrencySymbol == "" {
		n.CurrencySymbol = info.NativeCurrency.Symbol
	}
	if n.BlockExplorer == "" {
		n.BlockExplorer = "auto"
	}
	return nil
}

func (n *Network) Save(e wltintf.Env) error {
	if n.Id == nil {
		// compute id
		n.Id = xuid.Must(xuid.FromKeyPrefix(n.Type+"."+n.ChainId, "net"))
	}
	return e.Save(n)
}

func NetworkIdForTypeAndChainId(typ, chainId string) *xuid.XUID {
	return xuid.Must(xuid.FromKeyPrefix(typ+"."+chainId, "net"))
}

func (n *Network) addToCache() {
	networkCacheLk.Lock()
	defer networkCacheLk.Unlock()

	networkCache[*n.Id] = n
}

func networkFromCache(id *xuid.XUID) *Network {
	networkCacheLk.Lock()
	defer networkCacheLk.Unlock()
	k, ok := networkCache[*id]
	if ok {
		return k
	}
	return nil
}

func (n *Network) SetCurrent(e wltintf.Env) error {
	// broadcast change
	go wltutil.BroadcastMsg("js:chainChanged", map[string]any{"chainId": n.ChainId})
	return e.SetCurrent("network", n.Id.String())
}

func (n *Network) MarshalJSON() ([]byte, error) {
	res := map[string]any{
		"Id":             n.Id,
		"Type":           n.Type,
		"ChainId":        n.ChainId,
		"Name":           n.Name,
		"RPC":            n.RPC,
		"CurrencySymbol": n.CurrencySymbol,
		"BlockExplorer":  n.BlockExplorer,
		"TestNet":        n.TestNet,
		"Created":        n.Created,
		"Updated":        n.Updated,
	}

	switch n.Type {
	case "evm":
		res["EVM_Info"], _ = n.GetChainInfo()
		return json.Marshal(res)
	default:
		return json.Marshal(res)
	}
}

func (n *Network) String() string {
	return n.Type + "." + n.ChainId
}

func (n *Network) GetChainInfo() (*chains.ChainInfo, error) {
	if n.Type != "evm" {
		return nil, errors.New("unsupported for this network Type")
	}
	idN, err := strconv.ParseUint(n.ChainId, 0, 64)
	if err != nil {
		return nil, err
	}
	info := chains.Get(idN)
	if info == nil {
		return nil, fmt.Errorf("unknown net id %d", idN)
	}
	return info, nil
}

func (n *Network) NativeSymbol() (string, error) {
	switch n.Type {
	case "evm":
		info, err := n.GetChainInfo()
		if err != nil {
			return "", err
		}
		return info.NativeCurrency.Symbol, nil
	case "bitcoin":
		switch n.ChainId {
		case "bitcoin":
			return "BTC", nil
		case "bitcoin-cash":
			return "BCH", nil
		case "litecoin":
			return "LTC", nil
		case "dogecoin":
			return "DOGE", nil
		default:
			return "", fmt.Errorf("unsupported bitcoin chain type %s", n.ChainId)
		}
	default:
		return "", errors.New("symbol not available (not supported type)")
	}
}

func (n *Network) TransactionUrl(txHash string) string {
	if e := n.BlockExplorer; e != "" && e != "auto" {
		return fmt.Sprintf("%s/tx/%s", e, txHash)
	}
	info, err := n.GetChainInfo()
	if err != nil {
		return ""
	}
	return info.TransactionUrl(txHash)
}

func (n *Network) getRPC() (ethrpc.Handler, error) {
	if n.Type == "bitcoin" {
		if n.validRPC != nil {
			return n.validRPC, nil
		}
		n.validRPC = ethrpc.New("https://rpc.modchain.net/api/" + ModChainApiKey + "/" + n.ChainId + "/rpc")
		return n.validRPC, nil
	}
	if n.RPC != "" && n.RPC != "auto" {
		return ethrpc.New(n.RPC), nil
	}
	if n.validRPC != nil {
		return n.validRPC, nil
	}

	// get RPC servers and select the best one
	info, err := n.GetChainInfo()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var rpcList []string

	switch info.ChainId {
	case 1:
		n.validRPC = ethrpc.New("https://rpc.modchain.net/api/" + ModChainApiKey + "/ethereum/rpc")
		return n.validRPC, nil
	case 137:
		n.validRPC = ethrpc.New("https://rpc.modchain.net/api/" + ModChainApiKey + "/polygon/rpc")
		return n.validRPC, nil
	}

	for _, r := range info.RPC {
		r = strings.ReplaceAll(r, "${INFURA_API_KEY}", infuraKey)
		if strings.Contains(r, "${") {
			continue
		}
		rpcList = append(rpcList, r)
	}

	list, err := ethrpc.Evaluate(ctx, rpcList...)
	if err != nil {
		return nil, err
	}

	n.validRPC = list
	return list, nil
}

func (n *Network) DoRPC(method string, args ...any) (json.RawMessage, error) {
	e, err := n.getRPC()
	if err != nil {
		return nil, err
	}

	return e.DoCtx(context.Background(), method, args...)
}

type AddressProvider interface {
	GetAddress() string
}

func (n *Network) nativeBalance(acct AddressProvider) (*ellipxobj.Amount, error) {
	switch n.Type {
	case "evm":
		// fetch from RPC
		// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_getbalance
		amt, err := ethrpc.ReadString(n.DoRPC("eth_getBalance", acct.GetAddress(), "latest"))
		if err != nil {
			return nil, err
		}
		info, err := n.GetChainInfo()
		if err != nil {
			return nil, err
		}
		// amt is amount in hex
		// info.NativeCurrency.Decimals
		i, ok := new(big.Int).SetString(amt, 0)
		if !ok {
			return nil, errors.New("invalid amount obtained from rpc")
		}
		decimals := n.CurrencyDecimals
		if decimals == 0 {
			decimals = info.NativeCurrency.Decimals
		}
		return ellipxobj.NewAmountRaw(i, decimals), nil
	default:
		return nil, fmt.Errorf("unsupporte type %s", n.Type)
	}
}

func (n *Network) NativeAsset(e wltintf.Env, acct AddressProvider) (*wltasset.Asset, error) {
	switch n.Type {
	case "evm":
		amt, err := n.nativeBalance(acct)
		if err != nil {
			return nil, err
		}
		info, err := n.GetChainInfo()
		if err != nil {
			return nil, err
		}

		asset := &wltasset.Asset{
			Key:     n.String() + ".NATIVE",
			Name:    info.NativeCurrency.Name,
			Symbol:  info.NativeCurrency.Symbol,
			Amount:  amt,
			Network: n.Id,
			Type:    "fungible",
			TestNet: n.TestNet,
		}
		asset.Info, err = wltasset.CoinInfoBySymbol(e, info.NativeCurrency.Symbol)
		if err != nil {
			log.Printf("error fetching coin infos: %s", err)
		}

		return asset, nil
	case "bitcoin":
		amt, err := n.nativeBalance(acct)
		if err != nil {
			return nil, err
		}

		sym, _ := n.NativeSymbol()

		asset := &wltasset.Asset{
			Key:     n.String() + ".NATIVE",
			Name:    n.Name,
			Symbol:  sym,
			Amount:  amt,
			Network: n.Id,
			Type:    "fungible",
			TestNet: n.TestNet,
		}
		asset.Info, _ = wltasset.CoinInfoBySymbol(e, sym)

		return asset, nil
	default:
		return nil, fmt.Errorf("unsupporte type %s", n.Type)
	}
}

func (n *Network) Nfts(e wltintf.Env, acct AddressProvider) (*[]wltnft.Nft, error) {
	switch n.Type {
	case "evm":
		if n.ChainId != "1" {
			return nil, errors.New("unsupported for this ethereum network with chain id " + n.ChainId)
		}
		nfts, err := n.NftList(e, acct)
		if err != nil {
			return nil, err
		}
		return nfts, nil
	case "bitcoin":
		return nil, fmt.Errorf("unsupporte type %s", n.Type)
	default:
		return nil, fmt.Errorf("unsupporte type %s", n.Type)
	}
}

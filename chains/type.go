package chains

import "fmt"

type ChainFeature struct {
	Name string `json:"name"`
}

type ChainCurrency struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

type ChainENS struct {
	Registry string `json:"registry"`
}

type ChainExplorer struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Standard string `json:"standard"` // EIP3091
}

type ChainInfo struct {
	Name           string           `json:"name"`
	Chain          string           `json:"chain"`
	Icon           string           `json:"icon"`
	RPC            []string         `json:"rpc"`
	Features       []*ChainFeature  `json:"features"`
	Faucets        []string         `json:"faucets"`
	NativeCurrency *ChainCurrency   `json:"nativeCurrency"`
	InfoURL        string           `json:"infoURL"`
	ShortName      string           `json:"shortName"`
	ChainId        uint64           `json:"chainId"`
	NetworkId      uint64           `json:"networkId"`
	Slip44         int              `json:"slip44,omitempty"`
	ENS            *ChainENS        `json:"ens"`
	Explorers      []*ChainExplorer `json:"explorers"`
}

func (ci *ChainInfo) HasFeature(feat string) bool {
	for _, s := range ci.Features {
		if s.Name == feat {
			return true
		}
	}
	return false
}

func (ci *ChainInfo) TransactionUrl(txHash string) string {
	if len(ci.Explorers) == 0 {
		return ""
	}
	return fmt.Sprintf("%s/tx/%s", ci.Explorers[0].URL, txHash)
}

func (ci *ChainInfo) ExplorerURL() string {
	if len(ci.Explorers) > 0 {
		return ci.Explorers[0].URL
	}
	return ""
}

package wltnft

import (
	"time"

	"github.com/KarpelesLab/xuid"
)

type NftAttribute struct {
	TraitType   string `json:"trait_type,omitempty"`
	DisplayType string `json:"display_type,omitempty"`
	Value       any    `json:"value"`
}

type Nft struct {
	Id              *xuid.XUID      `json:"id,omitempty" gorm:"primaryKey"`
	Key             string          `json:"key" gorm:"index:Key,unique"`
	ContractAddress string          `json:"contract_address"`
	ContractName    string          `json:"contract_name"`
	TokenId         string          `json:"token_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Image           string          `json:"image,omitempty"`
	ImageUrl        string          `json:"image_url,omitempty"`
	AnimationUrl    string          `json:"animation_url,omitempty"`
	BackgroundColor string          `json:"background_color,omitempty"`
	YoutubeUrl      string          `json:"youtube_url,omitempty"`
	ExternalUrl     string          `json:"external_url,omitempty"`
	Decimals        string          `json:"decimals,omitempty"`
	Attributes      []*NftAttribute `json:"attributes" gorm:"serializer:json"`
	Network         *xuid.XUID      `json:"network,omitempty"`
	Created         time.Time       `gorm:"autoCreateTime"`
	Updated         time.Time       `gorm:"autoUpdateTime"`
}

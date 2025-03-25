package wltnet

import "github.com/EllipX/libwallet/wltintf"

func InitEnv(e wltintf.Env) {
	e.AutoMigrate(&Network{})
	MakeDefaultNetworks(e)
}

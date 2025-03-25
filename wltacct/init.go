package wltacct

import "github.com/EllipX/libwallet/wltintf"

func InitEnv(e wltintf.Env) {
	e.AutoMigrate(&Account{})
}

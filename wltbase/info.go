package wltbase

import (
	"context"
	"errors"
	"os"

	"github.com/EllipX/ellipxobj"
	"github.com/EllipX/libwallet/wltacct"
	"github.com/EllipX/libwallet/wltwallet"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/pobj"
)

var (
	dateTag string
	gitTag  string
)

func init() {
	pobj.RegisterStatic("Info:ping", infoPing)
	pobj.RegisterStatic("Info:version", infoVersion)
	pobj.RegisterStatic("Info:paths", infoPaths)
	pobj.RegisterStatic("Info:first_run", infoFirstRun)
	pobj.RegisterStatic("Info:onboarding", infoOnboarding)
}

func infoPing() (any, error) {
	return "pong", nil
}

func infoVersion() (any, error) {
	return map[string]any{
		"dateTag": dateTag,
		"gitTag":  gitTag,
	}, nil
}

func infoPaths(ctx context.Context) (any, error) {
	res := make(map[string]any)

	e := apirouter.GetObject[env](ctx, "@env")
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	if v, err := os.UserCacheDir(); err == nil {
		res["UserCacheDir"] = v
	}
	if v, err := os.UserConfigDir(); err == nil {
		res["UserConfigDir"] = v
	}
	if v, err := os.UserHomeDir(); err == nil {
		res["UserHomeDir"] = v
	}
	res["TempDir"] = os.TempDir()
	res["DataDir"] = e.dataDir
	res["Environ"] = os.Environ()

	return res, nil
}

func infoFirstRun(ctx context.Context) (any, error) {
	e := apirouter.GetObject[env](ctx, "@env")
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	v, err := e.DBSimpleGet([]byte("info"), []byte("first_run"))
	if err != nil {
		return nil, err
	}
	t := &ellipxobj.TimeId{}
	err = t.UnmarshalBinary(v)
	return t, err
}

func infoOnboarding(ctx context.Context) (any, error) {
	e := apirouter.GetObject[env](ctx, "@env")
	if e == nil {
		return nil, errors.New("failed to get env")
	}

	acct := wltacct.HasAccount(e)
	wallet := wltwallet.HasWallet(e)

	res := map[string]any{
		"has_account": acct,
		"has_wallet":  wallet,
	}
	return res, nil
}

package wltwallet

import (
	"net/http"

	"github.com/KarpelesLab/apirouter"
)

var (
	ErrBadPassword = &apirouter.Error{Message: "wrong password", Token: "error_wrong_password", Code: http.StatusForbidden}
	ErrBadStoreKey = &apirouter.Error{Message: "wrong storeKey, try to restore your wallet from the cloud", Token: "error_wrong_store_key", Code: http.StatusForbidden}
)

package wltsign

import (
	"context"
	"crypto"
	"math/big"
)

type Opts struct {
	Context context.Context
	IL      *big.Int
	Keys    []*KeyDescription
}

func (a *Opts) HashFunc() crypto.Hash {
	return crypto.Hash(0)
}

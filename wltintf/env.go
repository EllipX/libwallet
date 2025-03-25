package wltintf

import (
	"context"
	"errors"
	"time"

	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/emitter"
	"github.com/KarpelesLab/spotlib"
)

type Env interface {
	Save(obj any) error
	Delete(obj any) error
	DeleteWhere(obj any, where map[string]any) error
	DeleteAll(obj any) error
	SetCurrent(k, v string) error
	GetCurrent(k string) (string, error)
	Emitter() *emitter.Hub
	Spot() *spotlib.Client
	CacheGet(ctx context.Context, u string, timeout, refresh time.Duration) ([]byte, error)
	AutoMigrate(obj any)

	// db stuff
	DBSimpleGet(bucket, key []byte) (r []byte, err error)
	DBSimpleDel(bucket []byte, keys ...[]byte) error
	DBSimpleSet(bucket, key, val []byte) error
	First(res any) error
	FirstId(res, id any) error
	FirstWhere(res any, where map[string]any) error
	Find(res any, where map[string]any) error
	ListHelper(ctx context.Context, target any, sort string, searchKey ...string) error
	Count(obj any) int64
}

func GetEnv(ctx context.Context) Env {
	var c *apirouter.Context
	ctx.Value(&c)
	if c == nil {
		return nil
	}
	v, ok := c.GetObject("@env").(Env)
	if ok {
		return v
	}
	return nil
}

func ByPrimaryKey[T any](e Env, id any) (*T, error) {
	var res *T
	err := e.FirstId(&res, id)
	return res, err
}

func ListHelper[T any](ctx context.Context, sort string, searchKey ...string) (any, error) {
	var res []*T
	e := GetEnv(ctx)
	if e == nil {
		return nil, errors.New("failed to get env")
	}
	err := e.ListHelper(ctx, &res, sort, searchKey...)
	return res, err
}

package wltbase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/EllipX/ellipxobj"
	"github.com/EllipX/libwallet/wltacct"
	"github.com/EllipX/libwallet/wltasset"
	"github.com/EllipX/libwallet/wltcontact"
	"github.com/EllipX/libwallet/wltcrash"
	"github.com/EllipX/libwallet/wltnet"
	"github.com/EllipX/libwallet/wltnft"
	"github.com/EllipX/libwallet/wlttx"
	"github.com/EllipX/libwallet/wltwallet"
	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/emitter"
	"github.com/KarpelesLab/rest"
	"github.com/KarpelesLab/spotlib"
	_ "github.com/glebarez/go-sqlite"
	bolt "go.etcd.io/bbolt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type env struct {
	context.Context
	dataDir string
	db      *bolt.DB
	sql     *gorm.DB
	spot    *spotlib.Client
	em      *emitter.Hub
}

type client struct {
	c   net.Conn
	enc *json.Encoder
	wlk sync.Mutex // write lock
}

func InitEnv(dataDir string) (any, error) {
	e := &env{Context: context.Background(), dataDir: dataDir, em: emitter.New()}
	if err := e.init(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *env) init() error {
	// open or create db
	var err error

	rest.Host = "app.ellipx.com"

	// make sure dataDir exists and is a directory
	if st, err := os.Stat(e.dataDir); err != nil {
		err = os.MkdirAll(e.dataDir, 0755)
		if err != nil {
			return err
		}
	} else if !st.IsDir() {
		return errors.New("dataDir exists but is not a directory")
	}

	// connect Spot using dynamic (temporary) key
	e.spot, err = spotlib.New(map[string]string{"project": "libwallet"})
	if err != nil {
		return err
	}
	go e.handleStatusEvent(e.spot.Events.On("status"))

	// open bolt db
	e.db, err = bolt.Open(filepath.Join(e.dataDir, "data.db"), 0600, nil)
	if err != nil {
		return err
	}

	currentVersion := []byte{0, 0, 0, 3}

	if v, err := e.DBSimpleGet([]byte("info"), []byte("version")); err == nil && bytes.Equal(v, currentVersion) {
		// all good
	} else {
		// set version
		e.DBSimpleSet([]byte("info"), []byte("version"), currentVersion)
		// because previously we had invalid wallets created, erase it
		e.dbDeleteBucket([]byte("wallet"))
		e.dbDeleteBucket([]byte("account"))
		e.dbDeleteBucket([]byte("network"))
	}

	if _, err := e.DBSimpleGet([]byte("info"), []byte("first_run")); err != nil {
		// first run?
		now := ellipxobj.NewTimeId().Bytes(nil)
		e.DBSimpleSet([]byte("info"), []byte("first_run"), now)
	}

	// open sql database
	e.sql, err = gorm.Open(sqlite.New(sqlite.Config{DriverName: "sqlite", DSN: filepath.Join(e.dataDir, "sql.db") + "?_pragma=journal_mode(WAL)"}), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true, NoLowerCase: true}})
	if err != nil {
		return err
	}

	// create tables
	wltasset.InitEnv(e)
	e.sql.AutoMigrate(&request{})
	e.sql.AutoMigrate(&currentItem{})
	e.sql.AutoMigrate(&connectedSite{})
	wltnet.InitEnv(e)
	wlttx.InitEnv(e)
	wltacct.InitEnv(e)
	wltwallet.InitEnv(e)
	wltcontact.InitEnv(e)
	wltnft.InitEnv(e)
	wltcrash.InitEnv(e)

	return nil
}

func (e *env) Emitter() *emitter.Hub {
	return e.em
}

func (e *env) Spot() *spotlib.Client {
	return e.spot
}

func (e *env) ListHelper(ctx context.Context, target any, sort string, searchKey ...string) error {
	var c *apirouter.Context
	if ctx != nil {
		ctx.Value(&c)
	}

	tx := e.sql
	if c != nil {
		tx = tx.Scopes(c.Paginate(50))
	}
	if sort != "" {
		tx = tx.Order(sort)
	}

	if len(searchKey) > 0 {

		if c != nil {
			where := make(map[string]any)
			for _, k := range searchKey {
				if v := c.GetParam(k); v != nil {
					where[k] = v
				}
			}
			if len(where) > 0 {
				tx = tx.Where(where)
			}
		}
	}

	tx = tx.Find(target)
	return tx.Error
}

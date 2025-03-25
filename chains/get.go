package chains

import (
	"encoding/json"
	"sync"
)

var (
	chainCache = make(map[uint64]*ChainInfo)
	lk         sync.RWMutex
)

func Get(id uint64) *ChainInfo {
	buf, ok := chainJSON[id]
	if !ok {
		return nil
	}
	var res *ChainInfo
	err := json.Unmarshal([]byte(buf), &res)
	if err != nil {
		return nil
	}
	return res
}

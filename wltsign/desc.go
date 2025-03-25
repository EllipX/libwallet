package wltsign

type KeyDescription struct {
	Type string // StoreKey | RemoteKey | Plain | Password
	Key  string // if StoreKey, a key. If RemoteKey, a RemoteKey. If Plain, ignored.
	Id   string // if of key
}

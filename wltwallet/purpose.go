package wltwallet

type keyUsagePurpose int

const (
	keySignPurpose keyUsagePurpose = iota
	keyResharePurpose
	keyRecryptPurpose
)

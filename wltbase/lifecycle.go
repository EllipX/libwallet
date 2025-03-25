package wltbase

import "github.com/KarpelesLab/pobj"

func init() {
	pobj.RegisterStatic("Lifecycle:update", lifecycleUpdate)
}

func lifecycleUpdate(in struct {
	Status string `json:"status"`
}) (any, error) {
	return map[string]any{"status": in.Status}, nil
}

package wltutil

import (
	"context"
	"time"

	"github.com/KarpelesLab/apirouter"
)

func BroadcastMsg(ev string, data any) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	apirouter.BroadcastJson(ctx, map[string]any{"result": "event", "event": ev, "data": data})
}

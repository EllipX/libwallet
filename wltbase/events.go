package wltbase

import (
	"context"

	"github.com/KarpelesLab/apirouter"
	"github.com/KarpelesLab/emitter"
)

func (e *env) handleStatusEvent(ch <-chan *emitter.Event) {
	ctx := context.Background()

	for ev := range ch {
		data := map[string]any{
			"online": false,
		}
		if v, _ := emitter.Arg[int](ev, 0); v == 1 {
			data["online"] = true
		}
		apirouter.BroadcastJson(ctx, map[string]any{"result": "event", "event": "online_status", "data": data})
	}
}

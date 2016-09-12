package lego

import (
	"time"

	"github.com/stairlin/lego/stats"
)

// heartbeat sends a heartbeat to stats periodically
type hearbeat struct {
	app  *App
	stop chan bool
}

// Start starts sending a heartbeat
func (h *hearbeat) Start() {
	h.stop = make(chan bool, 1)

	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-h.stop:
			break
		case <-tick:
			metric := stats.Metric{
				Key: "heartbeat",
				Values: map[string]interface{}{
					"value": 1,
				},
				T: time.Now(),
				Meta: map[string]string{
					"type": h.app.Ctx().Name(),
				},
			}
			h.app.Ctx().Stats().Add(&metric)
		}
	}
}

// Stop stops sending a heartbeat
func (h *hearbeat) Stop() {
	h.stop <- true
}

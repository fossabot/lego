package lego

import "time"

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
			tags := map[string]string{
				"type": h.app.Ctx().Name(),
			}

			h.app.Ctx().Stats().Inc("heartbeat", tags)
		}
	}
}

// Stop stops sending a heartbeat
func (h *hearbeat) Stop() {
	h.stop <- true
}

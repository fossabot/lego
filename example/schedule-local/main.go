package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/stairlin/lego"
	"github.com/stairlin/lego/ctx/journey"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/net/http"
	"github.com/stairlin/lego/schedule"
)

type AppConfig struct {
	Foo string `json:"foo"`
}

func main() {
	// Create lego
	config := &AppConfig{}
	app, err := lego.New("api", config)
	if err != nil {
		fmt.Println("Error initialising lego", err)
		os.Exit(1)
	}

	if err := start(app); err != nil {
		fmt.Println("Error starting lego", err)
		os.Exit(1)
	}
}

func start(app *lego.App) error {
	// Create scheduler
	scheduler := app.Scheduler()
	scheduler.HandleFunc("foo", func(ctx journey.Ctx, id string, data []byte) error {
		ctx.Trace("schedule.process", "Process job",
			log.String("job_id", id),
			log.String("job_data", string(data)),
		)
		return nil
	})
	scheduler.HandleFunc("err", func(ctx journey.Ctx, id string, data []byte) error {
		ctx.Trace("schedule.process", "Process job",
			log.String("job_id", id),
			log.String("job_data", string(data)),
		)
		return errors.New("job failed")
	})
	scheduler.HandleFunc("panic", func(ctx journey.Ctx, id string, data []byte) error {
		ctx.Trace("schedule.process", "Process job",
			log.String("job_id", id),
			log.String("job_data", string(data)),
		)
		panic("BOOM!")
	})

	// Register HTTP handler
	h := handler{scheduler: scheduler}
	s := http.NewServer()
	s.HandleFunc("/job/{target}", http.POST, h.scheduleJob)
	app.RegisterServer("127.0.0.1:3000", s)

	// Start serving requests
	err := app.Serve()
	if err != nil {
		return err
	}
	return nil
}

type handler struct {
	scheduler schedule.Scheduler
}

type job struct {
	Target string
	In     time.Duration
	Data   string
}

func (h *handler) scheduleJob(ctx journey.Ctx, w http.ResponseWriter, r *http.Request) {
	ctx.Trace("schedule", "Schedule job")

	// Unmarshal job from request body
	j := job{}
	if err := json.NewDecoder(r.HTTP.Body).Decode(&j); err != nil {
		ctx.Warning("schedule.decode.err", "Error decoding request body", log.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Set target from URL parameter
	j.Target = r.Params["target"]

	// Schedule job
	id, err := h.scheduler.In(ctx, j.In*time.Second, j.Target, []byte(j.Data))
	if err != nil {
		ctx.Warning("schedule.create.err", "Error creating job", log.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(id))
}

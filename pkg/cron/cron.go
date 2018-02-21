package cron

import (
	"errors"
	"sync"
	"time"

	"github.com/codefresh-io/cronus/pkg/hermes"
	"github.com/codefresh-io/cronus/pkg/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
)

type (
	//CronJobEngine basic interface to underlying cron job engine
	CronJobEngine interface {
		Start()
		AddJob(spec string, cmd cron.Job) (cron.EntryID, error)
		Remove(id cron.EntryID)
	}

	// Runner CRON runner
	Runner struct {
		hermesSvc hermes.Service
		store     types.EventStore
		cron      CronJobEngine
		jobs      *sync.Map
	}

	// JobManager job manager interface to add/remove running jobs
	JobManager interface {
		AddCronJob(e types.Event) error
		RemoveCronJob(uri string) error
		TriggerEvent(e types.Event) error
	}

	// TriggerJob struct that keeps event and triggers cron job execution
	TriggerJob struct {
		manager JobManager
		event   types.Event
	}
)

// Run implements cron.Job interface
func (job *TriggerJob) Run() {
	log.Debug("running cron job")
	err := job.manager.TriggerEvent(job.event)
	if err != nil {
		log.WithError(err).Error("failed to trigger event pipelines")
	}
}

// NewCronRunner create new CRON runner with default cron job engine
func NewCronRunner(store types.EventStore, svc hermes.Service) *Runner {
	return NewCronRunnerFull(store, svc, cron.New())
}

// NewCronRunnerFull create new CRON runner with pluggable cron job engine
func NewCronRunnerFull(store types.EventStore, svc hermes.Service, cron CronJobEngine) *Runner {
	log.Debug("creating new cron runner")
	runner := new(Runner)
	runner.hermesSvc = svc
	runner.store = store
	runner.cron = cron
	runner.jobs = new(sync.Map)
	runner.init()
	return runner
}

func (r *Runner) init() {
	log.Debug("initializing cron runner")
	// get all stored events
	events, err := r.store.GetAllEvents()
	if err != nil {
		log.WithError(err).Error("load existing cron job events")
	}
	// add already defined CRON jobs
	for _, e := range events {
		log.WithFields(log.Fields{
			"expression":  e.Expression,
			"message":     e.Message,
			"description": e.Description,
		}).Debug("creating a cron job based on event spec")
		job, err := r.cron.AddJob(e.Expression, &TriggerJob{manager: r, event: e})
		if err != nil {
			log.WithError(err).Warn("failed to create a new cron job")
		}
		// store job ID
		r.jobs.Store(types.GetURI(e), job)
	}
	// start CRON job runner
	r.cron.Start()
}

// TriggerEvent trigger event
func (r *Runner) TriggerEvent(e types.Event) error {
	log.WithFields(log.Fields{
		"cron":    e.Expression,
		"message": e.Message,
	}).Debug("triggering cron event")

	// create normalized event
	event := hermes.NewNormalizedEvent()
	// reuse secret from event creation
	event.Secret = e.Secret

	// pass event details
	event.Variables["message"] = e.Message
	event.Variables["timestamp"] = time.Now().Format(time.RFC3339)

	// attempt to invoke trigger
	log.Debug("invoke hermes API to trigger event")
	return r.hermesSvc.TriggerEvent(types.GetURI(e), event)
}

// AddCronJob add new CRON job
func (r *Runner) AddCronJob(e types.Event) error {
	log.WithField("event", e).Debug("adding new cron job")
	uri := types.GetURI(e)
	_, ok := r.jobs.Load(uri)
	if ok {
		return errors.New("this cron job already exist")
	}
	// add cron job to job runner
	job, err := r.cron.AddJob(e.Expression, &TriggerJob{manager: r, event: e})
	if err != nil {
		return errors.New("failed to create a new cron job")
	}
	// store cron event into persistent store
	err = r.store.StoreEvent(e)
	if err != nil {
		// remove cron job from job runner
		r.cron.Remove(job)
		return err
	}
	// store job ID to global jobs map
	r.jobs.Store(uri, job)
	return nil
}

// RemoveCronJob remove CRON job
func (r *Runner) RemoveCronJob(uri string) error {
	log.WithField("event-uri", uri).Debug("removing cron job")
	job, ok := r.jobs.Load(uri)
	if !ok {
		return errors.New("cron job not found")
	}
	// remove cron job from job runner
	r.cron.Remove(job.(cron.EntryID))
	// store job ID to global jobs map
	r.jobs.Delete(uri)
	// remove cron event from persistent store
	return r.store.DeleteEvent(uri)
}

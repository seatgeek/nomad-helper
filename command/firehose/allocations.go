package firehose

import (
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"time"

	"fmt"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/command/firehose/sink"
)

// Sink ...s
type Sink interface {
	Start() error
	Stop()
	Put(data []byte) error
}

// AllocationFirehose ...
type AllocationFirehose struct {
	nomadClient     *nomad.Client
	consulClient    *consul.Client
	consulSessionID string
	consulLock      *consul.Lock
	stopCh          chan struct{}
	lastChangeTime  int64
	sink            Sink
}

// AllocationUpdate ...
type AllocationUpdate struct {
	Name               string
	AllocationID       string
	DesiredStatus      string
	DesiredDescription string
	ClientStatus       string
	ClientDescription  string
	JobID              string
	GroupName          string
	TaskName           string
	EvalID             string
	TaskState          string
	TaskFailed         bool
	TaskStartedAt      time.Time
	TaskFinishedAt     time.Time
	TaskEvent          *nomad.TaskEvent
}

// NewAllocationFirehose ...
func NewAllocationFirehose(lock *consul.Lock, sessionID string) (*AllocationFirehose, error) {
	nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
	if err != nil {
		return nil, err
	}

	consulClient, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, err
	}

	sink, err := getSink()
	if err != nil {
		return nil, err
	}

	return &AllocationFirehose{
		nomadClient:     nomadClient,
		consulClient:    consulClient,
		consulSessionID: sessionID,
		consulLock:      lock,
		sink:            sink,
	}, nil
}

func getSink() (Sink, error) {
	sinkType := os.Getenv("SINK_TYPE")
	if sinkType == "" {
		return nil, fmt.Errorf("Missing SINK_TYPE: amqp or kinesis")
	}

	switch sinkType {
	case "amqp":
		fallthrough
	case "rabbitmq":
		return sink.NewRabbitmq()
	case "kinesis":
		return sink.NewKinesis()
	default:
		return nil, fmt.Errorf("Invalid SINK_TYPE: amqp or kinesis")
	}
}

// Start the firehose
func (f *AllocationFirehose) Start() error {
	// Restore the last change time from Consul
	err := f.restoreLastChangeTime()
	if err != nil {
		return err
	}

	go f.sink.Start()

	// Stop chan for all tasks to depend on
	f.stopCh = make(chan struct{})

	// setup signal handler for graceful shutdown
	go f.signalHandler()

	// watch for allocation changes
	go f.watch()

	//
	go f.persistLastChangeTime()

	// wait forever for a stop signal to happen
	for {
		select {
		case <-f.stopCh:
			return nil
		}
	}
}

// Stop the firehose
func (f *AllocationFirehose) Stop() {
	close(f.stopCh)
	f.sink.Stop()
	f.writeLastChangeTime()
}

// Read the Last Change Time from Consul KV, so we don't re-process tasks over and over on restart
func (f *AllocationFirehose) restoreLastChangeTime() error {
	kv, _, err := f.consulClient.KV().Get(consulLockValue, &consul.QueryOptions{})
	if err != nil {
		return err
	}

	// Ensure we got
	if kv != nil && kv.Value != nil {
		sv := string(kv.Value)
		v, err := strconv.ParseInt(sv, 10, 64)
		if err != nil {
			return err
		}

		f.lastChangeTime = v
		log.Infof("Restoring Last Change Time to %s", sv)
	} else {
		log.Info("No Last Change Time restore point, starting from scratch")
	}

	return nil
}

// Write the Last Change Time to Consul so if the process restarts,
// it will try to resume from where it left off, not emitting tons of double events for
// old events
func (f *AllocationFirehose) persistLastChangeTime() {
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-f.stopCh:
			break
		case <-ticker.C:
			f.writeLastChangeTime()
		}
	}
}

func (f *AllocationFirehose) writeLastChangeTime() {
	v := strconv.FormatInt(f.lastChangeTime, 10)

	log.Infof("Writing lastChangedTime to KV: %s", v)
	kv := &consul.KVPair{
		Key:     consulLockValue,
		Value:   []byte(v),
		Session: f.consulSessionID,
	}
	_, err := f.consulClient.KV().Put(kv, &consul.WriteOptions{})
	if err != nil {
		log.Error(err)
	}
}

// Publish an update from the firehose
func (f *AllocationFirehose) Publish(update *AllocationUpdate) {
	log.Infof("%s -> %s -> %s: %s", update.JobID, update.GroupName, update.TaskName, update.TaskEvent.DriverMessage)
	b, err := json.Marshal(update)
	if err != nil {
		log.Error(err)
	}

	log.Info(string(b))

	f.sink.Put(b)
}

// Continously watch for changes to the allocation list and publish it as updates
func (f *AllocationFirehose) watch() {
	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

	newMax := f.lastChangeTime

	for {
		allocations, meta, err := f.nomadClient.Allocations().List(q)
		if err != nil {
			log.Errorf("Unable to fetch allocations: %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		remoteWaitIndex := meta.LastIndex
		localWaitIndex := q.WaitIndex

		// Only work if the WaitIndex have changed
		if remoteWaitIndex <= localWaitIndex {
			log.Debugf("Allocations index is unchanged (%d <= %d)", remoteWaitIndex, localWaitIndex)
			continue
		}

		log.Debugf("Allocations index is changed (%d <> %d)", remoteWaitIndex, localWaitIndex)

		// Iterate allocations and find events that have changed since last run
		for _, allocation := range allocations {
			for taskName, taskInfo := range allocation.TaskStates {
				for _, taskEvent := range taskInfo.Events {
					if taskEvent.Time <= f.lastChangeTime {
						continue
					}

					if taskEvent.Time > newMax {
						newMax = taskEvent.Time
					}

					payload := &AllocationUpdate{
						Name:               allocation.Name,
						AllocationID:       allocation.ID,
						EvalID:             allocation.EvalID,
						DesiredStatus:      allocation.DesiredStatus,
						DesiredDescription: allocation.DesiredDescription,
						ClientStatus:       allocation.ClientStatus,
						ClientDescription:  allocation.ClientDescription,
						JobID:              allocation.JobID,
						GroupName:          allocation.TaskGroup,
						TaskName:           taskName,
						TaskEvent:          taskEvent,
						TaskState:          taskInfo.State,
						TaskFailed:         taskInfo.Failed,
						TaskStartedAt:      taskInfo.StartedAt,
						TaskFinishedAt:     taskInfo.FinishedAt,
					}

					f.Publish(payload)
				}
			}
		}

		// Update WaitIndex and Last Change Time for next iteration
		q.WaitIndex = meta.LastIndex
		f.lastChangeTime = newMax
	}
}

// Close the stopCh if we get a signal, so we can gracefully shut down
func (f *AllocationFirehose) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		log.Info("Caught signal, releasing lock and stopping...")
		f.Stop()
	case <-f.stopCh:
		break
	}
}

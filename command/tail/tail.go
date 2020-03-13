package tail

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/seatgeek/nomad-helper/helpers"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

const (
	// bytesToLines is an estimation of how many bytes are in each log line.
	// This is used to set the offset to read from when a user specifies how
	// many lines to tail from.
	bytesToLines int64 = 120

	// defaultTailLines is the number of lines to tail by default if the value
	// is not overridden.
	defaultTailLines int64 = 15
)

func Run(c *cli.Context) error {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	alloc, err := helpers.FindAllocation(c, nomadClient)
	if err != nil {
		return err
	}

	taskName, err := helpers.FindTask(alloc, c.String("task"))
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	ch := make(chan interface{}, 0)

	if c.BoolT("stdout") {
		wg.Add(1)
		logger := log.WithField("log_type", "stdout")
		go Tail(getWriter("stdout", c.String("writer"), c.String("theme")), "stderr", taskName, alloc, nomadClient, &wg, logger)
	}

	if c.BoolT("stderr") {
		wg.Add(1)
		logger := log.WithField("log_type", "stderr")
		go Tail(getWriter("stderr", c.String("writer"), c.String("theme")), "stdout", taskName, alloc, nomadClient, &wg, logger)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-sigs:
		log.Info("Caught signal, exiting...")
	case <-ch:
		log.Info("Tailing completed, exiting...")
	}

	return nil
}

func getWriter(target, kind, theme string) io.Writer {
	switch kind {
	case "color":
		return colorLogWriter{Type: target, Theme: theme}
	case "simple":
		return simpleLogWriter{Type: target}
	case "raw":
		return rawLogWriter{Type: target}
	default:
		panic("Invalid log ")
	}
}

func Tail(wr io.Writer, logType, task string, alloc *api.Allocation, client *api.Client, wg *sync.WaitGroup, logger *log.Entry) {
	var err error
	var r io.ReadCloser
	var readErr error

	defer wg.Done()

	// Parse the offset
	var offset = defaultTailLines * bytesToLines
	r, readErr = followFile(client, alloc, logger, true, task, logType, api.OriginEnd, offset)
	r = NewLineLimitReader(r, int(defaultTailLines), int(defaultTailLines*bytesToLines), 1*time.Second)
	if readErr != nil {
		readErr = fmt.Errorf("Error tailing file: %v", readErr)
	}

	if readErr != nil {
		logger.Error(readErr.Error())
		return
	}

	defer r.Close()
	_, err = io.Copy(wr, r)
	if err != nil {
		logger.Error(fmt.Sprintf("error following logs: %s", err))
		return
	}
}

func followFile(client *api.Client, alloc *api.Allocation, logger *log.Entry,
	follow bool, task, logType, origin string, offset int64) (io.ReadCloser, error) {

	cancel := make(chan struct{})
	frames, errCh := client.AllocFS().Logs(alloc, follow, task, logType, origin, offset, cancel, nil)
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Create a reader
	var r io.ReadCloser
	frameReader := api.NewFrameReader(frames, errCh, cancel)
	frameReader.SetUnblockTime(500 * time.Millisecond)
	r = frameReader

	go func() {
		<-signalCh
		r.Close()
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			if isAllocDone(client, alloc, logger, task) {
				r.Close()
				ticker.Stop()
				return
			}
			<-ticker.C
		}
	}()

	return r, nil
}

func isAllocDone(client *api.Client, alloc *api.Allocation, logger *log.Entry, task string) bool {
	allocation, err := helpers.FindAllocationByID(alloc.ID, client)
	if err != nil {
		return true
	}

	doneAllocStates := map[string]bool{
		"complete": true,
		"failed":   true,
		"lost":     true,
	}

	return doneAllocStates[allocation.ClientStatus]
}

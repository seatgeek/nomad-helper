package firehose

import (
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

const (
	consulLockKey   = "service/nomad-helper/firehose.lock"
	consulLockValue = "service/nomad-helper/firehose.value"
)

// App ...
func App() error {
	lock, sessionID, err := waitForLock()
	if err != nil {
		return err
	}
	defer lock.Unlock()

	worker, err := NewAllocationFirehose(lock, sessionID)
	if err != nil {
		return err
	}

	err = worker.Start()
	if err != nil {
		return err
	}

	return nil
}

// Wait for exclusive lock to the Consul KV key.
// This will ensure we are the only applicating running and processing
// allocation events to the firehose
func waitForLock() (*consul.Lock, string, error) {
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, "", err
	}

	log.Info("Trying to acquire leader lock")
	sessionID, err := session(client)
	if err != nil {
		return nil, "", err
	}

	lock, err := client.LockOpts(&consul.LockOptions{
		Key:     consulLockKey,
		Session: sessionID,
	})
	if err != nil {
		return nil, "", err
	}

	_, err = lock.Lock(nil)
	if err != nil {
		return nil, "", err
	}

	log.Info("Lock acquired")
	return lock, sessionID, nil
}

// Create a Consul session used for locks
func session(c *consul.Client) (string, error) {
	s := c.Session()
	se := &consul.SessionEntry{
		Name: "nomad-helper-firehose",
		TTL:  "15s",
	}

	id, _, err := s.Create(se, nil)
	if err != nil {
		return "", err
	}

	return id, nil
}

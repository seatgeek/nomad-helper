package firehose

// AllocationFirehose ...
type AllocationFirehose struct {
}

// AllocationUpdate ...
type AllocationUpdate struct {
}

// NewAllocationFirehose ...
func NewAllocationFirehose() (*AllocationFirehose, error) {
	return &AllocationFirehose{}, nil
}

// Start the firehose
func (f *AllocationFirehose) Start() {

}

// Stop the firehose
func (f *AllocationFirehose) Stop() {

}

// Publish an update from the firehose
func (f *AllocationFirehose) Publish(update *AllocationUpdate) {

}

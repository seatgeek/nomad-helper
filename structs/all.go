package structs

// JobState ...
type JobState map[string]int

// NomadState ...
type NomadState struct {
	Info map[string]string
	Jobs map[string]TaskGroupState
}

// TaskGroupState ...
type TaskGroupState map[string]int

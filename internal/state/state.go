// Package state defines the supervisor's lifecycle states. The transitions are
// driven by the supervisor loop; this package is just the vocabulary so other
// packages (status reporting, persistence) can refer to states by name.
package state

// State is one node of the supervisor state machine.
type State string

const (
	// Running: the agent is working; the supervisor only observes.
	Running State = "RUNNING"
	// Limited: a usage-limit message was detected; resolving the reset time.
	Limited State = "LIMITED"
	// Waiting: sleeping until reset + buffer, showing a countdown.
	Waiting State = "WAITING"
	// Resuming: reset reached; confirming idle and injecting the resume prompt.
	Resuming State = "RESUMING"
	// Detached: supervisor passive; the agent session belongs to the user.
	Detached State = "DETACHED"
	// Ended: the session/agent is gone (exited or killed out from under us);
	// there is nothing left to watch and the supervisor stops.
	Ended State = "ENDED"
)

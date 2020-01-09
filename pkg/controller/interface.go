package controller

import (
	"fmt"
)

// Status of a Thing, like "doesn't exist", "created", "spawned", etc.
type Status interface {
	Satisfies(other Status) bool
	Equals(other Status) bool
}

// Action to be performed on a Thing to change its Status from one to other
type Action interface {
	Apply() error
	// IsExclusive() being true means the action can not be run in parallel with
	// any other actions and that we shouldn't print "running..." messages.
	// One example of such action is running a user application on remote cluster.
	IsExclusive() bool
	Prerequisites() ([]Target, error)
}

// Target is a pair of (Thing, desired Status) which we want to reach
// either because user wanted to or because it's a prerequisite for
// some Action
type Target struct {
	Thing         Thing
	DesiredStatus Status
	MatchExact    bool
}

func (t Target) String() string {
	return fmt.Sprintf("Target{%s:%s[exact=%t]}", t.Thing, t.DesiredStatus, t.MatchExact)
}

// Thing is the entity that has Status and know its Status transition graph
// plus Actions needed to change Status from one to another.
// Examples: image, cluster, run-script-task
type Thing interface {
	Status() Status
	SetStatus(status Status) error
	// get all statuses from which "to" status is reachable
	GetTransitions(to Status) ([]Status, error)
	// get the action to move Thing from its "current" status to "Target" status
	GetAction(current Status, target Status) (Action, error)
	Equals(other Thing) bool
}

// ReachTarget plans and performs the execution of the graph so that given
// Thing reaches desired status, e.g. cluster reaches "spawned" status.
// Thing is considered done when its status Satisfies desired
func ReachTarget(thing Thing, desiredStatus Status, simulate bool) error {
	return ReachTargetEx(Target{thing, desiredStatus, false}, simulate)
}

// ReachTargetEx does the same as ReachTarget but gives more flexibility in composing the target
func ReachTargetEx(target Target, simulate bool) error {
	executor := executorState{
		done:     make(chan executorTaskState),
		simulate: simulate,
	}

	return executor.execute(target)
}

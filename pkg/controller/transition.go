package controller

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type transition struct {
	target        Target
	action        Action
	fromStatus    Status
	prerequisites []Target
	started       time.Time
}

func (t transition) String() string {
	return fmt.Sprintf("[Action{%s} <== %s]", t.action, t.target)
}

func targetDone(t Target) bool {
	return compareStatus(t.Thing.Status(), t.DesiredStatus, t.MatchExact)
}

func compareStatus(s1, s2 Status, matchExact bool) bool {
	if matchExact {
		return s1.Equals(s2)
	}

	return s1.Satisfies(s2)
}

func buildStatusChains(thing Thing, desired Status, matchExact bool,
	visited map[Status]bool) ([][]Status, error) {
	visited[desired] = true

	defer func() {
		visited[desired] = false
	}()

	if compareStatus(thing.Status(), desired, matchExact) {
		log.WithFields(log.Fields{
			"thing":       thing,
			"from-status": thing.Status(),
			"to-status":   desired,
			"exact":       matchExact,
		}).Trace("current status satisfies desired status")

		return [][]Status{}, nil
	}

	transitions, err := thing.GetTransitions(desired)
	if err != nil {
		return [][]Status{}, err
	}

	result := [][]Status{}

	for _, from := range transitions {
		if visited[from] {
			continue
		}

		chains, err := buildStatusChains(thing, from, matchExact, visited)
		if err == nil {
			if len(chains) != 0 {
				for idx, chain := range chains {
					chains[idx] = append(chain, from)
				}
			} else {
				chains = make([][]Status, 1)
				chains[0] = []Status{from}
			}

			result = append(result, chains...)
		}
	}

	if len(result) == 0 {
		return [][]Status{}, fmt.Errorf("cannot reach %s from %s", desired, thing.Status())
	}

	return result, nil
}

func (exec *executorState) makeAction(target Target, toStatus Status) (*transition, error) {
	logger := log.WithFields(log.Fields{
		"thing":       target.Thing,
		"from-status": target.Thing.Status(),
		"to-status":   toStatus,
		"executor":    exec,
	})

	action, err := target.Thing.GetAction(target.Thing.Status(), toStatus)
	if err != nil {
		logger.Errorf("makeAction: cannot get action: %s", err)
		return nil, err
	}

	logger = logger.WithField("action", action)
	logger.Info("makeAction: got action")

	actPrereqs, err := action.Prerequisites()
	if err != nil {
		logger.Errorf("makeAction: cannot get action prerequisites: %s", err)
		return nil, err
	}

	logger.Infof("makeAction: got action prerequisites: %v", actPrereqs)

	allPrereqs := true

	for _, preTarget := range actPrereqs {
		logPre := logger.WithField("prerequisite", preTarget)

		if !targetDone(preTarget) {
			if preTransit, err := exec.findTransition(preTarget); err == nil {
				if preTransit != nil {
					return preTransit, nil
				}
				// target not done but nil transition - target blocked
				allPrereqs = false
			} else {
				return nil, err
			}
		} else {
			logPre.Info("makeAction: prerequisite already satisfied")
		}
	}

	if !allPrereqs {
		logger.Info("makeAction: some prerequisites unsatisfied but currently blocked")
		return nil, nil
	}

	return &transition{
		target: Target{
			Thing:         target.Thing,
			DesiredStatus: toStatus,
			MatchExact:    target.MatchExact,
		},
		fromStatus:    target.Thing.Status(),
		action:        action,
		prerequisites: actPrereqs,
	}, nil
}

func (exec *executorState) findTransition(target Target) (*transition, error) {
	logger := log.WithFields(log.Fields{
		"thing":       target.Thing,
		"from-status": target.Thing.Status(),
		"to-status":   target.DesiredStatus,
		"executor":    exec,
	})

	visited := make(map[Status]bool)

	chains, err := buildStatusChains(target.Thing, target.DesiredStatus, target.MatchExact, visited)
	if err != nil {
		logger.Errorf("findTransition: cannot build status chains: %s", err)
		return nil, err
	}

	minLen, minIdx := len(chains[0]), 0

	for idx, chain := range chains {
		if len(chain) < minLen {
			minLen = len(chain)
			minIdx = idx
		}
	}

	chain := chains[minIdx]
	if len(chain) == 0 {
		logger.Infof("findTransition: already at desired status, nothing to do")
		return nil, nil
	}

	if !chain[0].Equals(target.Thing.Status()) {
		logger.Fatalf("findTransition: first element in status chain %s is not equal to from-status", chain[0])
	}

	chain = append(chain, target.DesiredStatus)
	chainStrs := []string{}

	for _, status := range chain {
		chainStrs = append(chainStrs, fmt.Sprintf("%s", status))
	}

	logger.Infof("findTransition: created status chain: %s", strings.Join(chainStrs, " -> "))

	toStatus := chain[1]

	transit, err := exec.makeAction(target, toStatus)
	if err != nil {
		return nil, err
	}

	if transit != nil && exec.canExecute(transit) {
		return transit, nil
	}

	logger.Info("findTransition: cannot execute transition, currently blocked by others")

	return nil, nil
}

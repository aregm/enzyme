package controller

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type executorTaskState struct {
	task       *transition
	fromStatus Status
	err        error
}

type executorState struct {
	running           []Thing
	activeTransitions []transition
	constraints       []Target
	done              chan executorTaskState
	simulate          bool
}

func targetsConflict(t1, t2 Target) bool {
	if !t1.Thing.Equals(t2.Thing) {
		return false
	}

	return !(compareStatus(t1.DesiredStatus, t2.DesiredStatus, t1.MatchExact) &&
		compareStatus(t2.DesiredStatus, t1.DesiredStatus, t2.MatchExact))
}

func targetsEqual(t1, t2 Target) bool {
	return t1.Thing.Equals(t2.Thing) && t1.DesiredStatus.Equals(t2.DesiredStatus)
}

func (exec *executorState) canExecute(t *transition) bool {
	if t.action.IsExclusive() && len(exec.running) != 0 {
		log.WithField("action", t.action).Infof(
			"action is exclusive but currently there are %d other actions running, so not running it",
			len(exec.running))

		return false
	}

	for _, runner := range exec.running {
		if runner.Equals(t.target.Thing) {
			return false
		}
	}

	for _, prereq := range t.prerequisites {
		for _, constraint := range exec.constraints {
			if targetsConflict(constraint, prereq) {
				return false
			}
		}
	}

	return true
}

func (exec *executorState) runTask(task *transition) {
	exec.running = append(exec.running, task.target.Thing)
	exec.constraints = append(exec.constraints, task.prerequisites...)

	if exec.simulate {
		fmt.Printf("simulating:\t%s\n", task.action)
	} else {
		task.started = time.Now()
		fmt.Printf("Starting: %s\n", task.action)
	}

	go func() {
		var err error = nil
		if !exec.simulate {
			err = task.action.Apply()
		}
		result := executorTaskState{
			task:       task,
			fromStatus: task.fromStatus,
			err:        err,
		}
		exec.done <- result
	}()

	exec.activeTransitions = append(exec.activeTransitions, *task)
}

func (exec *executorState) finishTask(result executorTaskState) error {
	found := false

	for idx, runner := range exec.running {
		if runner.Equals(result.task.target.Thing) {
			exec.running = append(exec.running[:idx], exec.running[idx+1:]...)
			found = true

			break
		}
	}

	if !found {
		panic(fmt.Sprintf("cannot find task %v in list of running tasks", result.task.target.Thing))
	}

	found = false

	for idx, transition := range exec.activeTransitions {
		if transition.target.Thing.Equals(result.task.target.Thing) {
			exec.activeTransitions = append(exec.activeTransitions[:idx], exec.activeTransitions[idx+1:]...)
			found = true

			break
		}
	}

	if !found {
		panic(fmt.Sprintf("cannot find transition for task %v in list of running transitions",
			result.task.target.Thing))
	}

	for _, prereq := range result.task.prerequisites {
		found = false

		for idx, constraint := range exec.constraints {
			if targetsEqual(constraint, prereq) {
				exec.constraints = append(exec.constraints[:idx], exec.constraints[idx+1:]...)
				found = true

				break
			}
		}

		if !found {
			panic(fmt.Sprintf("cannot find target %v in list of constraints", prereq))
		}
	}

	if result.err == nil {
		if !result.task.target.Thing.Status().Equals(result.fromStatus) {
			log.WithFields(log.Fields{
				"thing":           result.task.target.Thing,
				"expected status": result.fromStatus,
				"current status":  result.task.target.Thing.Status(),
			}).Errorf("transition failed, unexpected current status")
			fmt.Printf("Failed: %s [took %s]\n", result.task.action, time.Since(result.task.started))

			return fmt.Errorf("unexpected current status (%v), expected %v",
				result.task.target.Thing.Status(), result.fromStatus)
		}

		if err := result.task.target.Thing.SetStatus(result.task.target.DesiredStatus); err != nil {
			log.WithFields(log.Fields{
				"thing":          result.task.target.Thing,
				"desired status": result.task.target.DesiredStatus,
			}).Errorf("cannot set status: %s", err)
			fmt.Printf("Failed: %s [took %s]\n", result.task.action, time.Since(result.task.started))

			return err
		}

		if exec.simulate {
			fmt.Printf("simulated:\t%s\n", result.task.action)
		} else {
			fmt.Printf("Complete: %s [took %s]\n", result.task.action, time.Since(result.task.started))
		}
	} else {
		log.WithFields(log.Fields{
			"thing":          result.task.target.Thing,
			"desired status": result.task.target.DesiredStatus,
		}).Errorf("transition failed: %s", result.err)
		fmt.Printf("Failed: %s [took %s]\n", result.task.action, time.Since(result.task.started))
	}

	return result.err
}

func (exec *executorState) waitForAny(ticker *time.Ticker) executorTaskState {
	if exec.simulate {
		return <-exec.done
	}

	if len(exec.activeTransitions) == 1 && exec.activeTransitions[0].action.IsExclusive() {
		// don't interrupt exclusive action by printing anything, wait for it patiently
		return <-exec.done
	}

	for {
		select {
		case res := <-exec.done:
			return res
		case <-ticker.C:
			for _, transition := range exec.activeTransitions {
				fmt.Printf("Running: %s [%s]\n", transition.action, time.Since(transition.started))
			}
		}
	}
}

func (exec *executorState) execute(target Target) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for !targetDone(target) {
		candidate, err := exec.findTransition(target)
		if err != nil {
			log.WithFields(log.Fields{
				"target":   target,
				"executor": exec,
			}).Errorf("execute: cannot find a transition: %s", err)

			return err
		}

		if candidate != nil {
			log.WithFields(log.Fields{
				"candidate": candidate,
				"executor":  exec,
			}).Info("execute: executing candidate")
			exec.runTask(candidate)
		} else {
			if len(exec.running) == 0 {
				return fmt.Errorf("execute: blocked execution - nothing runs but no candidate found")
			}

			taskResult := exec.waitForAny(ticker)

			if err := exec.finishTask(taskResult); err != nil {
				// empty up running tasks
				for len(exec.running) != 0 {
					taskResult = exec.waitForAny(ticker)

					if err := exec.finishTask(taskResult); err != nil {
						log.WithField("task", taskResult.task.String).Fatalf(
							"execute: finishTask failed: %s", err)
					}
				}

				return err
			}
		}
	}

	return nil
}

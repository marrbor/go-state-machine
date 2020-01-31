/*
 * Copyright (c) 2020 Genetec corporation
 * -*- coding:utf-8 -*-
 *
 * statemachine function about state machine.
 *
 */
package statemachine

import (
	"fmt"

	"github.com/marrbor/golog"
)

// Machine Type
type MachineTypeEnum struct{ typeCode int }

var MachineTypes = struct {
	MealyMachine MachineTypeEnum
	MooreMachine MachineTypeEnum
}{
	MealyMachine: MachineTypeEnum{MealyMachine},
	MooreMachine: MachineTypeEnum{MooreMachine},
}

const (
	// pseudo state have to be defined less than zero.
	NoNewState  = -1   // no need (self) transit, keep current state to next event. Valid only moore machine.
	FinishState = -100 // no more transit needed, finish to run machine.

	// predefined events. have to use under 0 value.
	Exit  = -1
	Start = -2
	Stop  = -3

	// callback message
	Stopped = "stopped"

	// typeCode
	MealyMachine = iota
	MooreMachine
)

// IsPseudoState returns whether given state number is pseudo state or not.
func IsPseudoState(state int) bool {
	return state < 0
}

var (
	FinishToTransitError = fmt.Errorf("finish to transit")
	MooreResponseError   = fmt.Errorf("invalid pseudo state has been returned")
)

type (
	Action func(sm *StateMachine) (result int, err error)

	StateMachine struct {
		state         int
		stt           *[][]*STTEntry
		machineType   MachineTypeEnum
		eventQueue    chan int    // event queue (read by this)
		callbackQueue chan string // message queue (write by this)
	}

	STTEntry struct {
		action Action
		next   []int // When moore machine table, length of next is always 1.
	}
)

// Start runs state machine
func (sm *StateMachine) Start() {
	go sm.run()
	sm.sendEvent(Start)
}

// Stop stop running state machine
func (sm *StateMachine) Stop() {
	golog.Debug("call Stop() !!!")
	sm.sendEvent(Stop)
}

// sendEvent send event to state machine
func (sm *StateMachine) sendEvent(event int) {
	sm.eventQueue <- event
}

// receiveEvent read from event queue. Blocked while receiving
func (sm *StateMachine) receiveEvent() int {
	ev := <-sm.eventQueue
	return ev
}

// sendMessage send callback message to caller.
func (sm *StateMachine) sendMessage(msg string) {
	sm.callbackQueue <- msg
}

// run wait for receive event and call transit when received.
func (sm *StateMachine) run() {
	for {
		ev := sm.receiveEvent()
		if ev == Exit {
			break
		}
		err := sm.transit(ev)
		if err == FinishToTransitError {
			break
		}
	}
	golog.Debug("exit state machine running.")
	sm.sendMessage(Stopped)
}

//
func (sm *StateMachine) transit(event int) error {
	tbl := *sm.stt
	entry := tbl[sm.state][event]
	for {
		if entry == nil {
			return nil // this event ignored at this state.
		}
		res, err := entry.action(sm)
		if err != nil {
			return err
		}
		if res == FinishState {
			return FinishToTransitError // transition finished.
		}
	}

	if res <= len(tbl[i].next) {
		return fmt.Errorf("action returns OB of next states. state: %d event: %d, number of next: %d", sm.state, event, len(tbl[i].next))
	}
	sm.state = tbl[i].next[res]
	golog.Debug(fmt.Sprintf("transit to state %d", sm.state))
	break
}
}
return nil
}

//
func (sm *StateMachine) mooreTransit(event int) error {
	tbl := *sm.mooreTable
	entry := tbl[sm.state][event]

	// seek transition
	for i := range tbl {
		if (tbl[i].state == sm.state || tbl[i].state == AllState) && tbl[i].event == event {

			for {
				sm.state = tbl[i].next
				next, err := tbl[i].action(sm)
				if err != nil {
					return err
				}
				if next == NoNewState {
					return nil // wait next event.
				}
				if next == FinishState {
					return FinishToTransitError // transition finished.
				}
				if IsPseudoState(next) {
					// other pseudo state should not be given action response value.
					return MooreResponseError
				}

				next, err :=
				if err := td.actFunction(sm); err != nil {
					golog.Error(fmt.Sprintf("errror currentstate:%d event: %d detected:%+v\n", sm.state, event, err))
					break
				}

				golog.Debug(fmt.Sprintf("transit to state %d", sm.state))
				break
			}
		}
	}

}

// transit check transition table and transit state if needed.
func (sm *StateMachine) transit(event int) {
	for i := range sm.stt {
		td := sm.stt[i]
		if (td.currentState == sm.state || td.currentState == AllState) && td.receiveEvent == event {
			if err := td.actFunction(sm); err != nil {
				golog.Error(fmt.Sprintf("errror currentstate:%d event: %d detected:%+v\n", sm.state, event, err))
				break
			}
			sm.state = td.nextState
			golog.Debug(fmt.Sprintf("transit to state %d", sm.state))
			break
		}
	}
}

// NewStateMachine returns given type state machine instance.
func NewStateMachine(mType MachineTypeEnum, table *[][]STTEntry) (*StateMachine, chan string) {
	ch := make(chan string)
	return &StateMachine{
		state:         0,
		stt:           table,
		machineType:   mType,
		eventQueue:    make(chan int, 1),
		callbackQueue: ch,
	}, ch
}

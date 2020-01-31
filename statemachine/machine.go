/*
 * Copyright (c) 2020 Genetec corporation
 * -*- coding:utf-8 -*-
 *
 * state machine function about state machine.
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
	FinishToRun = -100 // no more transit needed, finish to run machine.

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

var (
	FinishToTransitError = fmt.Errorf("finish to transit")
	MooreResponseError   = fmt.Errorf("invalid pseudo state has been returned")
)

type (
	Action func(sm *StateMachine) (result int, err error)

	StateMachine struct {
		state         int
		stt           *[][]*STTEntry
		actionTable   *[]Action
		machineType   MachineTypeEnum
		eventQueue    chan int    // event queue (read by this)
		callbackQueue chan string // message queue (write by this)
	}

	STTEntry struct {
		next   []int // When moore machine table, length of next is always 1.
		action int   // actionID to run.
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

// isMoore returns whether this machine is moore machine or not.
func (sm *StateMachine) isMoore() bool {
	return sm.machineType == MachineTypes.MooreMachine
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

		var err error
		if sm.isMoore() {
			err = sm.mooreTransit(ev)
		} else {
			err = sm.mealyTransit(ev)
		}
		if err == FinishToTransitError {
			break
		}
		golog.Error(err)
	}
	golog.Debug("exit state machine")
	sm.sendMessage(Stopped)
}

// mealy machine transit to next state and wait next event.
func (sm *StateMachine) mealyTransit(event int) error {
	stt := *sm.stt
	entry := stt[sm.state][event]
	if entry == nil {
		return nil // this event ignored at this state.
	}

	at := *sm.actionTable
	res, err := at[entry.action](sm)
	if err != nil {
		return err
	}
	if res == FinishToRun {
		return FinishToTransitError // transition finished.
	}
	if res <= len(entry.next) {
		return fmt.Errorf("action returns out of bounds for next states. state: %d event: %d, action returns : %d", sm.state, event, res)
	}
	sm.state = entry.next[res]
	return nil
}

//
func (sm *StateMachine) mooreTransit(event int) error {
	stt := *sm.stt
	entry := stt[sm.state][event]
	if entry == nil {
		return nil // this event ignored at this state.
	}
	if len(entry.next) != 1 {
		return fmt.Errorf("illegal entry detect: state:%d event: %d", sm.state, event)
	}

	at := *sm.actionTable
	next := entry.next[0]
	for {
		// moore machine performs state transition before call action and may make self-transition.
		sm.state = next

		res, err := at[sm.state](sm) // call action
		if err != nil {
			return err
		}
		if res == NoNewState {
			return nil // no self-transition.
		}
		if res == FinishToRun {
			return FinishToTransitError // transition finished.
		}
		next = res // do self-transition.
	}
}

// NewStateMachine returns given type state machine instance.
func NewStateMachine(mType MachineTypeEnum, stt *[][]*STTEntry, at *[]Action) (*StateMachine, chan string) {
	ch := make(chan string)
	return &StateMachine{
		state:         0,
		stt:           stt,
		actionTable:   at,
		machineType:   mType,
		eventQueue:    make(chan int, 1),
		callbackQueue: ch,
	}, ch
}

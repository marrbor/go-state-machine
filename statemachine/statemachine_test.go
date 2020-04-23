package statemachine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/marrbor/go-state-machine/statemachine"
	"github.com/marrbor/golog"
	"github.com/stretchr/testify/assert"
)

type Z struct{}

var z Z

var mq = make(chan string)

func TestNewStateMachineNG(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "notExist.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "open notExist.puml: no such file or directory")
}

func TestNewStateMachineNG2(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test2.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.NoEffectiveTransitionError.Error())
}

func TestNewStateMachineNG3(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test3.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.TriggerWithInitialTransitionError.Error())
}

func TestNewStateMachineNG4(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test4.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.GuardWithInitialTransitionError.Error())
}

func TestNewStateMachineNG5(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test5.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.ActionWithInitialTransitionError.Error())
}

func TestNewStateMachineNG6(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test6.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.MultipleInitialTransitionError.Error())
}

type X struct{}

func TestNewStateMachineNG7(t *testing.T) {
	var x X
	sm, err := statemachine.NewStateMachine(&x, "test7.puml", 1, mq, nil)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "following function(s) haven't be implemented: MaxCheck")
}

type qq struct {
	counter int
	retry   int
}

var q qq

func (q *qq) SaveResult() time.Duration {
	golog.Info("Do SaveResult")
	return statemachine.NoRetry
}

func (q *qq) Retry() time.Duration {
	q.retry += 1
	if 3 <= q.retry {
		return statemachine.NoRetry
	}
	return statemachine.GradualIncrease
}

func (q *qq) Retry2() time.Duration {
	return 1 * time.Second
}

func (q *qq) MaxCheck() bool {
	golog.Info(fmt.Sprintf("count should be %d -> %d.\n", q.counter, q.counter+1))
	q.counter += 1
	return 3 <= q.counter
}

func TestStateMachine(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name)
	sm.Send(statemachine.NewEvent("no effect")) // this event should be ignored

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name)
	sm.Send(statemachine.NewEvent("Succeeded"))

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState().Name)

	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState().Name)

	sm.Send(statemachine.NewEvent("Failed"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState().Name)

	sm.Send(statemachine.NewEvent("Succeeded"))

	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine2(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // initial transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine3(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))

	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine4(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))           // count 1
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))           // count 2
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))           // count 3 should be abort.

	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine5(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState().Name) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine6(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test9.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	sm.Send(statemachine.NewEvent("Try"))

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(statemachine.NewEvent("Aborted"))
	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine7(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq, nil)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	for i := 0; i < 5; i++ {
		sm.Send(statemachine.NewEvent("non registered event"))
		golog.Info(i)
	}

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(statemachine.NewEvent("Aborted"))
	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

type A struct {
	sm *statemachine.StateMachine
	dq chan string
}

func (a *A) EnterState1() {
	golog.Info("EnterState1")
}
func (a *A) DoState1() {
	golog.Info("DoState1")
	msg := <-a.dq
	response := "State1"
	if msg != response {
		response = fmt.Sprintf("invaild state:%s", msg)
	}
	a.sm.FinishDoAction(response)
}
func (a *A) ExitState1() {
	golog.Info("ExitState1")
}
func (a *A) EnterState2() {
	golog.Info("EnterState2")
}
func (a *A) DoState2() {
	golog.Info("DoState2")
	msg := <-a.dq
	response := "State2"
	if msg != response {
		response = fmt.Sprintf("invaild state:%s", msg)
	}
	a.sm.FinishDoAction(response)
}

func (a *A) ExitState2() {
	golog.Info("ExitState2")
}

var a A

func TestStateMachine8(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	a.dq = make(chan string)
	sm, err := statemachine.NewStateMachine(&a, "test10.puml", 1, mq, a.dq)
	a.sm = sm
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState().Name) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(5 * time.Second)

	// goto end status
	sm.Send(statemachine.NewEvent("Succeeded"))
	s := <-mq // Succeeded transit to this state.
	assert.True(t, statemachine.EndState.IsSame(sm.GetState()))
	assert.EqualValues(t, statemachine.Stopped, s)
}

func TestStateMachine9(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test11.puml", 1, mq, nil)
	assert.EqualError(t, err, "following function(s) haven't be implemented: EnterState3,DoState3,ExitState3")
}

func TestNewStateMachine(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test7.puml", 1, mq, nil)
	assert.NoError(t, err)
}

func TestNewStateMachine2(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test8.puml", 1, mq, nil)
	assert.EqualError(t, err, "following function(s) haven't be implemented: RetryX,SaveResultX,MaxCheckX")
}

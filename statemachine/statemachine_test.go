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
	sm, err := statemachine.NewStateMachine(&z, "notExist.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "open notExist.puml: no such file or directory")
}

func TestNewStateMachineNG2(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test2.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.NoEffectiveTransitionError.Error())
}

func TestNewStateMachineNG3(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test3.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.TriggerWithInitialTransitionError.Error())
}

func TestNewStateMachineNG4(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test4.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.GuardWithInitialTransitionError.Error())
}

func TestNewStateMachineNG5(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test5.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.ActionWithInitialTransitionError.Error())
}

func TestNewStateMachineNG6(t *testing.T) {
	sm, err := statemachine.NewStateMachine(&z, "test6.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, statemachine.MultipleInitialTransitionError.Error())
}

type X struct{}

func TestNewStateMachineNG7(t *testing.T) {
	var x X
	sm, err := statemachine.NewStateMachine(&x, "test7.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "following function(s) haven't be implemented: MaxCheck")
}

type qq struct {
	counter int
	retry   int
}

var q qq

func (q *qq) SaveResult() {
	golog.Info("Do SaveResult")
}

func (q *qq) Retry() {
	golog.Info("Retry")
}

func (q *qq) Retry2() {
	golog.Info("Retry2")
}

func (q *qq) MaxCheck() bool {
	golog.Info(fmt.Sprintf("count should be %d -> %d.\n", q.counter, q.counter+1))
	q.counter += 1
	return 3 <= q.counter
}

func TestStateMachine(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState())
	sm.Send(statemachine.NewEvent("no effect")) // this event should be ignored

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState())
	sm.Send(statemachine.NewEvent("Succeeded"))

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState())

	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState())

	sm.Send(statemachine.NewEvent("Failed"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState())

	sm.Send(statemachine.NewEvent("Succeeded"))

	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine2(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // initial transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine3(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))

	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine4(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))      // count 1
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))      // count 2
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))      // count 3 should be abort.

	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine5(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(statemachine.NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine6(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test9.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(statemachine.NewEvent("Try"))

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(statemachine.NewEvent("Aborted"))
	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine7(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	for i := 0; i < 5; i++ {
		sm.Send(statemachine.NewEvent("non registered event"))
		golog.Info(i)
	}

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(statemachine.NewEvent("Aborted"))
	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

type A struct {
	sm    *statemachine.StateMachine
	count int
}

func (a *A) EnterState1() {
	golog.Info("EnterState1")
	a.count = 0
}
func (a *A) DoState1() time.Duration {
	for {
		golog.Info(fmt.Sprintf("DoState1 %d", a.count))
		a.count++
		if 3 <= a.count {
			break
		}
		return 1 * time.Second
	}
	return 0
}

func (a *A) ExitState1() {
	golog.Info("ExitState1")
}

func (a *A) EnterState2() {
	golog.Info("EnterState2")
}

func (a *A) DoState2() time.Duration {
	golog.Info("DoState2")
	return 0
}

func (a *A) ExitState2() {
	golog.Info("ExitState2")
}

var a A

func TestStateMachine8(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	sm, err := statemachine.NewStateMachine(&a, "test10.puml", 1, mq)
	a.sm = sm
	assert.NoError(t, err)
	sm.Run()

	time.Sleep(10 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(statemachine.NewEvent("Succeeded"))

	time.Sleep(10 * time.Second)
	assert.EqualValues(t, "State2", sm.GetState()) // state should not be changed.

	// goto end status
	sm.Send(statemachine.NewEvent("Succeeded"))
	<-mq // Succeeded transit to this state.
	assert.True(t, sm.IsEnd())
}

func TestStateMachine9(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test11.puml", 1, mq)
	assert.EqualError(t, err, "following function(s) haven't be implemented: EnterState3,DoState3,ExitState3")
}

func TestNewStateMachine(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test7.puml", 1, mq)
	assert.NoError(t, err)
}

func TestNewStateMachine2(t *testing.T) {
	err := golog.SetFilterLevel(golog.TRACE)
	assert.NoError(t, err)
	_, err = statemachine.NewStateMachine(&q, "test8.puml", 1, mq)
	assert.EqualError(t, err, "following function(s) haven't be implemented: RetryX,SaveResultX,MaxCheckX")
}

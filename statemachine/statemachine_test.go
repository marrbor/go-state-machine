package statemachine

import (
	"fmt"
	"testing"
	"time"

	"github.com/marrbor/golog"
	"github.com/stretchr/testify/assert"
)

func TestRetryDuration(t *testing.T) {
	e := NewEvent("test")
	assert.False(t, e.calcRetryDuration(NoRetry))

	assert.True(t, e.calcRetryDuration(GradualIncrease))
	assert.EqualValues(t, 1, e.attempt)
	assert.EqualValues(t, DurationOfFirstRetry, e.delay)

	assert.True(t, e.calcRetryDuration(GradualIncrease))
	assert.EqualValues(t, 2, e.attempt)
	assert.EqualValues(t, DurationOfFirstRetry*2, e.delay)

	assert.True(t, e.calcRetryDuration(GradualIncrease))
	assert.EqualValues(t, 3, e.attempt)
	assert.EqualValues(t, DurationOfFirstRetry*2*2, e.delay) //

	d := 1 * time.Second
	assert.True(t, e.calcRetryDuration(int64(d)))
	assert.EqualValues(t, 4, e.attempt)
	assert.EqualValues(t, d, e.delay)

}

func TestParser1(t *testing.T) {
	tr := parseTransition("[*] --> State1")
	assert.EqualValues(t, &transition{
		from:    StartStateMark,
		to:      "State1",
		trigger: "",
		guard:   "",
		action:  "",
	}, tr)
	assert.EqualValues(t, StartStateMark, tr.from)
	assert.EqualValues(t, "State1", tr.to)
	assert.EqualValues(t, "", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "", tr.action)
}

func TestParser2(t *testing.T) {
	tr := parseTransition("State1 --> State2 : Succeeded")
	assert.EqualValues(t, "State1", tr.from)
	assert.EqualValues(t, "State2", tr.to)
	assert.EqualValues(t, "Succeeded", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "", tr.action)
}

func TestParser3(t *testing.T) {
	tr := parseTransition("State1 --> [*]: Aborted")
	assert.EqualValues(t, "State1", tr.from)
	assert.EqualValues(t, EndStateMark, tr.to)
	assert.EqualValues(t, "Aborted", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "", tr.action)
}

func TestParser4(t *testing.T) {
	tr := parseTransition("State2 --> State3: Succeeded/Act1")
	assert.EqualValues(t, "State2", tr.from)
	assert.EqualValues(t, "State3", tr.to)
	assert.EqualValues(t, "Succeeded", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "Act1", tr.action)
}

func TestParser5(t *testing.T) {
	tr := parseTransition("State2 --> [*] :Aborted [checkAbort]  / Act2")
	assert.EqualValues(t, "State2", tr.from)
	assert.EqualValues(t, EndStateMark, tr.to)
	assert.EqualValues(t, "Aborted", tr.trigger)
	assert.EqualValues(t, "checkAbort", tr.guard)
	assert.EqualValues(t, "Act2", tr.action)
}
func TestParser6(t *testing.T) {
	tr := parseTransition("State3 --> State3 : TimeOut")
	assert.EqualValues(t, "State3", tr.from)
	assert.EqualValues(t, "State3", tr.to)
	assert.EqualValues(t, "TimeOut", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "", tr.action)
}

func TestParser7(t *testing.T) {
	tr := parseTransition("State3 --> [*]: Succeeded / SaveResult")
	assert.EqualValues(t, "State3", tr.from)
	assert.EqualValues(t, EndStateMark, tr.to)
	assert.EqualValues(t, "Succeeded", tr.trigger)
	assert.EqualValues(t, "", tr.guard)
	assert.EqualValues(t, "SaveResult", tr.action)
}
func TestParser8(t *testing.T) {
	tr := parseTransition("@startuml")
	assert.Nil(t, tr)
}

func TestParser9(t *testing.T) {
	tr := parseTransition("@enduml")
	assert.Nil(t, tr)
}

type Z struct{}

var z Z

var mq = make(chan string)

func TestNewStateMachineNG(t *testing.T) {
	sm, err := NewStateMachine(&z, "notExist.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, "open notExist.puml: no such file or directory")
}

func TestNewStateMachine1(t *testing.T) {
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)
	assert.NotNil(t, sm)

	assert.EqualValues(t, 4, len(sm.states))
	assert.EqualValues(t, "State1", sm.GetState())

	for _, s := range sm.states {
		switch s.name {
		case "State1":
			assert.EqualValues(t, 2, len(s.transitions))
		case "State2":
			assert.EqualValues(t, 2, len(s.transitions))
		case "State3":
			assert.EqualValues(t, 3, len(s.transitions))
		case EndState.name:
			assert.EqualValues(t, 0, len(s.transitions))
		default:
			t.Errorf("invalid state data detect: %+v", s)
		}
	}
}

func TestNewStateMachineNG2(t *testing.T) {
	sm, err := NewStateMachine(&z, "test2.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, NoEffectiveTransitionError.Error())
}

func TestNewStateMachineNG3(t *testing.T) {
	sm, err := NewStateMachine(&z, "test3.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, TriggerWithInitialTransitionError.Error())
}

func TestNewStateMachineNG4(t *testing.T) {
	sm, err := NewStateMachine(&z, "test4.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, GuardWithInitialTransitionError.Error())
}

func TestNewStateMachineNG5(t *testing.T) {
	sm, err := NewStateMachine(&z, "test5.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, ActionWithInitialTransitionError.Error())
}

func TestNewStateMachineNG6(t *testing.T) {
	sm, err := NewStateMachine(&z, "test6.puml", 1, mq)
	assert.Nil(t, sm)
	assert.EqualError(t, err, MultipleInitialTransitionError.Error())
}

type X struct{}

func TestNewStateMachineNG7(t *testing.T) {
	var x X
	sm, err := NewStateMachine(&x, "test7.puml", 1, mq)
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
	return NoRetry
}

func (q *qq) Retry() time.Duration {
	q.retry += 1
	if 3 <= q.retry {
		return NoRetry
	}
	return GradualIncrease
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
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	sn := sm.GetState()
	assert.EqualValues(t, "State1", sn)
	sm.Send(NewEvent("no effect")) // this event should be ignored

	time.Sleep(1 * time.Second)

	sn = sm.GetState()
	assert.EqualValues(t, "State1", sn)
	sm.Send(NewEvent("Succeeded"))

	time.Sleep(1 * time.Second)

	sn = sm.GetState()
	assert.EqualValues(t, "State2", sn)

	sm.Send(NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	sn = sm.GetState()
	assert.EqualValues(t, "State3", sn)

	sm.Send(NewEvent("Failed"))
	time.Sleep(1 * time.Second)

	sn = sm.GetState()
	assert.EqualValues(t, "State3", sn)

	sm.Send(NewEvent("Succeeded"))

	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine2(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // initial transit to this state.
	sm.Send(NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine3(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Aborted"))

	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine4(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Aborted"))                   // count 1
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Aborted"))                   // count 2
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State3", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Aborted"))                   // count 3 should be abort.

	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine5(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(NewEvent("Succeeded"))
	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State2", sm.GetState()) // Succeed transit to this state.
	sm.Send(NewEvent("Aborted"))
	time.Sleep(1 * time.Second)

	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine6(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test9.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(NewEvent("Try"))

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(NewEvent("Aborted"))
	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine7(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&q, "test1.puml", 1, mq)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	for i := 0; i < 5; i++ {
		sm.Send(NewEvent("non registered event"))
		golog.Info(i)
	}

	time.Sleep(5 * time.Second)

	// goto end status with stop timer.
	sm.Send(NewEvent("Aborted"))
	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

type A struct {
	sm *StateMachine
}

func (a *A) EnterState1() {
	golog.Info("EnterState1")
}
func (a *A) DoState1() {
	golog.Info("DoState1")
	msg := <-a.sm.toDoActionQueue
	response := "State1"
	if msg.Error() != response {
		response = fmt.Sprintf("invaild state:%s", msg)
	}
	a.sm.fromDoActionQueue <- fmt.Errorf(response)
}
func (a *A) ExitState1() {
	golog.Info("ExitState1")
}
func (a *A) EnterState2() {
	golog.Info("EnterState2")
}
func (a *A) DoState2() {
	golog.Info("DoState2")
	msg := <-a.sm.toDoActionQueue
	response := "State2"
	if msg.Error() != response {
		response = fmt.Sprintf("invaild state:%s", msg)
	}
	a.sm.fromDoActionQueue <- fmt.Errorf(response)
}
func (a *A) ExitState2() {
	golog.Info("ExitState2")
}

var a A

func TestStateMachine8(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	sm, err := NewStateMachine(&a, "test10.puml", 1, mq)
	a.sm = sm
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.EqualValues(t, "State1", sm.GetState()) // state should not be changed.
	sm.Send(NewEvent("Succeeded"))
	time.Sleep(5 * time.Second)

	// goto end status
	sm.Send(NewEvent("Succeeded"))
	s := <-sm.msgQueue // Succeeded transit to this state.
	assert.EqualValues(t, EndState.name, sm.GetState())
	assert.EqualValues(t, Stopped, s)
}

func TestStateMachine9(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	_, err := NewStateMachine(&q, "test11.puml", 1, mq)
	assert.EqualError(t, err, "following function(s) haven't be implemented: EnterState3,DoState3,ExitState3")
}

func TestNewStateMachine(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	_, err := NewStateMachine(&q, "test7.puml", 1, mq)
	assert.NoError(t, err)
}

func TestNewStateMachine2(t *testing.T) {
	golog.SetFilterLevel(golog.TRACE)
	_, err := NewStateMachine(&q, "test8.puml", 1, mq)
	assert.EqualError(t, err, "following function(s) haven't be implemented: RetryX,SaveResultX,MaxCheckX")
}

func TestIsTrigger(t *testing.T) {
	assert.EqualValues(t, true, isTiming("entry"))
	assert.EqualValues(t, true, isTiming("do"))
	assert.EqualValues(t, true, isTiming("exit"))

	assert.EqualValues(t, false, isTiming("Entry"))
	assert.EqualValues(t, false, isTiming("Do"))
	assert.EqualValues(t, false, isTiming("Exit"))
}

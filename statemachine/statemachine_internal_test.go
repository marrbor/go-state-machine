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

func TestNewStateMachine(t *testing.T) {
	mq := make(chan string)
	dq := make(chan string)

	sm, err := NewStateMachine(&q, "test1.puml", 1, mq, dq)
	assert.NoError(t, err)
	assert.NotNil(t, sm)

	assert.EqualValues(t, 4, len(sm.states))
	assert.EqualValues(t, "State1", sm.GetState().Name)

	for _, s := range sm.states {
		switch s.Name {
		case "State1":
			assert.EqualValues(t, 2, len(s.transitions))
		case "State2":
			assert.EqualValues(t, 2, len(s.transitions))
		case "State3":
			assert.EqualValues(t, 3, len(s.transitions))
		case EndState.Name:
			assert.EqualValues(t, 0, len(s.transitions))
		default:
			t.Errorf("invalid state data detect: %+v", s)
		}
	}
}

func TestIsTrigger(t *testing.T) {
	assert.EqualValues(t, true, isTiming("entry"))
	assert.EqualValues(t, true, isTiming("do"))
	assert.EqualValues(t, true, isTiming("exit"))

	assert.EqualValues(t, false, isTiming("Entry"))
	assert.EqualValues(t, false, isTiming("Do"))
	assert.EqualValues(t, false, isTiming("Exit"))
}

package statemachine

import (
	"fmt"
	"testing"

	"github.com/marrbor/golog"
	"github.com/stretchr/testify/assert"
)

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

func TestNewStateMachine(t *testing.T) {
	mq := make(chan string)

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
		case endState.name:
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

package statemachine

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marrbor/golog"
)

const (
	// callback message
	Stopped = "stopped"

	// uml mark
	StartStateMark = "[*]"
	EndStateMark   = "[*]"

	// timing return value
	NoRetry         = 0
	GradualIncrease = -1

	// Default retry interval
	DurationOfFirstRetry = 500 * time.Microsecond
)

var (
	FinishToTransitError = fmt.Errorf("finish to transit")

	// UML error
	MultipleInitialTransitionError    = fmt.Errorf("multiple initial transition defined")
	TriggerWithInitialTransitionError = fmt.Errorf("initial transition has not have timing")
	GuardWithInitialTransitionError   = fmt.Errorf("initial transition has not have guard condition")
	ActionWithInitialTransitionError  = fmt.Errorf("initial transition has not have timing")
	NoEffectiveTransitionError        = fmt.Errorf("no effective transition found")

	reWhiteSpace = regexp.MustCompile(`\s+`)
)

//////////////////////////////////////////////////
// function type
type (
	Action func() time.Duration
	Guard  func() bool
)

//////////////////////////////////////////////////
// Entry unit of state transition table.
type STTItem struct {
	guard  string // guard function name which called to determine whether do transition or not.
	action string // timing function name to run with this transition
	next   *State // next state after timing runs.
}

//////////////////////////////////////////////////
// State Transition table
type stt map[string][]STTItem

func (s stt) add(ev *Event, item STTItem) {
	s[ev.name] = append(s[ev.name], item)
}

//////////////////////////////////////////////////
// State
type State struct {
	name        string
	transitions stt
	entry       string
	exit        string
	do          string
}

// Pre defined states
var (
	EndState = State{name: "END"}
)

// isSame returns whether given state is same state or not.
func (s *State) isSame(ss *State) bool {
	return s.name == ss.name
}

// addTransitionItem adds STTItem into stt.
func (s *State) addTransitionItem(trigger *Event, item STTItem) {
	s.transitions.add(trigger, item)
}

// NewState returns new state instance.
func NewState(name string) *State {
	if name == EndStateMark {
		return &EndState
	}
	return &State{
		name:        name,
		transitions: make(stt),
	}
}

//////////////////////////////////////////////////
// Event
type Event struct {
	name    string
	id      uuid.UUID
	attempt int
	delay   time.Duration
}

// NewEvent returns new Event instance
func NewEvent(name string) *Event {
	return &Event{
		name:    name,
		id:      uuid.New(),
		attempt: 0,
		delay:   0,
	}
}

// calcRetryDuration returns whether has to re-send this event or not.
// When need to retry, set wait duration for re-send and increment attempt times.
// Input parameter is return value of timing function.
func (e *Event) calcRetryDuration(ret int64) bool {
	if ret == NoRetry {
		return false // no need to retry
	}

	// need to retry
	e.attempt += 1

	// delay time is given.
	if 0 < ret {
		e.delay = time.Duration(ret)
		return true
	}

	// GradualIncrease
	if e.attempt <= 1 {
		// first time GradualIncrease.
		e.delay = DurationOfFirstRetry
		return true
	}
	e.delay *= 2 // 2+ time GradualIncrease.
	return true
}

//////////////////////////////////////////////////
// StateMachine
type StateMachine struct {
	bindClass         interface{}
	currentState      *State
	states            []*State
	eventQueue        chan *Event // event queue to pull event.
	msgQueue          chan string // message queue to Send message to external program.
	retryTimer        *time.Timer
	toDoActionQueue   chan error // send message to 'do action' goroutine.
	fromDoActionQueue chan error // sent message from 'do action' goroutine.
}

// Send sends event to state machine.
func (sm *StateMachine) Send(ev *Event) {
	sm.eventQueue <- ev
}

// GetState returns current state name
func (sm *StateMachine) GetState() string {
	return sm.currentState.name
}

// run wait for pull event and call transit when received.
func (sm *StateMachine) run() {
	for {
		ev := <-sm.eventQueue
		golog.Debug(fmt.Sprintf("state: %s,event: '%+v'", sm.currentState.name, ev.name))
		golog.Debug(fmt.Sprintf("event id:%+v attempt:%d delay:%d", ev.id, ev.attempt, ev.delay))
		if err := sm.transit(ev); err != nil {
			if err == FinishToTransitError {
				break
			}
			golog.Error(err)
		}
	}
	golog.Info("exit state machine")
	sm.msgQueue <- Stopped
}

// finish cleanup state machine.
func (sm *StateMachine) finish() {
	// stop retry timer.
	if sm.retryTimer != nil {
		if !sm.retryTimer.Stop() {
			<-sm.retryTimer.C // drain timer channel.
		}
	}
	golog.Info(fmt.Sprintf("transit %s -> %s", sm.currentState.name, EndState.name))
	sm.currentState = &EndState
}

// preTransit calls exit functions if needed.
func (sm *StateMachine) preTransit() {
	do := sm.currentState.do
	if 0 < len(do) {
		golog.Trace("Stop do action.")
		sm.toDoActionQueue <- fmt.Errorf(sm.currentState.name)
		sn := <-sm.fromDoActionQueue
		if sn.Error() != sm.currentState.name {
			golog.Error(fmt.Sprintf("unexpected message from (%s) detect", sn.Error()))
		}
	}

	exit := sm.currentState.exit
	if 0 < len(exit) {
		reflect.ValueOf(sm.bindClass).MethodByName(exit).Call([]reflect.Value{})
	}
}

// postTransit calls enter functions if needed.
func (sm *StateMachine) postTransit() {
	entry := sm.currentState.entry
	if 0 < len(entry) {
		reflect.ValueOf(sm.bindClass).MethodByName(entry).Call([]reflect.Value{})
	}
	do := sm.currentState.do
	if 0 < len(do) {
		go reflect.ValueOf(sm.bindClass).MethodByName(do).Call([]reflect.Value{})
	}
}

// transit make state transition in order to stt.
func (sm *StateMachine) transit(ev *Event) error {
	stt := sm.currentState.transitions[ev.name]
	if stt == nil {
		golog.Debug(fmt.Sprintf("event %s ignored at state %s", ev.name, sm.currentState.name))
		return nil
	}

	// get effective transition
	var item *STTItem = nil
	for i := range stt {
		item = &stt[i]
		if len(item.guard) <= 0 {
			golog.Trace("no guard condition defined. do transition")
			break
		}
		if !reflect.ValueOf(sm.bindClass).MethodByName(item.guard).Call([]reflect.Value{})[0].Bool() {
			golog.Trace(fmt.Sprintf("guard condition %s not match. seek next candidate", item.guard))
			item = nil
			continue
		}
		golog.Trace(fmt.Sprintf("guard condition %s match. do transition", item.guard))
		break // found.
	}

	if item == nil {
		golog.Trace(fmt.Sprintf("event %s ignored at state %s since guard condition not match.", ev.name, sm.currentState.name))
		return nil
	}

	// do timing if defined
	if 0 < len(item.action) {
		golog.Trace(fmt.Sprintf("do timing: '%+v'", item.action))
		ret := reflect.ValueOf(sm.bindClass).MethodByName(item.action).Call([]reflect.Value{})[0].Int()
		// retry and no transition when specified.
		golog.Trace(fmt.Sprintf("timing: '%+v' return %+v", item.action, ret))
		if ev.calcRetryDuration(ret) {
			golog.Trace(fmt.Sprintf("wait %+v for retry.", ev.delay))
			sm.retryTimer = time.AfterFunc(ev.delay, func() {
				sm.retryTimer = nil
				sm.Send(ev)
			})
			return nil
		}
	}

	// state transition. run exit action if specified.
	sm.preTransit()
	golog.Info(fmt.Sprintf("%s -> %s", sm.currentState.name, item.next.name))

	if item.next.isSame(&EndState) {
		// finish.
		sm.finish()
		return FinishToTransitError // reach to end state, shutdown machine.
	}

	// entering new state. run entry and/or do action if specified.
	sm.currentState = item.next
	sm.postTransit()
	return nil
}

// transition data import from plant uml line used at parsing.
type transition struct {
	from    string
	to      string
	trigger string
	guard   string
	action  string
}

// timing data import from plant uml line used at parsing.
type timing struct {
	state    string
	timing   string
	function string
}

// isTiming returns whether given string is timing or not.
func isTiming(s string) bool {
	return s == "entry" || s == "do" || s == "exit"
}

// getState add state that given name. if it has not generated yet, generate it.
func getState(states map[string]*State, state string) *State {
	s, ok := states[state]
	if !ok {
		s = NewState(state) // found new state
		states[s.name] = s
	}
	return s
}

// NewStateMachine returns state machine instance with state model transition generated from given uml file.
func NewStateMachine(k interface{}, path string, qSize int, mq chan string) (*StateMachine, error) {
	// array of function name that have to be implemented but not found.
	var missedFunctions []string

	// provision return value.
	sm := &StateMachine{
		bindClass:         k,
		currentState:      nil,
		states:            make([]*State, 0),
		eventQueue:        make(chan *Event, qSize),
		msgQueue:          mq,
		fromDoActionQueue: make(chan error),
		toDoActionQueue:   make(chan error),
	}

	// construct state transition data in order to given uml file.
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	stateMap := make(map[string]*State, 0)

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		tr := parseTransition(line)
		if tr == nil {
			tg := parseTiming(line)
			if tg == nil {
				continue // seek next entry.
			}

			// found timing function line.
			if !reflect.ValueOf(k).MethodByName(tg.function).IsValid() {
				missedFunctions = append(missedFunctions, tg.function)
			}

			s := getState(stateMap, tg.state)
			switch tg.timing {
			case "entry":
				s.entry = tg.function
			case "do":
				s.do = tg.function
			case "exit":
				s.exit = tg.function
			default:
				golog.Error(fmt.Sprintf("invalid timing (%s) detect", tg.timing))
			}
			continue // seek next entry.
		}

		// check timing and/or guard function exists or not.
		for _, f := range []string{tr.action, tr.guard} {
			if 0 < len(f) {
				if !reflect.ValueOf(k).MethodByName(f).IsValid() {
					missedFunctions = append(missedFunctions, f)
				}
			}
		}

		// initial transition
		if tr.from == StartStateMark {
			if sm.currentState != nil {
				return nil, MultipleInitialTransitionError
			}
			if tr.trigger != "" {
				return nil, TriggerWithInitialTransitionError
			}
			if tr.guard != "" {
				return nil, GuardWithInitialTransitionError
			}
			if tr.action != "" {
				return nil, ActionWithInitialTransitionError
			}
			if tr.to == EndStateMark {
				// initial transition never goto end state.
				return nil, NoEffectiveTransitionError
			}

			// set destination state into first state of this state machine
			sm.currentState = getState(stateMap, tr.to)
			continue
		}

		// other transition. generate both from/to states and register them if they have not registered yet.
		s := getState(stateMap, tr.from)
		n := getState(stateMap, tr.to)
		item := STTItem{guard: tr.guard, action: tr.action, next: n}
		s.addTransitionItem(NewEvent(tr.trigger), item)
	}

	// error exit when found non-implement function(s)
	missedFunction := strings.Join(missedFunctions, ",")
	if 0 < len(missedFunction) {
		return nil, fmt.Errorf("following function(s) haven't be implemented: %s", missedFunction)
	}

	for _, v := range stateMap {
		sm.states = append(sm.states, v)
	}

	// start state machine
	sm.postTransit() // run entry/do action if needed.
	go sm.run()
	return sm, nil
}

// parseTransition extract transition information from given string.
func parseTransition(s string) *transition {
	var ret transition
	ss := reWhiteSpace.ReplaceAllString(s, "")
	a := strings.Split(ss, "-->")
	if len(a) <= 1 {
		return nil // this is not a transition line.
	}

	// Get source state into from.
	ret.from = a[0]

	// search event
	b := strings.Split(a[1], ":")
	if len(b) <= 1 {
		// event not defined.
		ret.to = a[1]
		return &ret
	}

	// event defined.
	ret.to = b[0]
	c := strings.Split(b[1], "/")
	if 1 < len(c) {
		// timing defined.
		ret.action = c[1]
	}
	d := strings.Split(strings.TrimRight(c[0], "]"), "[")
	ret.trigger = d[0]
	if len(d) <= 1 {
		// guard not defined.
		return &ret
	}

	ret.guard = d[1]
	return &ret
}

// parseTiming extract transition information from given string.
func parseTiming(s string) *timing {
	var ret timing
	ss := reWhiteSpace.ReplaceAllString(s, "")
	if i := strings.Index(ss, "-->"); 0 <= i {
		return nil // this is not a timing line, maybe transition line.
	}

	a := strings.Split(ss, ":")
	if len(a) != 2 {
		return nil // this is not a timing line.
	}

	// Get state name that owned this timing.
	ret.state = a[0]

	// split to timing, function.
	b := strings.Split(a[1], "/")
	if len(b) != 2 || !isTiming(b[0]) {
		return nil // this is not a timing line.
	}

	ret.timing = b[0]
	ret.function = b[1]
	return &ret
}

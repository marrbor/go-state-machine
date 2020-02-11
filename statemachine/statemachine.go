package statemachine

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/marrbor/golog"
)

const (
	// callback message
	Stopped = "stopped"

	// uml mark
	StartStateMark = "[*]"
	EndStateMark   = "[*]"
)

var (
	FinishToTransitError = fmt.Errorf("finish to transit")

	// UML error
	MultipleInitialTransitionError    = fmt.Errorf("multiple initial transition defined")
	TriggerWithInitialTransitionError = fmt.Errorf("initial transition has not have trigger")
	GuardWithInitialTransitionError   = fmt.Errorf("initial transition has not have guard condition")
	ActionWithInitialTransitionError  = fmt.Errorf("initial transition has not have action")
	NoEffectiveTransitionError        = fmt.Errorf("no effective transition found")
	NoShutdownError                   = fmt.Errorf("'Shutdown' method is not implemented")

	reWhiteSpace = regexp.MustCompile(`\s+`)
)

//////////////////////////////////////////////////
// function type
type (
	Action func()
	Guard  func() bool
)

//////////////////////////////////////////////////
// Entry unit of state transition table.
type STTItem struct {
	guard  string // guard function name which called to determine whether do transition or not.
	action string // action function name to run with this transition
	next   *State // next state after action runs.
}

//////////////////////////////////////////////////
// State Transition table
type stt map[Event][]STTItem

func (s stt) add(ev Event, item STTItem) {
	s[ev] = append(s[ev], item)
}

//////////////////////////////////////////////////
// State
type State struct {
	name        string
	transitions stt
}

// Pre defined states
var (
	EndState = State{name: "-"}
)

// IsSame returns whether given state is same state or not.
func (s *State) IsSame(ss *State) bool {
	return s.name == ss.name
}

// addTransitionItem adds STTItem into stt.
func (s *State) addTransitionItem(trigger Event, item STTItem) {
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
type Event string

// IsSame returns whether given event is same event or not.
func (e *Event) IsSame(ev *Event) bool {
	return *e == *ev
}

// NewEvent returns new Event instance
func NewEvent(name string) Event {
	return Event(name)
}

// pre defined events
var (
	ShutdownEvent = Event("shutdown")
)

//////////////////////////////////////////////////
// eventQueue
type eventQueue chan Event

// push sends event to event queue.
func (eq eventQueue) push(event Event) {
	eq <- event
}

// pull receives event from event queue. Blocked until event sent.
func (eq eventQueue) pull() Event {
	return <-eq
}

// newEventQueue returns new event queue instance.
func newEventQueue() eventQueue {
	return make(eventQueue, 1)
}

//////////////////////////////////////////////////
// msgQueue
type msgQueue chan string

// push sends message to queue
func (mq msgQueue) push(msg string) {
	mq <- msg
}

// pull receives message from queue blocked until message sent.
func (mq msgQueue) pull() string {
	return <-mq
}

// newMsgQueue returns new message queue instance.
func newMsgQueue() msgQueue {
	return make(msgQueue)
}

//////////////////////////////////////////////////
// StateMachine
type StateMachine struct {
	bindClass    interface{}
	currentState *State
	states       []*State
	eventQueue   eventQueue // event queue to pull event.
	msgQueue     msgQueue   // message queue to Send message to external program.
}

// Listen message returns message from this state machine. block until message sent.
func (sm *StateMachine) Listen() string {
	return sm.msgQueue.pull()
}

// Start runs state machine
func (sm *StateMachine) Start() {
	go sm.run()
}

// Stop stop running state machine
func (sm *StateMachine) Stop() {
	golog.Debug("call Stop() !!!")
	sm.Send(ShutdownEvent)
}

// Send sends event to state machine
func (sm *StateMachine) Send(ev Event) {
	sm.eventQueue.push(ev)
}

// sendMessage Send callback message to caller.
func (sm *StateMachine) sendMessage(msg string) {
	sm.msgQueue.push(msg)
}

// run wait for pull event and call transit when received.
func (sm *StateMachine) run() {
	for {
		ev := sm.eventQueue.pull()
		golog.Trace(fmt.Sprintf("state: %s,event: '%+v'", sm.currentState.name, ev))
		if err := sm.transit(ev); err != nil {
			if err == FinishToTransitError {
				break
			}
			golog.Error(err)
		}
	}
	golog.Debug("exit state machine")
	sm.sendMessage(Stopped)
}

// transit make state transition in order to stt.
func (sm *StateMachine) transit(ev Event) error {

	// when shutdown event detect, call Shutdown method and return FinishToTransitError to notify state machine stopped.
	if ev.IsSame(&ShutdownEvent) {
		golog.Trace("detect shutdown.")
		reflect.ValueOf(sm.bindClass).MethodByName("Shutdown").Call([]reflect.Value{})
		sm.currentState = &EndState
		return FinishToTransitError
	}

	stt := sm.currentState.transitions[ev]
	if stt == nil {
		golog.Trace("this event ignored at this state.")
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
			golog.Trace("guard condition not match. seek next candidate")
			item = nil
			continue
		}
		golog.Trace("guard condition match. do transition")
		break // found.
	}

	if item == nil {
		golog.Trace("this event ignored at this state since guard condition not match.")
		return nil
	}

	// do action if defined
	if 0 < len(item.action) {
		golog.Trace(fmt.Sprintf("do action: '%+v'\n", item.action))
		reflect.ValueOf(sm.bindClass).MethodByName(item.action).Call([]reflect.Value{})
	}

	// state transition
	golog.Trace(fmt.Sprintf("%+v -> %+v", sm.currentState, item.next))
	sm.currentState = item.next
	if sm.currentState.IsSame(&EndState) {
		return FinishToTransitError // reach to end state, shutdown machine.
	}
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

// NewStateMachine returns state machine instance with state model transition generated from given uml file.
func NewStateMachine(k interface{}, path string) (*StateMachine, error) {
	sm := &StateMachine{
		bindClass:    k,
		currentState: nil,
		states:       make([]*State, 0),
		eventQueue:   newEventQueue(),
		msgQueue:     newMsgQueue(),
	}

	// check Shutdown Method implemented or not.
	v := reflect.ValueOf(k).MethodByName("Shutdown")
	if !v.IsValid() {
		return nil, NoShutdownError
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
		tr := parseLine(scanner.Text())
		if tr == nil {
			continue // seek next entry.
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

			// set destination state into first state of this state machine
			s, ok := stateMap[tr.to]
			if !ok {
				// found new state
				if tr.to == EndStateMark {
					// initial transition never goto end state.
					return nil, NoEffectiveTransitionError
				}
				s = NewState(tr.to)
				stateMap[s.name] = s
			}
			sm.currentState = s
			continue
		}

		// other transition. generate both from/to states and register them if they have not registered yet.
		s, ok := stateMap[tr.from]
		if !ok {
			// found new state
			s = NewState(tr.from)
			stateMap[s.name] = s
		}
		n, ok := stateMap[tr.to]
		if !ok {
			// found new state
			n = NewState(tr.to)
			stateMap[n.name] = n
		}
		item := STTItem{guard: tr.guard, action: tr.action, next: n}
		s.addTransitionItem(Event(tr.trigger), item)
	}

	for _, v := range stateMap {
		sm.states = append(sm.states, v)
	}
	return sm, nil
}

// parseLine extract transition information from given string.
func parseLine(s string) *transition {
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
		// action defined.
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

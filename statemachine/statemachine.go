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
	// message for goroutine
	changeState = "state changed"
	pleaseStop  = "stop!"
	stopped     = "stopped"

	// do-action tick timer duration when do-action is not running
	idleTickDuration = 1 * time.Hour

	// uml mark
	StartStateMark = "[*]"
	EndStateMark   = "[*]"
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
// Entry unit of state transition table.
type STTItem struct {
	guard  string // guard function name which called to determine whether do transition or not.
	action string // timing function name to run with this transition
	next   *State // next state after timing runs.
}

// hasGuard returns whether this item has guard condition or not.
func (i *STTItem) hasGuard() bool {
	return 0 < len(i.guard)
}

// hasAction returns whether this item has guard condition or not.
func (i *STTItem) hasAction() bool {
	return 0 < len(i.action)
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
	endState = State{name: "END"}
)

// hasEntry returns whether this state has entry action or not.
func (s *State) hasEntry() bool {
	return 0 < len(s.entry)
}

// hasDo returns whether this state has do action or not.
func (s *State) hasDo() bool {
	return 0 < len(s.do)
}

// hasExit returns whether this state has exit action or not.
func (s *State) hasExit() bool {
	return 0 < len(s.exit)
}

// isSame returns whether given state is same state or not.
func (s *State) isSame(ss *State) bool {
	return s.name == ss.name
}

// addTransitionItem adds STTItem into stt.
func (s *State) addTransitionItem(trigger *Event, item STTItem) {
	s.transitions.add(trigger, item)
}

// isEnd returns whether given state is end state or not.
func (s *State) isEnd() bool {
	return s.name == endState.name
}

// NewState returns new state instance.
func NewState(name string) *State {
	if name == EndStateMark {
		return &endState
	}
	return &State{
		name:        name,
		transitions: make(stt),
	}
}

//////////////////////////////////////////////////
// Event
type Event struct {
	name string
	id   uuid.UUID
}

// NewEvent returns new Event instance
func NewEvent(name string) *Event {
	return &Event{name: name, id: uuid.New()}
}

//////////////////////////////////////////////////
// StateMachine
type StateMachine struct {
	bindClass    interface{}
	currentState *State
	states       []*State

	eventQueue   chan *Event // event queue to pull event.
	toOwnerQueue chan string // message queue to send message to owner program.

	toDoActionQueue   chan string // send message to 'do-action' goroutine.
	fromDoActionQueue chan string // sent message from 'do-action' goroutine.
}

// Run starts state machine.
func (sm *StateMachine) Run() {
	go sm.do()         // start do-action goroutine
	go sm.run()        // start state transition engine
	sm.enterNewState() // transit from initial state.
}

// Send sends event to state machine.
func (sm *StateMachine) Send(ev *Event) {
	sm.eventQueue <- ev
}

// GetState returns current state.
func (sm *StateMachine) GetState() string {
	return sm.currentState.name
}

// IsEnd returns whether state machine is finished to run or not
func (sm *StateMachine) IsEnd() bool {
	return sm.currentState.isEnd()
}

// run is a goroutine that process state transition.
func (sm *StateMachine) run() {
	for {
		ev := <-sm.eventQueue
		golog.Debug(fmt.Sprintf("state: %s,event: '%+v(id:%+v)'", sm.currentState.name, ev.name, ev.id))

		if err := sm.transit(ev); err != nil {
			if err == FinishToTransitError {
				break
			}
			golog.Error(err)
		}
	}
	golog.Info("exit state machine")
	sm.toOwnerQueue <- stopped
}

// do is a goroutine to run do-action functions.
func (sm *StateMachine) do() {
	// start timer with idle duration to prevent following 'timer.C' with nil value
	isRunning := false
	timer := time.NewTimer(idleTickDuration)

LOOP:
	for {
		select {
		case msg := <-sm.toDoActionQueue: // message from state transition goroutine.
			switch msg {
			case pleaseStop:
				golog.Info("stop request for do-action goroutine detect. stop.")
				break LOOP
			case changeState:
				golog.Info(fmt.Sprintf("change state to %s", sm.currentState.name))

				// stop current running timer.
				if !timer.Stop() {
					<-timer.C // drain channel
				}

				if !sm.currentState.hasDo() {
					// this state not have do-action.
					isRunning = false
					timer.Reset(idleTickDuration) // don't have a do-action.
					break
				}

				// has a do-action, run it.
				do := sm.currentState.do
				golog.Info(fmt.Sprintf("run %s.", do))
				isRunning = true
				d := time.Duration(reflect.ValueOf(sm.bindClass).MethodByName(do).Call([]reflect.Value{})[0].Int())
				if d <= 0 {
					isRunning = false // No more do-actions need to be executed.
					d = idleTickDuration
				}
				golog.Info(fmt.Sprintf("reset do-action interval timer to %d", d))
				timer.Reset(d)
			default:
				golog.Warn(fmt.Sprintf("unexpected message detect:%s", msg))
			}
		case <-timer.C:
			// expire tick timer
			golog.Info(fmt.Sprintf("tick timer expired."))
			if !isRunning {
				timer.Reset(idleTickDuration) // do-action is not running. restart timer with idle duration.
				break
			}

			if !sm.currentState.hasDo() {
				golog.Error("isRunningDoAction is true but do-action is not found. set isRunningDoAction false.")
				isRunning = false
				timer.Reset(idleTickDuration) // do-action is not running. restart timer with idle duration.
				break
			}

			// re-run do-action.
			do := sm.currentState.do
			golog.Info(fmt.Sprintf("run do-action: %s", do))
			d := time.Duration(reflect.ValueOf(sm.bindClass).MethodByName(do).Call([]reflect.Value{})[0].Int())
			if d <= 0 {
				isRunning = false // No more do-actions need to be executed.
				d = idleTickDuration
			}
			golog.Info(fmt.Sprintf("reset do-action interval timer to %d", d))
			timer.Reset(d)
		}
	}
	if !timer.Stop() {
		<-timer.C // drain channel
	}
	// finish goroutine.
	golog.Info("exit do-action goroutine")
	sm.fromDoActionQueue <- stopped // reply.
}

// finish cleanup state machine.
func (sm *StateMachine) finish() {
	golog.Info(fmt.Sprintf("transit %s -> %s", sm.currentState.name, endState.name))
	sm.toDoActionQueue <- pleaseStop
	<-sm.fromDoActionQueue
	golog.Info("do-action goroutine stopped.")
	sm.currentState = &endState
}

// exitOldState calls exit-action functions if needed.
func (sm *StateMachine) exitOldState() {
	if sm.currentState.hasExit() {
		exit := sm.currentState.exit
		golog.Info(fmt.Sprintf("do exit-action: %s", exit))
		reflect.ValueOf(sm.bindClass).MethodByName(exit).Call([]reflect.Value{})
	}
}

// enterNewState calls enter functions if needed.
func (sm *StateMachine) enterNewState() {
	if sm.currentState.hasEntry() {
		entry := sm.currentState.entry
		golog.Info(fmt.Sprintf("Do entry-action: %s", entry))
		reflect.ValueOf(sm.bindClass).MethodByName(entry).Call([]reflect.Value{})
	}
	sm.toDoActionQueue <- changeState
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
		if !item.hasGuard() {
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

	// run action if defined
	if item.hasAction() {
		golog.Trace(fmt.Sprintf("run '%+v'", item.action))
		reflect.ValueOf(sm.bindClass).MethodByName(item.action).Call([]reflect.Value{})
	}

	// state transition. run exit action if specified.
	sm.exitOldState()
	golog.Info(fmt.Sprintf("%s -> %s", sm.currentState.name, item.next.name))

	if item.next.isSame(&endState) {
		// finish.
		sm.finish()
		return FinishToTransitError // reach to end state, shutdown machine.
	}

	// entering new state. run entry and/or do action if specified.
	sm.currentState = item.next
	sm.enterNewState()
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
func NewStateMachine(k interface{}, path string, qSize int, oq chan string) (*StateMachine, error) {
	// array of function name that have to be implemented but not found.
	var missedFunctions []string

	// provision return value.
	sm := &StateMachine{
		bindClass:         k,
		currentState:      nil,
		states:            make([]*State, 0),
		eventQueue:        make(chan *Event, qSize),
		toOwnerQueue:      oq,
		fromDoActionQueue: make(chan string, 1),
		toDoActionQueue:   make(chan string, 1),
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

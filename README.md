# go-state-machine

Execute state transition operation based on [state machine diagram](https://plantuml.com/state-diagram) defined in [Plant UML](https://plantuml.com/) format.

Currently support only flat (no composite) state model like:

```puml
@startuml
[*] --> State1
State1 --> State2 : Succeeded
State1 --> [*] : Aborted
State1 : entry / EnterState1
State1 : do / DoState1
State1 : exit / ExitState1
State2 --> State3 : Succeeded
State2 --> [*] : Aborted
State3 --> [*] : Succeeded / SaveResult
State3 --> [*] : Aborted [MaxCheck]
State3 --> State3 : Failed
@enduml
```

![](./test1.png)

## usage
1. write Plant UML state machine diagram.
1. write state transition code:
    1. Prototype of action function is `func()` (No arguments/No return values).
    1. Prototype of guard function is `func() bool` (No arguments).
    1. Prototype of entry/exit function is `func()` (No arguments/No return values).
    1. Prototype of do-action function is `func() time.Duration` (No arguments).
      - do-action function must return time.Duration value. State machine set timer for returned duration and when timer expired, call do-action function again.
      - do-action function may return zero (and minus) time.Duration value. In this case, state machine never call do-action function again.
    1. All action, guard, entry, exit, and do-action must be started with upper case since they will be called from external [reflect package](https://golang.org/pkg/reflect/).
    1. Generate StateMachine via `NewStateMachine` function with the Plant UML state machine diagram.
      - `NewStateMachine` parsed given diagram. When any non implement action and/or guard methods found, `NewStateMachine` will return error.
    1. Start StateMachine with `Run` method.
    1. send Event to StateMachine
    1. Listen StateMachine response when sent event that transit to end state t to StateMachine.

### example

```go
package x
import (
  "fmt"
  "os"
  "time"

  sm "github.com/marrbor/go-state-machine/statemachine"
)

type T struct{
 counter int
 machine *sm.StateMachine
}

// Action functions
func (t *T) SaveResult() time.Duration { return 1 * time.Second }

// Guard functions
func (t *T) MaxCheck() bool { return true }

// Entry/Exit Functions
func (t *T) EnterState1() {}
func (t *T) ExitState1() {}

// Do Function
func (t *T) DoState1() {
    tq := make(chan error)
    // something to do

    // waiting for action timing or stop message.
LOOP:
    for {
      select {
        case tmsg := <- tq:
          // some thing to do. 
      }
    }
}

func main() {
  var t T
  oq := make(chan string)

  m, err := sm.NewStateMachine(&t, "t.puml", 1, oq) // initial transit to State1
  if err != nil {panic(err)}
  m.Run() // do initial transition

  m.Send(sm.NewEvent("Succeeded")) // transit to State2
  m.Send(sm.NewEvent("Succeeded")) // transit to State3
  m.Send(sm.NewEvent("Aborted"))   // call MaxCheck guard function, if MaxCheck returns true, transit to endState. 
  s := <- oq // Wait for stopping machine.
  if s != sm.stopped {
    panic(fmt.Errorf(s))
  }
  os.Exit(0)
}
```

## License
MIT

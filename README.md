# go-state-machine

Execute state transition operation based on [state machine diagram](https://plantuml.com/state-diagram) defined in [Plant UML](https://plantuml.com/) format.

Currently support only flat (no composite) state model like:

```puml
@startuml
[*] --> State1
State1 --> State2 : Succeeded
State1 --> [*] : Aborted
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
    1. Prototype of action function is `func() time.Duration` (defined as type `Action`).
    1. Prototype of guard function is `func() bool` (defined as type `Guard`).
    1. Both action and guard function must be started with upper case since they will be called from [reflect package](https://golang.org/pkg/reflect/).
    1. generate and start StateMachine via `NewStateMachine` function with the Plant UML state machine diagram.
      - `NewStateMachine` parsed given diagram. When any non implement action and/or guard methods found, `NewStateMachine` will return error.
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

func main() {
  var t T
  m, err := sm.NewStateMachine(&t, "t.puml",1) // initial transit to State1
  if err != nil {panic(err)}

  m.Send(sm.NewEvent("Succeeded")) // transit to State2
  m.Send(sm.NewEvent("Succeeded")) // transit to State3
  m.Send(sm.NewEvent("Aborted"))   // call MaxCheck guard function, if MaxCheck returns true, transit to EndState. 
  s := m.Listen() // Wait for stopping machine.
  if s != sm.Stopped {
    panic(fmt.Errorf(s))
  }
  os.Exit(0)
}
```

## Todo 
- Support entry/exit activities in state.

## License
MIT

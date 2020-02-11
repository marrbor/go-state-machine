# go-state-machine

Execute state transition operation based on [state machine diagram](https://plantuml.com/state-diagram) defined in [Plant UML](https://plantuml.com/) format.

Currently support only flat (no composite) stat model like:
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
1. write plant uml state machine diagram.
1. write state transition code:
    1. define type to hold action and guard functions.
        1. Prototype of action function is `func(sm *StateMachine)` (defined as type `Action`).
        1. Prototype of guard function is `func(sm *StateMachine) bool` (defined as type `Guard`).
    1. generate StateMachine via `NewStateMachine` function with the diagram.
    1. run StateMachine
    1. send Event to StateMachine
    1. Listen StateMachine response when send Shutdown event to StateMachine.

```go
package x
import (
  "os"
  "fmt"
  sm "github.com/marrbor/go-state-machine/statemachine"
)

type T struct{ counter int }
var t T
func (t *T) SaveResult(sm *sm.StateMachine) { }
func (t *T) Shutdown(sm *sm.StateMachine) { }
func (t *T) MaxCheck(sm *sm.StateMachine) bool { return true }

func main() {
  m, err := sm.NewStateMachine(&t, "t.puml") // initial transit to State1
  if err != nil {panic(err)}

  m.Start()
  m.Send(sm.NewEvent("Succeeded")) // transit to State2
  m.Send(sm.NewEvent("Succeeded")) // transit to State3
  m.Send(sm.NewEvent("Aborted")) // call MaxCheck guard function, if MaxCheck returns true, transit to EndState. 
  m.Stop() // stop state machine
  s := m.Listen() // Wait for stopping machine.
  if s != sm.Stopped {
    panic(fmt.Errorf(s))
  }
  os.Exit(0)
}
```

## Todo 
- Support entry/exit activities in state.

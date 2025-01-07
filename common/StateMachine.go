package common

type State interface {
	Equals(State) bool
}
type Event interface {
	Equals(Event) bool
}
type Context interface {
}
type StateMachine struct {
	transitions map[State][]struct {
		to     State
		on     Event
		when   func(Event, Context) bool
		action func(Event, Context)
	}
	CurrnetState State
}

func NewStateMachine(initial State) *StateMachine {
	return &StateMachine{CurrnetState: initial}
}

func (s *StateMachine) AddTransactions(from []State, to State, on Event, when func(Event, Context) bool, action func(Event, Context)) {
	for _, f := range from {
		s.AddTransaction(f, to, on, when, action)
	}
}
func (s *StateMachine) AddTransaction(from State, to State, on Event, when func(Event, Context) bool, action func(Event, Context)) {
	if s.transitions == nil {
		s.transitions = make(map[State][]struct {
			to     State
			on     Event
			when   func(Event, Context) bool
			action func(Event, Context)
		})
	}
	s.transitions[from] = append(s.transitions[from], struct {
		to     State
		on     Event
		when   func(Event, Context) bool
		action func(Event, Context)
	}{to, on, when, action})
}
func (s *StateMachine) Handle(event Event, context Context) {
	if s.transitions == nil {
		return
	}
	for _, t := range s.transitions[s.CurrnetState] {
		if t.on.Equals(event) && t.when(event, context) {
			s.CurrnetState = t.to
			t.action(event, context)
			return
		}
	}
}

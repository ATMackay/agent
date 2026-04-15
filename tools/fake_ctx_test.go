package tools

import (
	"context"
	"iter"
	"maps"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
)

// fakeToolContext is a minimal tool.Context implementation for unit tests.
// It stores key-value state in a plain map and records StateDelta writes.
type fakeToolContext struct {
	context.Context
	st      *fakeState
	actions *session.EventActions
}

func newFakeToolContext(initialState map[string]any) *fakeToolContext {
	st := &fakeState{data: maps.Clone(initialState)}
	if st.data == nil {
		st.data = make(map[string]any)
	}
	return &fakeToolContext{
		Context: context.Background(),
		st:      st,
		actions: &session.EventActions{StateDelta: make(map[string]any)},
	}
}

// --- tool.Context methods used by our tools ---

func (f *fakeToolContext) State() session.State            { return f.st }
func (f *fakeToolContext) Actions() *session.EventActions  { return f.actions }
func (f *fakeToolContext) FunctionCallID() string          { return "test-call-id" }

// --- stub implementations for the rest of tool.Context ---

func (f *fakeToolContext) UserContent() *genai.Content    { return nil }
func (f *fakeToolContext) InvocationID() string           { return "test-inv" }
func (f *fakeToolContext) AgentName() string              { return "test-agent" }
func (f *fakeToolContext) ReadonlyState() session.ReadonlyState { return f.st }
func (f *fakeToolContext) UserID() string                 { return "test-user" }
func (f *fakeToolContext) AppName() string                { return "test-app" }
func (f *fakeToolContext) SessionID() string              { return "test-session" }
func (f *fakeToolContext) Branch() string                 { return "" }
func (f *fakeToolContext) Artifacts() agent.Artifacts     { return nil }
func (f *fakeToolContext) SearchMemory(_ context.Context, _ string) (*memory.SearchResponse, error) {
	return nil, nil
}
func (f *fakeToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }
func (f *fakeToolContext) RequestConfirmation(_ string, _ any) error            { return nil }

// fakeState implements session.State backed by a plain map.
type fakeState struct {
	data map[string]any
}

func (s *fakeState) Get(key string) (any, error) {
	v, ok := s.data[key]
	if !ok {
		return nil, session.ErrStateKeyNotExist
	}
	return v, nil
}

func (s *fakeState) Set(key string, val any) error {
	s.data[key] = val
	return nil
}

func (s *fakeState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

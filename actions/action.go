package actions

// Action defines the interface for all hook actions
type Action interface {
	Execute(args []string) error
}

// BaseAction provides common functionality for all actions
type BaseAction struct {
	name string
}

// NewBaseAction creates a new base action
func NewBaseAction(name string) *BaseAction {
	return &BaseAction{name: name}
}

// GetName returns the action name
func (a *BaseAction) GetName() string {
	return a.name
}

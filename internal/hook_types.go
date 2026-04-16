package internal

// HookEvent represents a point in the terraform processing lifecycle.
type HookEvent string

const (
	HookBeforeParse  HookEvent = "before_parse"   // Before parsing terraform output
	HookAfterParse   HookEvent = "after_parse"    // After parsing, before rendering
	HookBeforeRender HookEvent = "before_render"  // Before rendering markdown
	HookAfterRender  HookEvent = "after_render"   // After rendering, before output
)

// HookFunc is a function that processes data at a hook point.
type HookFunc func(*HookContext) error

// HookContext contains data passed to hook functions.
type HookContext struct {
	Event   HookEvent
	Summary *Summary
	Input   string
	Output  string
	Error   error
}

// Hook represents a registered hook with its event and handler.
type Hook struct {
	Event   HookEvent
	Handler HookFunc
	Name    string
}

// HookRegistry manages registered hooks.
type HookRegistry struct {
	hooks map[HookEvent][]Hook
}

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		hooks: make(map[HookEvent][]Hook),
	}
}

// Register adds a hook to the registry.
func (r *HookRegistry) Register(event HookEvent, name string, handler HookFunc) {
	r.hooks[event] = append(r.hooks[event], Hook{
		Event:   event,
		Handler: handler,
		Name:    name,
	})
}

// Execute runs all hooks for a given event.
func (r *HookRegistry) Execute(ctx *HookContext) error {
	hooks, ok := r.hooks[ctx.Event]
	if !ok {
		return nil
	}

	for _, hook := range hooks {
		if err := hook.Handler(ctx); err != nil {
			return err
		}
	}
	return nil
}

// HasHooks returns true if there are hooks registered for an event.
func (r *HookRegistry) HasHooks(event HookEvent) bool {
	hooks, ok := r.hooks[event]
	return ok && len(hooks) > 0
}

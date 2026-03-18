package tools

import "fmt"

// Deps enables arbitrary tool configurations. TODO may be refactored in future.
type Deps struct {
	Configs map[Kind]any
}

func (d *Deps) AddConfig(kind Kind, cfg any) {
	if d.Configs == nil {
		d.Configs = make(map[Kind]any)
	}
	d.Configs[kind] = cfg
}


// getConfig returns config for the specified tool type.
func getConfig[T any](kind Kind, deps *Deps) (T, error) {
	var zero T

	if deps.Configs == nil {
		return zero, fmt.Errorf("no configs provided")
	}

	raw, ok := deps.Configs[kind]
	if !ok {
		return zero, fmt.Errorf("missing config for tool %q", kind)
	}

	cfg, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("invalid config type for tool %q: got %T", kind, raw)
	}

	return cfg, nil
}
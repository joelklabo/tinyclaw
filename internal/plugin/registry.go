package plugin

import "fmt"

// Registry holds named plugin factories.
type Registry struct {
	transports map[string]func() Transport
	harnesses  map[string]func() Harness
	contexts   map[string]func() ContextProvider
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		transports: make(map[string]func() Transport),
		harnesses:  make(map[string]func() Harness),
		contexts:   make(map[string]func() ContextProvider),
	}
}

// RegisterTransport registers a transport factory by name.
func (r *Registry) RegisterTransport(name string, factory func() Transport) error {
	if _, ok := r.transports[name]; ok {
		return fmt.Errorf("transport %q already registered", name)
	}
	r.transports[name] = factory
	return nil
}

// GetTransport returns a new Transport instance by name.
func (r *Registry) GetTransport(name string) (Transport, error) {
	factory, ok := r.transports[name]
	if !ok {
		return nil, fmt.Errorf("transport %q not found", name)
	}
	return factory(), nil
}

// RegisterHarness registers a harness factory by name.
func (r *Registry) RegisterHarness(name string, factory func() Harness) error {
	if _, ok := r.harnesses[name]; ok {
		return fmt.Errorf("harness %q already registered", name)
	}
	r.harnesses[name] = factory
	return nil
}

// GetHarness returns a new Harness instance by name.
func (r *Registry) GetHarness(name string) (Harness, error) {
	factory, ok := r.harnesses[name]
	if !ok {
		return nil, fmt.Errorf("harness %q not found", name)
	}
	return factory(), nil
}

// RegisterContext registers a context provider factory by name.
func (r *Registry) RegisterContext(name string, factory func() ContextProvider) error {
	if _, ok := r.contexts[name]; ok {
		return fmt.Errorf("context %q already registered", name)
	}
	r.contexts[name] = factory
	return nil
}

// GetContext returns a new ContextProvider instance by name.
func (r *Registry) GetContext(name string) (ContextProvider, error) {
	factory, ok := r.contexts[name]
	if !ok {
		return nil, fmt.Errorf("context %q not found", name)
	}
	return factory(), nil
}

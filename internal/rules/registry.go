package rules

import (
	"fmt"
	"sort"
	"sync"
)

// registry is the package-level store of Rules. Access is guarded by
// a mutex so init-time registration from multiple sub-packages is
// race-free; reads after init (the hot path) take the read-lock.
var registry = struct {
	mu    sync.RWMutex
	byID  map[string]Rule
	order []string
}{
	byID: make(map[string]Rule),
}

// Register adds r to the registry. Called from rule sub-packages'
// init() functions. A duplicate ID panics — conflicting IDs mean a
// build mistake and silently preferring one rule over another would
// hide bugs in release builds.
func Register(r Rule) {
	if r == nil {
		panic("rules.Register: nil Rule")
	}
	id := r.ID()
	if id == "" {
		panic("rules.Register: empty rule ID")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.byID[id]; exists {
		panic(fmt.Sprintf("rules.Register: duplicate rule ID %q", id))
	}
	registry.byID[id] = r
	registry.order = append(registry.order, id)
	sort.Strings(registry.order)
}

// All returns the registered Rules in stable ID order. The returned
// slice is a copy — callers may freely sort or filter it without
// affecting the registry.
func All() []Rule {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	out := make([]Rule, len(registry.order))
	for i, id := range registry.order {
		out[i] = registry.byID[id]
	}
	return out
}

// Get returns the Rule with the given id, or nil if unknown.
func Get(id string) Rule {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.byID[id]
}

// reset clears the registry. Used only in tests to isolate registration
// between test cases. Exported as Reset for the engine's test harness.
func reset() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.byID = make(map[string]Rule)
	registry.order = registry.order[:0]
}

// Reset is the test-only entry point for clearing the registry. It is
// exported so engine_test (in a different package) can isolate its
// stub rules from the real registry. Production code must not call it.
func Reset() { reset() }

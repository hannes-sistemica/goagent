package context

import (
	"agent-server/internal/models"
	"context"
)

// ContextStrategy defines the interface for building conversation context
type ContextStrategy interface {
	BuildContext(ctx context.Context, systemPrompt, agentPrompt string, messages []*models.Message, config map[string]interface{}) ([]*models.Message, error)
	Name() string
	DefaultConfig() map[string]interface{}
}

// StrategyRegistry manages available context strategies
type StrategyRegistry struct {
	strategies map[string]ContextStrategy
}

// NewStrategyRegistry creates a new strategy registry with default strategies
func NewStrategyRegistry() *StrategyRegistry {
	registry := &StrategyRegistry{
		strategies: make(map[string]ContextStrategy),
	}

	// Register default strategies
	registry.Register(&LastNStrategy{})
	registry.Register(&SlidingWindowStrategy{})
	registry.Register(&SummarizeStrategy{})

	return registry
}

// Register adds a strategy to the registry
func (r *StrategyRegistry) Register(strategy ContextStrategy) {
	r.strategies[strategy.Name()] = strategy
}

// Get retrieves a strategy by name
func (r *StrategyRegistry) Get(name string) (ContextStrategy, bool) {
	strategy, exists := r.strategies[name]
	return strategy, exists
}

// List returns all available strategy names
func (r *StrategyRegistry) List() []string {
	names := make([]string, 0, len(r.strategies))
	for name := range r.strategies {
		names = append(names, name)
	}
	return names
}
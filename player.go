package orchestra

import "context"

// Player is the abstraction for a service that would run in goroutines
//
// life cycle of a player is: [Setup] -> [Play] -> [Clean]
//
// however it may also be   : [Setup] -> [Clean]
//
//
// It is gauranteed that Play is called only after the Setup returns a nil error
// This allows separation of initialization, clean up, and work logic
type Player interface {
	Play(context.Context) error
	Setup() error
	Clean()
}

// SimplePlayer can be used if the player doesn't require setup or clean up,
// It implements the `Player` interface, so a `func(context.Context) error` can be casted to it
// and can be used anywhere a `Player` maybe required. (for eg, in a stage)
type SimplePlayer func(context.Context) error

// Setup returns a nil error
func (sp SimplePlayer) Setup() error {
	return nil
}

// Clean does nothing
func (sp SimplePlayer) Clean() {}

// Play calls the SimplePlayer with the given context
func (sp SimplePlayer) Play(ctx context.Context) error {
	return sp(ctx)
}

// Package orchestra is a module that provides a minimal structure for orchestrating worker goroutines.
// It defines a life cycle for the workers.
package orchestra

import (
	"context"
	"fmt"
	"sync"
)

// ErrSetup is the error returned by (*Stage).Setup()
type ErrSetup struct {
	Player string // name of the player
	Err    error  // the error returned by setup method of the player
}

func (e ErrSetup) Error() string {
	return fmt.Sprintf("ErrSetup: %s: %s", e.Player, e.Err)
}

// ErrPlay is the error returned by (*Stage).Play()
type ErrPlay struct {
	Players map[string]error
}

func (e *ErrPlay) Error() string {
	k := "ErrPlay:"
	for name, err := range e.Players {
		k += fmt.Sprintf(" |%s: %s|", name, err)
	}
	return k
}

// Stage is the abstraction that allows services to be added and played together, and get cancelled.
// It facilitates graceful shutdown
//
// Note: Stage also implements `orchestra.Player`, so stages can nested
type Stage struct {
	players   map[string]Player
	beenSetup bool
}

// NewStage creates a new empty stage
func NewStage() *Stage {
	return &Stage{
		players: make(map[string]Player),
	}
}

// Add adds a player to the stage
func (s *Stage) Add(name string, p Player) {
	s.players[name] = p
}

// Setup sets up all the players in this stage.
// If any player returns error while setting up, Setup returns immediately.
// The stage is setup as a whole, "if any player fails to setup: The stage fails to setup".
//
// if err is non-nil, it is of type `ErrSetup`
// also, if err is non-nil, all the players that were successfully setup, before the faulty one, will be cleaned
func (s *Stage) Setup() error {
	// (*Stage).beenSetup is set iff all players are setup with nil errors.
	// because, "if any player fails to setup: The stage fails to setup"
	var err error
	var good []Player
	var faulty string
	for name, it := range s.players {
		err = it.Setup()
		if err != nil {
			faulty = name
			break
		}
		good = append(good, it)
	}
	if err != nil {
		for _, it := range good {
			it.Clean()
		}
		return ErrSetup{
			Player: faulty,
			Err:    err,
		}
	}
	s.beenSetup = true
	return nil
}

// Clean calls Clean on every player in this stage
func (s *Stage) Clean() {
	wg := &sync.WaitGroup{}
	wg.Add(len(s.players))
	for _, it := range s.players {
		go func(player Player) {
			defer wg.Done()
			player.Clean()
		}(it)
	}
	wg.Wait()
}

// Play starts a goroutine for every player in this stage, and calls each player's Play from within.
// It blocks till all the player returns, all the errors returned by the players are accumlated.
// Also, (*Stage).Play panics if the stage hasn't been setup successfully, i.e. with nil error
//
// A non-nil error is returned iff at least one player returned a non-nil error
func (s *Stage) Play(ctx context.Context) error {
	if !s.beenSetup {
		panic("(*Stage).Play: The stage hasn't been successfully setup")
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(s.players))
	echan := make(chan struct {
		Name string
		Err  error
	}, len(s.players))

	for name, it := range s.players {
		go func(n string, player Player) {
			defer wg.Done()
			e := player.Play(ctx)
			if e != nil {
				echan <- struct {
					Name string
					Err  error
				}{Name: n, Err: e} // this send will never block
			}
		}(name, it)
	}

	wg.Wait()    // wait till all the players are done with their shit
	close(echan) // since all the players are done, we can safely close this channel

	var err *ErrPlay = nil
	for e := range echan {
		if err == nil {
			err = &ErrPlay{
				Players: make(map[string]error),
			}
		}
		err.Players[e.Name] = e.Err
	}
	if err == nil {
		return nil
	}
	return err
}

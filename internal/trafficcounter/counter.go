// Package trafficcounter computes confirmed increases without guessing that a
// decreasing aggregate has reset. Its state must survive process restarts.
package trafficcounter

import "fmt"

type Direction struct {
	HighWater int64 `json:"high_water"`
	Pending   bool  `json:"pending,omitempty"`
}

type State struct {
	Initialized bool      `json:"initialized"`
	Up          Direction `json:"up"`
	Down        Direction `json:"down"`
	Epoch       uint64    `json:"epoch,omitempty"`
	Bits        uint8     `json:"bits,omitempty"`
}

type Delta struct {
	Up        int64
	Down      int64
	Uncertain bool
}

// Validate describes a counter domain, not the host's boot ID: an increasing epoch
// guarantees both counters started at zero. Its generation is persisted by the
// agent across restarts; it is not an arbitrary/reusable boot identifier. Bits=32 is only meaningful for a
// single modulo-2^32 counter, never a sum of independently wrapping interfaces.
func Validate(up, down int64, epoch uint64, bits uint8) error {
	if up < 0 || down < 0 {
		return fmt.Errorf("negative network counter")
	}

	if bits != 0 && bits != 32 && bits != 64 {
		return fmt.Errorf("network counter bits must be 0, 32 or 64")
	}
	if bits == 32 && (epoch == 0 || up >= 1<<32 || down >= 1<<32) {
		return fmt.Errorf("32-bit network counters require an epoch and values below 2^32")
	}
	return nil
}

// Advance returns a new state so failed writes cannot advance the baseline.
// Legacy observations without an epoch are lower-bound measurements after a
// discontinuity: retain the high-water mark and flag every affected interval.
// Rates are intentionally not used to invent a reset or cap legitimate growth.
func (s State) Advance(up, down int64, epoch uint64, bits uint8) (State, Delta, error) {
	if err := Validate(up, down, epoch, bits); err != nil {
		return s, Delta{}, err
	}
	if !s.Initialized {
		return State{Initialized: true, Up: Direction{HighWater: up}, Down: Direction{HighWater: down}, Epoch: epoch, Bits: bits}, Delta{}, nil
	}
	if s.Up.HighWater < 0 || s.Down.HighWater < 0 {
		return s, Delta{}, fmt.Errorf("invalid persisted traffic high-water mark")
	}
	if s.Epoch != 0 && epoch < s.Epoch {
		// A generation must increase across resets. Old or missing metadata cannot
		// roll the accepted counter domain backwards, even after a server restart.
		s.Up.Pending, s.Down.Pending = true, true
		return s, Delta{Uncertain: true}, nil
	}
	if s.Epoch == epoch && s.Epoch != 0 && (s.Bits == 32) != (bits == 32) {
		return s, Delta{}, fmt.Errorf("counter width changed without a new epoch")
	}
	if s.Epoch != epoch {
		confirmed := s.Epoch != 0 && epoch > s.Epoch
		next := State{Initialized: true, Up: Direction{HighWater: up}, Down: Direction{HighWater: down}, Epoch: epoch, Bits: bits}
		if confirmed {
			return next, Delta{Up: up, Down: down}, nil
		}
		return next, Delta{Uncertain: true}, nil
	}
	s.Bits = bits

	advance := func(d *Direction, current int64) (int64, bool) {
		if current >= d.HighWater {
			increase, pending := current-d.HighWater, d.Pending
			d.HighWater, d.Pending = current, false
			return increase, pending
		}
		if bits == 32 && !d.Pending && d.HighWater >= 3*(1<<30) && current < 1<<30 {
			increase := (int64(1) << 32) - d.HighWater + current
			d.HighWater = current
			return increase, false
		}
		d.Pending = true
		return 0, true
	}
	u, uq := advance(&s.Up, up)
	d, dq := advance(&s.Down, down)
	return s, Delta{Up: u, Down: d, Uncertain: uq || dq}, nil
}

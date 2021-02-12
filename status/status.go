package status

import (
	"sync"
)

type Snapturn int

const (
	None Snapturn = iota
	Left
	Right
)

type ControlState struct {
	sync.RWMutex
	Direction, Velocity float32
	Turning             Snapturn
}

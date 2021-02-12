package motorspeed

import (
	"fmt"
)

type Motorspeed struct {
	X, Y, Z, W int
}

func (m Motorspeed) Serialize() (s string) {
	s += fmt.Sprintf("x%+.2x\n", m.X)
	s += fmt.Sprintf("y%+.2x\n", m.Y)
	s += fmt.Sprintf("z%+.2x\n", m.Z)
	s += fmt.Sprintf("w%+.2x\n", m.W)
	return
}

func (mf Motorspeed) ToAll(f func(int) int) (mt Motorspeed) {
	mt.X = f(mf.X)
	mt.Y = f(mf.Y)
	mt.Z = f(mf.Z)
	mt.W = f(mf.W)
	return
}

func (ma Motorspeed) Combine(mb Motorspeed, f func(int, int) int) (mt Motorspeed) {
	mt.X = f(ma.X, mb.X)
	mt.Y = f(ma.Y, mb.Y)
	mt.Z = f(ma.Z, mb.Z)
	mt.W = f(ma.W, mb.W)
	return
}

func (mf Motorspeed) Gain(g float32) Motorspeed {
	return mf.ToAll(func(f int) int {
		return int(float32(f) * g)
	})
}

func (a Motorspeed) Lerp(b Motorspeed, v float32) (tm Motorspeed) {
	return a.Combine(b, func(a, b int) int {
		return a + int(float32(b-a) * v)
	})
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (mf Motorspeed) Limit() Motorspeed {
	return mf.ToAll(func(f int) int {
		return min(255, max(-255, f))
	})
}

func SnapturnLeft() Motorspeed {
	return Motorspeed{255, 255, 255, 255}
}

func SnapturnRight() Motorspeed {
	return Motorspeed{-255, -255, -255, -255}
}

func MoveForward() Motorspeed {
	return Motorspeed{255, -255, -255, 255}
}

func MoveBack() Motorspeed {
	return Motorspeed{-255, 255, 255, -255}
}

func MoveLeft() Motorspeed {
	return Motorspeed{-255, -255, 255, 255}
}

func MoveRight() Motorspeed {
	return Motorspeed{255, 255, -255, -255}
}

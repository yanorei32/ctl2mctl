package sender

import (
	"fmt"
	"io"
	"time"

	"github.com/yanorei32/ctl2mctl/motorspeed"
	"github.com/yanorei32/ctl2mctl/status"
)

type Motor int

type Sender struct {
	MaxChangeAmount int
	Interval        time.Duration
}

func createMotorspeed(
	s *status.ControlState,
) motorspeed.Motorspeed {
	s.RLock()
	defer s.RUnlock()

	if s.Turning == status.Right {
		return motorspeed.SnapturnRight().Gain(0.5)

	} else if s.Turning == status.Left {
		return motorspeed.SnapturnLeft().Gain(0.5)

	} else if s.Direction < -90 {
		return motorspeed.MoveLeft().Lerp(
			motorspeed.MoveBack(),
			float32(-s.Direction-90)/90,
		).Gain(s.Velocity)

	} else if s.Direction < 0 {
		return motorspeed.MoveForward().Lerp(
			motorspeed.MoveLeft(),
			float32(-s.Direction)/90,
		).Gain(s.Velocity)

	} else if s.Direction < 90 {
		return motorspeed.MoveForward().Lerp(
			motorspeed.MoveRight(),
			float32(s.Direction)/90,
		).Gain(s.Velocity)

	} else {
		return motorspeed.MoveRight().Lerp(
			motorspeed.MoveBack(),
			float32(s.Direction-90)/90,
		).Gain(s.Velocity)

	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func findMax(m motorspeed.Motorspeed) int {
	maxV := 0

	if maxV < m.X {
		maxV = m.X
	}

	if maxV < m.Y {
		maxV = m.Y
	}

	if maxV < m.Z {
		maxV = m.Z
	}

	if maxV < m.W {
		maxV = m.W
	}

	return maxV
}

func (s Sender) dampMotorspeed(
	virt motorspeed.Motorspeed,
	curr motorspeed.Motorspeed,
) motorspeed.Motorspeed {
	absDist := virt.Combine(curr, func(a, b int) int {
		return abs(a - b)
	})

	maxAbsDist := findMax(absDist)

	if maxAbsDist < s.MaxChangeAmount {
		return virt
	}

	dist := virt.Combine(curr, func(a, b int) int {
		return a - b
	})

	return curr.Combine(
		dist.Gain(float32(s.MaxChangeAmount) / float32(maxAbsDist)),
		func(a, b int) int {
			return a + b
		},
	).Limit()
}

func (s Sender) Send(
	from *status.ControlState,
	to io.Writer,
) {
	currMS := motorspeed.Motorspeed{}

	sendTick := time.NewTicker(s.Interval)
	for {
		select {
		case <-sendTick.C:
			virtMS := createMotorspeed(from).Limit()
			currMS = s.dampMotorspeed(virtMS, currMS)
			fmt.Fprintf(to, currMS.Serialize())
		}
	}
}

func NewSender() Sender {
	return Sender{
		Interval:        50 * time.Millisecond,
		MaxChangeAmount: 10,
	}
}

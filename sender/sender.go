package sender

import (
	"fmt"
	"io"
	"time"

	"github.com/yanorei32/ctl2mctl/motorspeed"
	"github.com/yanorei32/ctl2mctl/status"
)

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

func (s Sender) dampMotorspeed(
	virt motorspeed.Motorspeed,
	curr motorspeed.Motorspeed,
) (next motorspeed.Motorspeed) {
	return virt
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

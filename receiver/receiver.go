package receiver

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yanorei32/ctl2mctl/status"
)

type Receiver struct {
	Timeout          time.Duration
	SnapturnDuration time.Duration
}

func (r Receiver) processSnapturnCmd(
	l string,
	s *status.ControlState,
) error {
	argv := strings.Split(l, " ")

	s.Lock()
	defer s.Unlock()

	switch argv[1] {
	case "left":
		s.Turning = status.Left

	case "right":
		s.Turning = status.Right

	default:
		return fmt.Errorf("TurnCmd : Invalid turn \"%s\"", argv[1])
	}

	go (func() {
		time.Sleep(r.SnapturnDuration)

		s.Lock()
		defer s.Unlock()
		s.Turning = status.None
	})()

	return nil
}

func (r Receiver) processMoveCmd(
	l string,
	s *status.ControlState,
) error {
	argv := strings.Split(l, " ")

	vel, err := strconv.ParseFloat(argv[1], 32)
	if err != nil {
		return err
	}

	if vel < 0 || 1 < vel {
		return fmt.Errorf("MoveCmd : Invalid velocity \"%f\"", vel)
	}

	dir, err := strconv.ParseFloat(argv[2], 32)
	if err != nil {
		return err
	}

	if dir < -180 || 180 < dir {
		return fmt.Errorf("MoveCmd : Invalid direction \"%f\"", dir)
	}

	s.Lock()
	s.Velocity = float32(vel)
	s.Direction = float32(dir)
	s.Unlock()

	return nil
}

func (r Receiver) processLine(
	l string,
	s *status.ControlState,
) error {
	var regexpMove = regexp.MustCompile(
		"^move \\d(\\.\\d+)? -?\\d+(\\.\\d+)?$",
	)

	var regexpTurn = regexp.MustCompile(
		"^snapturn (left|right)$",
	)

	if regexpMove.MatchString(l) {
		return r.processMoveCmd(l, s)

	} else if regexpTurn.MatchString(l) {
		return r.processSnapturnCmd(l, s)

	} else {
		return fmt.Errorf("ProcessLine : unsupported command \"%s\"", l)

	}
}

func (r Receiver) Receive(
	from io.Reader,
	to *status.ControlState,
	log *log.Logger,
) {
	incoming := make(chan string)
	ioerr := make(chan error)

	go (func() {
		reader := bufio.NewReader(from)
		for {
			nl, err := reader.ReadString('\n')

			if err != nil {
				ioerr <- err
			}

			// remove '\n'
			incoming <- nl[:len(nl)-1]
		}
	})()

	for {
		timeout := time.NewTimer(r.Timeout)

		select {
		case l := <-incoming:
			if err := r.processLine(l, to); err != nil {
				log.Println(err)
			}

		case <-timeout.C:
			to.Lock()
			to.Velocity = 0.0
			to.Unlock()

		case err := <-ioerr:
			log.Fatal(err)

		}
	}
}

func NewReceiver() Receiver {
	return Receiver{
		Timeout:          250 * time.Millisecond,
		SnapturnDuration: 250 * time.Millisecond,
	}
}

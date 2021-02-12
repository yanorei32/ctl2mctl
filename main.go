package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type SnapTurn int

const (
	None SnapTurn = iota
	Left
	Right
)

type MotorSpeed struct {
	X, Y, Z, W int
}

type ControlState struct {
	sync.RWMutex
	Direction, Velocity float32
	Turning             SnapTurn
}

type MotorState struct {
	sync.RWMutex
	MotorSpeed MotorSpeed
}

// func debug(log *log.Logger) {
// 	for {
// 		time.Sleep(time.Millisecond * 50)
//
// 		state.RLock()
// 		log.Println(state.Turning)
// 		state.RUnlock()
// 	}
// }

func motorspeedLerp(a, b MotorSpeed, v float32) (x MotorSpeed) {
	x.X = a.X + int(float32(b.X-a.X)*v)
	x.Y = a.Y + int(float32(b.Y-a.Y)*v)
	x.Z = a.Z + int(float32(b.Z-a.Z)*v)
	x.W = a.W + int(float32(b.W-a.W)*v)
	return x
}

func motorspeedGain(a MotorSpeed, gain float32) (x MotorSpeed) {
	x.X = int(float32(a.X) * gain)
	x.Y = int(float32(a.Y) * gain)
	x.Z = int(float32(a.Z) * gain)
	x.W = int(float32(a.W) * gain)
	return x
}

func max(a, b int) int {
	if a < b {
		return b
	} else {
		return a
	}
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func serializeSingleMotorSpeed(v int) string {
	s := ""

	if v < 0 {
		s += "-"
		v = -v
	} else {
		s += "+"
	}

	s += fmt.Sprintf("%.2x", max(min(v, 255), 0))

	return s
}

func translator(c *ControlState, m *MotorState, interval time.Duration) {
	stRight := MotorSpeed{255, 255, 255, 255}
	stLeft := MotorSpeed{-255, -255, -255, -255}
	forward := MotorSpeed{255, -255, -255, 255}
	back := MotorSpeed{-255, 255, 255, -255}
	left := MotorSpeed{-255, -255, 255, 255}
	right := MotorSpeed{255, 255, -255, -255}

	var ms MotorSpeed

	for {
		c.RLock()

		if c.Turning == Right {
			// turn right
			ms = motorspeedGain(
				stRight,
				0.5,
			)
		} else if c.Turning == Left {
			// turn left
			ms = motorspeedGain(
				stLeft,
				0.5,
			)
		} else if c.Direction < -90 {
			// left - back
			ms = motorspeedGain(
				motorspeedLerp(
					left, back,
					float32(-c.Direction-90)/90,
				),
				c.Velocity,
			)
		} else if c.Direction < 0 {
			// left - forward
			ms = motorspeedGain(
				motorspeedLerp(
					forward, left,
					float32(-c.Direction)/90,
				),
				c.Velocity,
			)
		} else if c.Direction < 90 {
			// forward - right
			ms = motorspeedGain(
				motorspeedLerp(
					forward, right,
					float32(c.Direction)/90,
				),
				c.Velocity,
			)
		} else {
			// right - back
			ms = motorspeedGain(
				motorspeedLerp(
					right, back,
					float32(c.Direction-90)/90,
				),
				c.Velocity,
			)
		}

		c.RUnlock()

		m.Lock()
		m.MotorSpeed = ms
		m.Unlock()

		time.Sleep(interval)
	}
}

func receiver(
	r *bufio.Reader,
	c *ControlState,
	log *log.Logger,
	timeout time.Duration,
) {
	snapTurnDuration := 250 * time.Millisecond
	moveRegexp := regexp.MustCompile("^move \\d(\\.\\d+)? -?\\d+(\\.\\d+)?$")
	snapTurnRegexp := regexp.MustCompile("^snapturn (left|right)$")

	renew := make(chan int)

	go (func() {
		for {
			select {
			case <-renew:
			case <-time.After(1 * time.Second):
				c.Lock()
				c.Velocity = float32(0)
				c.Unlock()
			}
		}
	})()

	for {
		l, err := r.ReadString('\n')

		if err != nil {
			log.Fatal(err)
		}

		l = l[:len(l)-1]

		lb := []byte(l)
		renew <- 1

		if moveRegexp.Match(lb) {
			argv := strings.Split(l, " ")

			velocity, err := strconv.ParseFloat(argv[1], 32)
			if err != nil {
				log.Fatal(err)
			}

			if velocity < 0 || 1 < velocity {
				log.Printf("Invalid velocity: %f\n", velocity)
				continue
			}

			direction, err := strconv.ParseFloat(argv[2], 32)
			if err != nil {
				log.Fatal(err)
			}

			if direction < -180 || 180 < direction {
				log.Printf("Invalid direction: %f\n", direction)
				continue
			}

			c.Lock()
			c.Velocity = float32(velocity)
			c.Direction = float32(direction)
			c.Unlock()

		} else if snapTurnRegexp.Match(lb) {
			var t SnapTurn
			argv := strings.Split(l, " ")

			if argv[1] == "left" {
				t = Left

			} else if argv[1] == "right" {
				t = Right

			} else {
				log.Fatal("Invalid sanpturn: " + argv[1])
			}

			go (func() {
				time.Sleep(snapTurnDuration)
				c.Lock()
				c.Turning = None
				c.Unlock()
			})()

			c.Lock()
			c.Turning = t
			c.Unlock()
		} else {
			log.Printf("invalid command: %v\n", l)
		}
	}
}

func dumper(v *MotorState, r *MotorState, interval time.Duration) {
	for {
		v.RLock()
		r.Lock()

		r.Unlock()
		v.RUnlock()

		time.Sleep(interval)
	}
}

func sender(m *MotorState, interval time.Duration, o io.Writer) {
	for {
		m.RLock()

		m.RUnlock()
		time.Sleep(interval)
	}
}

func main() {
	l := log.New(os.Stderr, "", 0)

	cState := ControlState{}
	virtMState := MotorState{}
	realMState := MotorState{}

	timeout := 250 * time.Millisecond
	interval := 50 * time.Millisecond

	go receiver(bufio.NewReader(os.Stdin), &cState, l, timeout)
	go translator(&cState, &virtMState, interval)
	go dumper(&virtMState, &realMState, interval)
	go sender(&realMState, interval, os.Stdout)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yanorei32/ctl2mctl/receiver"
	"github.com/yanorei32/ctl2mctl/sender"
	"github.com/yanorei32/ctl2mctl/status"
)

func main() {
	l := log.New(os.Stderr, "", 0)

	state := status.ControlState{}

	go receiver.NewReceiver().Receive(os.Stdin, &state, l)
	go sender.NewSender().Send(&state, os.Stdout)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

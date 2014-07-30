package pipeline

/*
Package pipeline - Goroutine Pipeline Management

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"log"
)

type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
	}
}

var debug = debugging(false)

func SetDebug(enabled bool) {
	debug = debugging(enabled)
}

type Upstreamer interface {
	AbortWithError(error) error
	Close()
	Init(errors chan error, done chan struct{})
	Run() error
	AbortSending()
	ErrorChannel() chan error
	Upstreamer() Upstreamer
	SetUpstream(Upstreamer)
}

type Node struct {
	abortCh  chan struct{}
	doneCh   chan struct{}
	errorCh  chan error
	Upstream Upstreamer
}

func (n *Node) String() string {
	return fmt.Sprintf("Node:%p", n)
}

func (n *Node) Upstreamer() Upstreamer {
	return n.Upstream
}

func (n *Node) SetUpstream(upstream Upstreamer) {
	n.Upstream = upstream
}

func (n *Node) AbortChannel() chan struct{} {
	return n.abortCh
}

func (n *Node) AbortWithError(err error) error {
	if up := n.Upstreamer(); up != nil {
		up.AbortSending()
	}
	n.ErrorChannel() <- err // non-buffered, makes sure someone is listening
	// make sure "return node.AbortWithError(e)" does not send the error twice
	return nil
}

func (n *Node) Init(errors chan error, done chan struct{}) {
	if n.AbortChannel() != nil {
		panic("Node has already been initialized")
	}
	n.errorCh = errors
	n.doneCh = done
	// aborts are buffered, sender can finish its termination without waiting
	n.abortCh = make(chan struct{}, 1)
}

func (n *Node) AbortSending() {
	// non-blocking send, channel may already be blocked
	select {
	case n.AbortChannel() <- struct{}{}:
	default:
		debug.Printf("abort channel already full, no need to re-send")
	}
}

func (n *Node) Close() {
	// abort channel it left open, they are closed by Wait() at the very end
	if up := n.Upstreamer(); up != nil {
		up.AbortSending()
	}
	n.doneCh <- struct{}{}
}

func (n *Node) ErrorChannel() chan error {
	return n.errorCh
}

func Start(node Upstreamer) {
	go func() {
		defer node.Close()
		if err := node.Run(); err != nil {
			node.AbortWithError(err)
		}
	}()
}

func Wait(upstream ...Upstreamer) <-chan error {
	n := []Upstreamer{}
	// filter out nil values
	var prev Upstreamer
	done := make(chan struct{}, 0)
	// error channel is unbuffered to help locating concurrency/teardown issues
	errors := make(chan error, 0)

	remaining := 0
	for i := 0; i < len(upstream); i++ {
		node := upstream[i]
		if node != nil {
			if prev != nil {
				node.SetUpstream(prev)
			}
			node.Init(errors, done)
			Start(node)
			n = append(n, node)
			prev = node
			remaining++
		}
	}

	waitOutput := make(chan error, 0) // blocking on purpose
	go func() {
		defer func() {
			debug.Printf("Wait() finished processing")
			close(waitOutput)
			close(done)
			close(errors)
		}()
		debug.Printf("Wait() started")
		for {
			select {
			case node, ok := <-done:
				if ok {
					debug.Printf("Wait(): node %s finished\n",
						node)
					remaining--
					if remaining <= 0 {
						debug.Printf("Wait() all nodes finished\n")
						return
					}
				} else {
					panic("done channel closed by someone")
				}
			case err, ok := <-errors:
				if ok {
					debug.Printf("Wait() got error: %s\n", err)
					waitOutput <- err
				} else {
					panic("error channel closed by someone")
				}
			}
		}
	}()
	return waitOutput
}

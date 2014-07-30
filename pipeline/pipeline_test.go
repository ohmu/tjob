package pipeline

import (
	"fmt"
	"testing"
)

type MyNode struct {
	Node
	Input      chan int
	Output     chan int
	abortAt    int
	wasAborted bool
}

func (n *MyNode) Close() {
	n.Node.Close()
	if n.Output != nil {
		close(n.Output)
	}
}

type generator struct {
	MyNode
}

func (node *generator) Run() error {
	abort := node.AbortChannel()
	for i := 0; i < 10; i++ {
		select {
		case _, ok := <-abort:
			if ok {
				fmt.Println("generator got abort signal")
				node.wasAborted = true
				return nil
			}
			fmt.Println("generator abort ch closed")
			abort = nil
		default:
			if node.abortAt > 0 && i >= node.abortAt {
				fmt.Println("generator aborting with error")
				return node.AbortWithError(
					fmt.Errorf("value %d too big", i))
			}
			fmt.Println("generator sending", i)
			node.Output <- i
			//time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("generator exiting normally")
	return nil
}

type modifier struct {
	MyNode
}

func (node *modifier) Run() error {
	input, errors := node.Input, node.UpstreamErrors()
	count := 0
	for {
		select {
		case value, ok := <-input:
			if ok {
				fmt.Println("modifier got value", value)
				node.Output <- value * value
				count++
				if node.abortAt > 0 && count > node.abortAt {
					return node.AbortWithError(
						fmt.Errorf("modifier failed"))
				}
			} else {
				fmt.Println("modifier input closed")
				input = nil
			}
		case err, ok := <-errors:
			if ok {
				fmt.Println("modifier upstream error:", err)
				node.ErrorChannel() <- err
			} else {
				fmt.Println("modifier error channel closed")
				errors = nil
			}
		}
		if input == nil && errors == nil {
			fmt.Println("modifier exited normally")
			return nil
		}
	}
}

type printer struct {
	MyNode
}

func (node *printer) Run() error {
	input, errors := node.Input, node.UpstreamErrors()
	for {
		select {
		case value, ok := <-input:
			if !ok {
				fmt.Println("printer input closed")
				input = nil
			} else {
				fmt.Println("printer got value:", value)
			}
		case err, ok := <-errors:
			if !ok {
				fmt.Println("printer error channel closed")
				errors = nil

			} else {
				//time.Sleep(100 * time.Millisecond)
				fmt.Println("printer upstream error:", err)
				node.ErrorChannel() <- err
			}
		}
		if input == nil && errors == nil {
			fmt.Println("printer exited normally")
			return nil
		}
	}
}

func TestBasic(t *testing.T) {
	SetDebug(true)

	n1 := generator{}
	n1.Output = make(chan int, 10)

	n2 := modifier{}
	n2.Input = n1.Output
	n2.Output = make(chan int, 10)

	n3 := printer{}
	n3.Input = n2.Output

	// extra nil's must be ignored
	for err := range Wait(nil, &n1, nil, &n2, nil, &n3, nil) {
		t.Errorf("did not expect error: %s", err)
	}
}

func TestAbortFirst(t *testing.T) {
	SetDebug(true)

	n1 := generator{MyNode{Output: make(chan int, 2)}}
	n2 := modifier{MyNode: MyNode{Input: n1.Output, Output: make(chan int, 10),
		abortAt: 5}} // modifier aborts in the middle of the input
	n3 := printer{MyNode{Input: n2.Output}}

	errorCh := Wait(&n1, &n2, &n3)
	err := <-errorCh
	if err == nil || err.Error() != "modifier failed" {
		t.Errorf("first error with wrong msg: %s", err)
	}
	for err := range errorCh {
		t.Errorf("did not expect more errors: %s", err)
	}
	if !n1.wasAborted {
		t.Errorf("n1 was not aborted")
	}
	if n2.wasAborted {
		t.Errorf("n2 should have not been aborted")
	}
	if n3.wasAborted {
		t.Errorf("n3 should have not been aborted")
	}
}

/*
func TestSketchApi(t *testing.T) {
	SetDebug(true)

	n1 := generator{Output: make(chan int, 10)}
	n2 := modifier{Input: n1.Output, Output: make(chan int, 10)}
	n3 := printer{Input: n2.Input}

	for err := range Wait(&n1, &n2, &n3) {
		t.Errorf("did not expect error: %s", err)
	}
}
*/

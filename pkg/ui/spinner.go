package ui

import (
	"context"
	"fmt"
	"time"
)

// Spinner displays an animated spinner until Stop() is called.
type Spinner struct {
	stop chan struct{}
}

func StartSpinner(message string) *Spinner {
	return StartSpinnerWithContext(context.Background(), message)
}

// StartSpinnerWithContext starts a spinner that cancels with ctx.
func StartSpinnerWithContext(ctx context.Context, message string) *Spinner {
	s := &Spinner{stop: make(chan struct{})}
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Print("\r\033[K")
				return
			case <-ctx.Done():
				fmt.Print("\r\033[K")
				return
			default:
				fmt.Printf("\r%s %s", Dim(frames[i]), Dim(message))
				i = (i + 1) % len(frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop stops the spinner and clears its line from the terminal.
func (s *Spinner) Stop() {
	close(s.stop)
}

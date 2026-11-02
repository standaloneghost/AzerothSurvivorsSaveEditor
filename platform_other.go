//go:build !windows

package main

import (
	"fmt"
	"os"
)

// Used only for testing: a file named by GAME_RUNNING_FLAG stands in for the game.
func gameRunning() bool {
	f := os.Getenv("GAME_RUNNING_FLAG")
	if f == "" {
		return false
	}
	_, err := os.Stat(f)
	return err == nil
}

func openBrowser(string) {}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

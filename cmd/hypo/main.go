package main

import (
	"fmt"
	"os"

	"github.com/rtcall/hypo/cpu"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s file\n", os.Args[0])
		return
	}

	buf, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	c, err := cpu.New(buf)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	for c.State() {
		if err := c.Step(); err != nil {
			fmt.Printf("fatal: %s\n\n", err)
			c.WriteTrace(os.Stdout)
			break
		}
	}
}

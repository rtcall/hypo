package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rtcall/hypo/asm"
)

func main() {
	outPath := flag.String("o", "out", "output path")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Printf("usage: %s [-o path] file\n", os.Args[0])
		os.Exit(1)
	}

	inPath := flag.Arg(0)

	f, err := os.OpenFile(*outPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	f.Truncate(0)

	in, err := os.Open(inPath)
	if err != nil {
		f.Close()
		os.Remove(*outPath)
		panic(err)
	}

	defer in.Close()

	_, err = asm.Gen(in, f, os.Stderr)
	if err != nil {
		f.Close()
		os.Remove(*outPath)
		fmt.Printf("%s: %s\n", inPath, err)
		os.Exit(1)
	}
}

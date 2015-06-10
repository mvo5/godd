package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cheggaaa/pb"
)

type ddOpts struct {
	src string
	dst string
}

func parseArgs(args []string) (*ddOpts, error) {
	// support:
	//   # godd in-file out-file
	// for the lazy people
	if len(args) == 2 &&
		!strings.Contains(args[0], "=") &&
		!strings.Contains(args[1], "=") {
		return &ddOpts{src: args[0], dst: args[1]}, nil
	}

	// ok, real work
	opts := ddOpts{}
	for _, arg := range args {
		l := strings.SplitN(arg, "=", 2)
		switch l[0] {
		case "if":
			opts.src = l[1]
		case "of":
			opts.dst = l[1]
		default:
			return nil, fmt.Errorf("unknown argument %q", arg)
		}
	}

	return &opts, nil
}

func dd(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		dst.Sync()
		dst.Close()
	}()

	stat, err := src.Stat()
	if err != nil {
		return err
	}
	pbar := pb.New64(stat.Size()).SetUnits(pb.U_BYTES)
	pbar.Start()
	mw := io.MultiWriter(dst, pbar)
	_, err = io.Copy(mw, src)
	return err
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse args %v", err))
		os.Exit(1)
	}

	if err := dd(opts.src, opts.dst); err != nil {
		fmt.Println(fmt.Errorf("failed to dd %v", err))
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cheggaaa/pb"
)

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
	if err := dd(os.Args[1], os.Args[2]); err != nil {
		fmt.Println(fmt.Errorf("failed to dd %v", err))
		os.Exit(1)
	}
}

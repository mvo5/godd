package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cheggaaa/pb"
)

// var to allow tests to change it
var defaultBufSize = 4 * 1024 * 1024

// go does not offer support to customize the buffer size for
// io.Copy directly, so we need to implement a custom type with:
// ReadFrom and Write
type FixedBuffer struct {
	w   io.Writer
	buf []byte
}

func NewFixedBuffer(w io.Writer, size int) *FixedBuffer {
	return &FixedBuffer{
		w:   w,
		buf: make([]byte, size),
	}
}

func (f *FixedBuffer) ReadFrom(r io.Reader) (int, error) {
	return r.Read(f.buf)
}

func (f *FixedBuffer) Write(data []byte) (int, error) {
	return f.w.Write(data)
}

// the dd releated stuff
type ddOpts struct {
	src string
	dst string
	bs  int
}

func ddAtoi(s string) (int, error) {
	if len(s) < 2 {
		return strconv.Atoi(s)
	}

	// dd supports suffixes via two chars like "kB"
	fac := 1
	switch s[len(s)-2:] {
	case "kB":
		fac = 1000
	case "MB":
		fac = 1000 * 1000
	case "GB":
		fac = 1000 * 1000 * 1000
	case "TB":
		fac = 1000 * 1000 * 1000 * 1000
	}
	// adjust string if its from xB group
	if fac%10 == 0 {
		s = s[:len(s)-2]
	}

	// check for single char digests
	switch s[len(s)-1] {
	case 'b':
		fac = 512
	case 'K':
		fac = 1024
	case 'M':
		fac = 1024 * 1024
	case 'G':
		fac = 1024 * 1024 * 1024
	case 'T':
		fac = 1024 * 1024 * 1024 * 1024
	}
	// ajust string if its from the X group
	if fac%512 == 0 {
		s = s[:len(s)-1]
	}

	n, err := strconv.Atoi(s)
	n *= fac
	return n, err
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
	opts := ddOpts{
		bs: defaultBufSize,
	}
	for _, arg := range args {
		l := strings.SplitN(arg, "=", 2)
		switch l[0] {
		case "if":
			opts.src = l[1]
		case "of":
			opts.dst = l[1]
		case "bs":
			bs, err := ddAtoi(l[1])
			if err != nil {
				return nil, err
			}
			opts.bs = bs
		default:
			return nil, fmt.Errorf("unknown argument %q", arg)
		}
	}

	return &opts, nil
}

var mountinfoPath = "/proc/self/mountinfo"

func sanityCheckDst(dstPath string) error {
	// see https://www.kernel.org/doc/Documentation/filesystems/proc.txt,
	// sec. 3.5
	f, err := os.Open(mountinfoPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	// resolve any symlink to the target
	resolvedDstPath, err := filepath.EvalSymlinks(dstPath)
	if err == nil {
		dstPath = resolvedDstPath
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := strings.Fields(scanner.Text())
		if len(l) == 0 {
			continue
		}
		mountPoint := l[4]
		mountSrc := l[9]

		// resolve any symlinks in mountSrc
		resolvedMountSrc, err := filepath.EvalSymlinks(mountSrc)
		if err == nil {
			mountSrc = resolvedMountSrc
		}

		if strings.HasPrefix(mountSrc, dstPath) {
			return fmt.Errorf("%s is mounted on %s", mountSrc, mountPoint)
		}
	}

	return scanner.Err()
}

func dd(srcPath, dstPath string, bs int) error {
	if bs == 0 {
		bs = defaultBufSize
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := sanityCheckDst(dstPath); err != nil {
		return err
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		dst.Sync()
		dst.Close()
	}()

	// huge default bufsize
	w := NewFixedBuffer(dst, bs)

	stat, err := src.Stat()
	if err != nil {
		return err
	}
	pbar := pb.New64(stat.Size()).SetUnits(pb.U_BYTES)
	pbar.Start()
	mw := io.MultiWriter(w, pbar)
	_, err = io.Copy(mw, src)
	return err
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse args %v", err))
		os.Exit(1)
	}

	if err := dd(opts.src, opts.dst, opts.bs); err != nil {
		fmt.Println(fmt.Errorf("failed to dd: %v", err))
		os.Exit(1)
	}
}

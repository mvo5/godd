package main

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/danielrh/go-xz"
	"github.com/mvo5/godd/udev"
)

var (
	// var to allow tests to change it
	defaultBufSize = int64(4 * 1024 * 1024)

	Stdin  = os.Stdin
	Stdout = os.Stdout
)

// go does not offer support to customize the buffer size for
// io.Copy directly, so we need to implement a custom type with:
// ReadFrom and Write
type FixedBuffer struct {
	w   io.Writer
	buf []byte
}

func NewFixedBuffer(w io.Writer, size int64) *FixedBuffer {
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

type compType uint8

const (
	compNone compType = 1 << iota
	compGzip
	compBzip2
	compXz
	compAuto compType = 0
)

func (c compType) String() string {
	switch c {
	case compNone:
		return "none"
	case compGzip:
		return "gzip"
	case compBzip2:
		return "bzip2"
	case compXz:
		return "xz"
	case compAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// the dd releated stuff
type ddOpts struct {
	src string
	dst string
	bs  int64

	comp compType
}

func ddComp(s string) (compType, error) {
	switch s {
	case "auto":
		return compAuto, nil
	case "none":
		return compNone, nil
	case "gz", "gzip":
		return compGzip, nil
	case "bz2", "bzip2":
		return compBzip2, nil
	case "xz":
		return compXz, nil
	default:
		return compAuto, fmt.Errorf("unknown compression type %q", s)
	}
}

func ddAtoi(s string) (int64, error) {
	if len(s) < 2 {
		return strconv.ParseInt(s, 10, 64)
	}

	// dd supports suffixes via two chars like "kB"
	fac := int64(1)
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

	n, err := strconv.ParseInt(s, 10, 64)
	n *= fac
	return n, err
}

func findNonCdromRemovableDeviceFiles() (res []string) {
	c := udev.New(nil)
	for _, d := range c.QueryBySubsystem("block") {
		if d.GetSysfsAttr("removable") == "1" && d.GetProperty("ID_CDROM") != "1" {
			res = append(res, d.GetDeviceFile())
		}
	}

	return res
}

func parseArgs(args []string) (*ddOpts, error) {

	// support: auto-detect removable devices
	if len(args) == 1 {
		fmt.Printf(`
No target selected, detected the following removable device:
  %s

`, strings.Join(findNonCdromRemovableDeviceFiles(), "\n  "))
		return nil, fmt.Errorf("please select target device")
	}

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
		case "comp":
			comp, err := ddComp(l[1])
			if err != nil {
				return nil, err
			}
			opts.comp = comp
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

func guessComp(src string) compType {
	switch filepath.Ext(src) {
	case ".gz":
		return compGzip
	case ".bz2":
		return compBzip2
	case ".xz":
		return compXz
	default:
		return compNone
	}
}

func (dd *ddOpts) open() (io.ReadCloser, error) {
	if dd.src == "-" {
		return Stdin, nil
	}

	r, err := os.Open(dd.src)
	if err != nil {
		return nil, err
	}

	comp := dd.comp
	if comp == compAuto {
		comp = guessComp(dd.src)
	}

	switch comp {
	case compNone:
		return r, nil
	case compGzip:
		return gzip.NewReader(r)
	case compBzip2:
		return ioutil.NopCloser(bzip2.NewReader(r)), nil
	case compXz:
		cr := xz.NewDecompressionReadCloser(r)
		return &cr, nil
	}

	panic("can't happen")
}

func (dd *ddOpts) create() (*os.File, error) {
	if dd.dst == "-" {
		return Stdout, nil
	}
	return os.Create(dd.dst)
}

func (dd *ddOpts) Run() error {
	if dd.bs == 0 {
		dd.bs = defaultBufSize
	}

	src, err := dd.open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err := sanityCheckDst(dd.dst); err != nil {
		return err
	}

	dst, err := dd.create()
	if err != nil {
		return err
	}
	defer func() {
		dst.Sync()
		dst.Close()
	}()

	// huge default bufsize
	w := NewFixedBuffer(dst, dd.bs)

	var pbar *pb.ProgressBar
	switch src := src.(type) {
	case *os.File:
		stat, err := src.Stat()
		if err != nil {
			return err
		}
		pbar = pb.New64(stat.Size()).SetUnits(pb.U_BYTES)
	default:
		pbar = pb.New64(0).SetUnits(pb.U_BYTES)
	}
	pbar.Start()
	mw := io.MultiWriter(w, pbar)
	_, err = io.Copy(mw, src)

	return err
}

func main() {
	dd, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse args: %v", err))
		os.Exit(1)
	}

	if err := dd.Run(); err != nil {
		fmt.Println(fmt.Errorf("failed to dd: %v", err))
		os.Exit(1)
	}
}

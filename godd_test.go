package main

import (
	"io/ioutil"
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type GoddTestSuite struct {
	cwd string
}

var _ = Suite(&GoddTestSuite{})

func (g *GoddTestSuite) SetUpTests(c *C) {
	var err error
	g.cwd, err = os.Getwd()
	c.Assert(err, IsNil)
	os.Chdir(c.MkDir())
}

func (g *GoddTestSuite) TearDownTests(c *C) {
	os.Chdir(g.cwd)
}

func (g *GoddTestSuite) TestSimple(c *C) {
	// ensure we test with tiny buffer
	defaultBufSize = 2

	canary := []byte("foo bar")
	err := ioutil.WriteFile("src", canary, 0644)
	c.Assert(err, IsNil)

	err = dd("src", "dst")
	c.Assert(err, IsNil)

	read, err := ioutil.ReadFile("dst")
	c.Assert(err, IsNil)
	c.Assert(read, DeepEquals, canary)
}

func (g *GoddTestSuite) TestParseTrivial(c *C) {
	opts, err := parseArgs([]string{"src", "dst"})
	c.Assert(err, IsNil)
	c.Check(opts.src, Equals, "src")
	c.Check(opts.dst, Equals, "dst")
}

func (g *GoddTestSuite) TestParseIfOf(c *C) {
	opts, err := parseArgs([]string{"if=src", "of=dst"})
	c.Assert(err, IsNil)
	c.Check(opts.src, Equals, "src")
	c.Check(opts.dst, Equals, "dst")
}

func (g *GoddTestSuite) TestParseError(c *C) {
	opts, err := parseArgs([]string{"if=src", "invalid=command"})
	c.Assert(err, ErrorMatches, `unknown argument "invalid=command"`)
	c.Assert(opts, IsNil)
}

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
	canary := []byte("foo bar")
	err := ioutil.WriteFile("src", canary, 0644)
	c.Assert(err, IsNil)

	err = dd("src", "dst")
	c.Assert(err, IsNil)

	read, err := ioutil.ReadFile("dst")
	c.Assert(err, IsNil)
	c.Assert(read, DeepEquals, canary)
}

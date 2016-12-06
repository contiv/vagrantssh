// +build vagrant

package remotessh

import (
	"os"
	"strings"
	"sync"
	. "testing"

	. "gopkg.in/check.v1"

	log "github.com/Sirupsen/logrus"
)

type vagrantTestSuite struct {
	vagrant Testbed
}

var _ = Suite(&vagrantTestSuite{})

func TestVagrant(t *T) {
	if os.Getenv("HOST_TEST") != "" {
		os.Exit(0)
	}

	log.SetLevel(log.DebugLevel)

	TestingT(t)
}

func (v *vagrantTestSuite) SetUpSuite(c *C) {
	vagrant := &Vagrant{}
	c.Assert(vagrant.Setup(false, []string{}, 3), IsNil)
	v.vagrant = vagrant
}

func (v *vagrantTestSuite) TestRunEnv(c *C) {
	vagrant := &Vagrant{}
	c.Assert(vagrant.Setup(false, []string{"MYENV=foo"}, 3), IsNil)
	outChan := make(chan string, 3)

	c.Assert(vagrant.IterateNodes(func(node TestbedNode) error {
		out, err := node.RunCommandWithOutput("echo $MYENV")
		outChan <- out
		return err
	}), IsNil)

	for x := 0; x < 3; x++ {
		c.Assert(strings.TrimSpace(<-outChan), Equals, "foo")
	}
}

func (v *vagrantTestSuite) TestSetupInvalidArgs(c *C) {
	vagrant := &Vagrant{}
	c.Assert(vagrant.Setup(1, "foo"), ErrorMatches, "Unexpected args to Setup.*Expected:.*Received:.*")
}

func (v *vagrantTestSuite) TestRun(c *C) {
	for _, node := range v.vagrant.GetNodes() {
		c.Assert(node.RunCommand("ls"), IsNil)
	}

	for _, node := range v.vagrant.GetNodes() {
		c.Assert(node.RunCommand("exit 1"), NotNil)
	}
}

func (v *vagrantTestSuite) TestRunWithOutput(c *C) {
	for _, node := range v.vagrant.GetNodes() {
		out, err := node.RunCommandWithOutput("whoami")
		c.Assert(err, IsNil)
		c.Assert(strings.TrimSpace(out), Equals, "vagrant")
	}

	for _, node := range v.vagrant.GetNodes() {
		_, err := node.RunCommandWithOutput("exit 1")
		c.Assert(err, NotNil)
	}
}

func (v *vagrantTestSuite) TestIterateNodes(c *C) {
	mutex := &sync.Mutex{}
	var i int
	c.Assert(v.vagrant.IterateNodes(func(node TestbedNode) error {
		mutex.Lock()
		i++
		mutex.Unlock()
		return node.RunCommand("exit 0")
	}), IsNil)

	c.Assert(i, Equals, 3)

	i = 0
	c.Assert(v.vagrant.IterateNodes(func(node TestbedNode) error {
		mutex.Lock()
		i++
		mutex.Unlock()
		return node.RunCommand("exit 1")
	}), NotNil)
	c.Assert(i, Equals, 3)
}

func (v *vagrantTestSuite) TestScpFileFromRemoteNodes(c *C) {
	for _, node := range v.vagrant.GetNodes() {
		localFilename := "./"+node.GetName()+"_test_local"
		remoteFilename := "test_remote_to_local"

		_, err := node.RunCommandWithOutput("echo `pwd` > " + remoteFilename)
		c.Assert(err, IsNil)

		err = node.ScpFromRemoteToLocal(remoteFilename, localFilename)
		c.Assert(err, IsNil)

		// verify local file was created
		_, err = os.Stat(localFilename)
		c.Assert(err, IsNil)

		// cleanup
		os.Remove(localFilename)
		node.RunCommand("rm " + remoteFilename)
	}
}

func (v *vagrantTestSuite) TestScpFileToRemoteNodes(c *C) {
	for _, node := range v.vagrant.GetNodes() {
		localFilename := "test_local"
		remoteFilename := "test_local_to_remote"

		f, err := os.Create(localFilename)
		_, err = f.WriteString("I am testing scp")

		err = node.ScpFromLocalToRemote(localFilename, remoteFilename)
		c.Assert(err, IsNil)

		// verify scp created a file on remote node
		out, err := node.RunCommandWithOutput("ls")
		c.Assert(err, IsNil)
		if !strings.Contains(out, remoteFilename) {
			c.Errorf("Output of ls on remote node: %s", out)
		}

		// cleanup
		os.Remove(localFilename)
		node.RunCommand("rm " + remoteFilename)
	}
}

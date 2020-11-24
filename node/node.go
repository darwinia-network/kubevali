package node

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/darwinia-network/kubevali/config"
	"github.com/sirupsen/logrus"
)

type Node struct {
	Config config.Node

	Cmd    *exec.Cmd
	Stdout io.Reader
	Stderr io.Reader
}

func NewNode(conf config.Node) *Node {
	cmd := exec.Command(conf.Command[0], conf.Command[1:]...)
	cmd.Stdin = os.Stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	node := &Node{
		Cmd:    cmd,
		Stdout: stdout,
		Stderr: stderr,
	}

	return node
}

func (n *Node) Run() error {
	logrus.Infof("Starting node with: %s", n.ShellCommand())
	return n.Cmd.Run()
}

func (n *Node) ShellCommand() string {
	var shcmd strings.Builder

	shcmd.WriteString(strconv.Quote(n.Cmd.Path))

	for _, a := range n.Cmd.Args[1:] {
		shcmd.WriteString(" ")
		shcmd.WriteString(strconv.Quote(a))
	}

	return shcmd.String()
}

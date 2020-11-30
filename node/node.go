package node

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

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
		logrus.Fatalf("Unable to pipe stdout: %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.Fatalf("Unable to pipe stderr: %s", err)
	}

	node := &Node{
		Cmd:    cmd,
		Stdout: stdout,
		Stderr: stderr,
	}

	return node
}

func (n *Node) Run(ctx context.Context) error {
	if err := n.Cmd.Start(); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		n.Cmd.Process.Signal(syscall.SIGTERM)
	}()

	return n.Cmd.Wait()
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

package node

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/darwinia-network/kubevali/config"
	"github.com/sirupsen/logrus"
)

type Node struct {
	Cmd    *exec.Cmd
	Stdout io.Reader
	Stderr io.Reader
}

func validateConfig(conf config.Config) error {
	if len := len(conf.NodeTemplate.Command); len < 1 {
		return fmt.Errorf("Config nodeTemplate.Command[] should at least have 1 element, got %d", len)
	}

	return nil
}

func renderOrDie(tpl *template.Template, text string) string {
	t := template.Must(tpl.Clone())
	t = template.Must(t.New("").Parse(text))

	var buf strings.Builder
	if err := t.Execute(&buf, nil); err != nil {
		panic(err)
	}

	return buf.String()
}

func renderCmd(conf config.Config) []string {
	tpl := template.New("")
	config.InitTemplateFuncMap(tpl)
	tpl = template.Must(tpl.Parse(conf.CommonTemplate))

	var cmd []string

	for _, value := range conf.NodeTemplate.Command {
		v := renderOrDie(tpl, value)
		cmd = append(cmd, v)
	}

	for key, value := range conf.NodeTemplate.Args {
		a := fmt.Sprintf("--%s", key)
		v := renderOrDie(tpl, value)
		cmd = append(cmd, a, v)
	}

	return cmd
}

func NewNode(conf config.Config) *Node {
	if err := validateConfig(conf); err != nil {
		panic(err)
	}

	cmdStr := renderCmd(conf)
	cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
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

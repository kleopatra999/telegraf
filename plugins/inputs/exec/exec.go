package exec

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"

	"github.com/gonuts/go-shellquote"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/parsers"
)

const sampleConfig = `
  ### Commands array
  commands = ["/tmp/test.sh", "/usr/bin/mycollector --foo=bar"]

  ### measurement name suffix (for separating different commands)
  name_suffix = "_mycollector"

  ### Data format to consume. This can be "json", "influx" or "graphite" (line-protocol)
  ### Each data format has it's own unique set of configuration options, read
  ### more about them here:
  ### https://github.com/influxdata/telegraf/blob/master/DATA_FORMATS.md
  data_format = "json"
`

type Exec struct {
	Commands []string
	Command  string

	// Data Format Arguments:
	DataFormat string
	Separator  string
	Templates  []string
	TagKeys    []string
	parser     parsers.Parser

	wg sync.WaitGroup
	sync.Mutex

	runner Runner
	errc   chan error
}

type Runner interface {
	Run(*Exec, string) ([]byte, error)
}

type CommandRunner struct{}

func (c CommandRunner) Run(e *Exec, command string) ([]byte, error) {
	split_cmd, err := shellquote.Split(command)
	if err != nil || len(split_cmd) == 0 {
		return nil, fmt.Errorf("exec: unable to parse command, %s", err)
	}

	cmd := exec.Command(split_cmd[0], split_cmd[1:]...)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec: %s for command '%s'", err, command)
	}

	return out.Bytes(), nil
}

func NewExec() *Exec {
	return &Exec{runner: CommandRunner{}}
}

func (e *Exec) ProcessCommand(command string, acc telegraf.Accumulator) {
	defer e.wg.Done()

	out, err := e.runner.Run(e, command)
	if err != nil {
		e.errc <- err
		return
	}

	metrics, err := e.parser.Parse(out)
	if err != nil {
		e.errc <- err
	} else {
		for _, metric := range metrics {
			acc.AddFields(metric.Name(), metric.Fields(), metric.Tags(), metric.Time())
		}
	}
}

func (e *Exec) initConfig() error {
	e.Lock()
	defer e.Unlock()

	if e.Command != "" && len(e.Commands) < 1 {
		e.Commands = []string{e.Command}
	}
	if e.DataFormat == "" {
		e.DataFormat = "json"
	}

	var err error
	e.parser, err = parsers.NewParser(&parsers.Config{
		DataFormat: e.DataFormat,
		Separator:  e.Separator,
		Templates:  e.Templates,
		TagKeys:    e.TagKeys,
		MetricName: "exec",
	})

	if err != nil {
		return fmt.Errorf("exec configuration error: %s ", err.Error())
	}

	return nil
}

func (e *Exec) SampleConfig() string {
	return sampleConfig
}

func (e *Exec) Description() string {
	return "Read metrics from one or more commands that can output to stdout"
}

func (e *Exec) Gather(acc telegraf.Accumulator) error {
	if e.parser == nil {
		if err := e.initConfig(); err != nil {
			return err
		}
	}

	e.Lock()
	e.errc = make(chan error, 10)
	e.Unlock()

	for _, command := range e.Commands {
		e.wg.Add(1)
		go e.ProcessCommand(command, acc)
	}
	e.wg.Wait()

	select {
	default:
		close(e.errc)
		return nil
	case err := <-e.errc:
		close(e.errc)
		return err
	}

}

func init() {
	inputs.Add("exec", func() telegraf.Input {
		return NewExec()
	})
}

package scaffold

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"integration-testing/log"
)

type Scenario struct {
	info    string
	scripts []*Script
}

func NewScenario(info string) *Scenario {
	return &Scenario{info: info}
}

func (scen *Scenario) AddScript(script *Script) {
	scen.scripts = append(scen.scripts, script)
}

func (scen *Scenario) String() string {
	return scen.info
}

func (scen *Scenario) Scripts() []*Script {
	return scen.scripts
}

func LoadScens(path string) ([]*Scenario, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	var script *Script
	var scen *Scenario
	var scens []*Scenario
	var line string
	for {
		if line, err = readLine(reader); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		log.Debug2f("read script '%s'", line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "Scenario") {
			if scen != nil {
				scens = append(scens, scen)
			}
			scen = NewScenario(line)
		} else if scen != nil {
			if script, err = ParseScript(line); err != nil {
				return nil, err
			}
			scen.AddScript(script)
		} else {
			return nil, fmt.Errorf("not start with scenario")
		}
		line = ""
	}
	if scen != nil {
		scens = append(scens, scen)
	}
	return scens, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	var line string
	for {
		part, prefix, err := reader.ReadLine()
		if err != nil {
			return "", err
		}
		line += string(part)
		if !prefix {
			break
		}
	}
	line = strings.Trim(line, " ")
	return line, nil
}

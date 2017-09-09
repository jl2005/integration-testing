package scaffold

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"integration-testing/log"
)

const (
	New          = "new"
	Call         = "call"
	ParallelCall = "pcall"
)

type Script struct {
	src string

	command       string
	className     string
	method        string
	objectIdRegex string
	param         *Param
}

func ParseScript(str string) (*Script, error) {
	if len(str) == 0 {
		return nil, fmt.Errorf("empty str for parse")
	}
	var i, j int
	token := strings.Split(strings.Trim(str, " \n\r"), " ")
	s := &Script{
		src:     str,
		command: token[0],
		param:   newParam(),
	}
	switch s.command {
	case New:
		s.className = token[1]
		i = 2
	case Call:
		fallthrough
	case ParallelCall:
		s.objectIdRegex = token[1]
		s.method = token[2]
		i = 3
	default:
		return nil, fmt.Errorf("unsport action '%s'", token[0])
	}
	for i < len(token) {
		if !strings.HasPrefix(token[i], "-") {
			return nil, fmt.Errorf("param '%s' not start with '-'", token[i])
		}
		for j = i + 1; j < len(token); j++ {
			if strings.HasPrefix(token[j], "-") {
				break
			}
		}
		key := strings.Trim(token[i], "-")
		val := strings.Trim(strings.Join(token[i+1:j], " "), "'")
		s.param.Add(key, val)
		i = j
	}
	return s, nil
}

func (s *Script) Command() string {
	return s.command
}

func (s *Script) ClassName() string {
	return s.className
}

func (s *Script) Match(name string) bool {
	if matched, err := regexp.MatchString(s.objectIdRegex, name); err != nil {
		log.Errorf("script match error regex=%s name=%s err=%s",
			s.objectIdRegex, name, err)
		return false
	} else if matched {
		return true
	}
	return false
}

func (s *Script) String() string {
	return s.src
}

func (s *Script) Method() string {
	return s.method
}

func (s *Script) Param() *Param {
	return s.param
}

type Param struct {
	m     map[string]string
	mutex sync.RWMutex
}

func newParam() *Param {
	return &Param{
		m: make(map[string]string),
	}
}

func (p *Param) Add(key, val string) {
	p.mutex.Lock()
	p.m[key] = val
	p.mutex.Unlock()
}

func (p *Param) GetString(key string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if val, ok := p.m[key]; ok {
		return val
	}
	return ""
}

func (p *Param) GetDuration(key string) time.Duration {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if val, ok := p.m[key]; ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		} else {
			log.Debug2f("parse durtion failed. %s=%s", key, val)
		}
	}
	return 0
}

func (p *Param) GetInt(key string) int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if val, ok := p.m[key]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func (p *Param) GetBool(key string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if val, ok := p.m[key]; ok && val == "true" {
		return true
	}
	return false
}

func (p *Param) String() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if data, err := json.Marshal(p.m); err == nil {
		return string(data)
	}
	return ""
}

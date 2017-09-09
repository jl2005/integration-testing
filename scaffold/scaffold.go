package scaffold

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"
	"integration-testing/log"
)

type Config interface{}

//TODO add usage -->统一接口调用规则，通过函数文档说明
type NewFunc func(context.Context, int, *Param) (Role, error)
type LoadConfFunc func(file string) (Config, error)

var (
	newFuncs      = make(map[string]NewFunc)
	descriptions  = make(map[string]string)
	newFuncsMutex sync.Mutex

	confs      = make(map[string]LoadConfFunc)
	confsMutex sync.Mutex
)

func Register(name string, f NewFunc, desc ...string) {
	newFuncsMutex.Lock()
	defer newFuncsMutex.Unlock()
	newFuncs[name] = f
	if len(desc) > 0 {
		descriptions[name] = desc[0]
	} else {
		descriptions[name] = ""
	}
}

func RegisterConf(name string, f LoadConfFunc) {
	confsMutex.Lock()
	defer confsMutex.Unlock()
	confs[name] = f
}

func GetNewFunc(name string) (NewFunc, error) {
	if f, ok := newFuncs[name]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("not found new func for '%s'", name)
}

func ObjectsDesc() map[string]string {
	m := make(map[string]string)
	newFuncsMutex.Lock()
	defer newFuncsMutex.Unlock()
	for k, v := range descriptions {
		m[k] = v
	}
	return m
}

func Run(scens []*Scenario) {
	for _, scen := range scens {
		ctl := NewControl()
		if err := ctl.Run(scen); err != nil {
			log.Errorf(" |- run failed %s", err)
			log.Errorf("run scenario failed %s %s", scen.String(), err)
			return
		}
	}
}

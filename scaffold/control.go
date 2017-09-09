package scaffold

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/context"
	"integration-testing/log"
)

type Role interface {
	Id() string
	Run(ctx context.Context, methodName string, p *Param) error
	Stop()
}

type Control struct {
	roles []Role
	mutex sync.Mutex

	wg      sync.WaitGroup
	errChan chan error

	ctx    context.Context
	cancel context.CancelFunc
}

func NewControl() *Control {
	ctl := &Control{
		errChan: make(chan error, 1),
	}
	ctl.ctx, ctl.cancel = context.WithCancel(context.Background())
	return ctl
}

func (ctl *Control) Run(scen *Scenario) error {
	defer ctl.Stop()
	var err error

	//TODO create scen time context
	log.Infof(scen.String())
	scripts := scen.Scripts()
	for i, script := range scripts {
		log.Infof("#%d %s", i, script.String())
		if err = ctl.RunScript(script); err != nil {
			return err
		}
	}
	ch := make(chan interface{})
	go func() {
		ctl.wg.Wait()
		close(ch)
	}()
	select {
	case <-ctl.ctx.Done():
		return nil
	case <-ch:
		return nil
	case err = <-ctl.errChan:
		return err
	}
	return nil
}

func (ctl *Control) RunScript(script *Script) error {
	//TODO change this time from config
	tCtx, c := context.WithTimeout(ctl.ctx, 3*time.Second)
	defer c()
	switch script.Command() {
	case New:
		var err error
		var newFunc NewFunc
		var num int
		var role Role
		if newFunc, err = GetNewFunc(script.ClassName()); err != nil {
			return err
		}
		if num = script.Param().GetInt("num"); num == 0 {
			num = 1
		}
		for i := 0; i < num; i++ {
			if role, err = newFunc(ctl.ctx, i, script.Param()); err != nil {
				return err
			}
			ctl.Add(role)
		}
	case Call:
		for _, r := range ctl.roles {
			if script.Match(r.Id()) {
				if err := r.Run(tCtx, script.Method(), script.Param()); err != nil {
					return err
				}
			}
		}
	case ParallelCall:
		//TODO 同一个角色的异步调用，应该进行排队
		for _, r := range ctl.roles {
			if script.Match(r.Id()) {
				ctl.wg.Add(1)
				go func(r Role, script *Script) {
					if err := r.Run(ctl.ctx, script.Method(), script.Param()); err != nil {
						select {
						case ctl.errChan <- err:
						default:
						}
					}
					ctl.wg.Done()
				}(r, script)
			}
		}
	default:
		return fmt.Errorf("not find command '%s'", script.Command())
	}
	return nil
}

func (ctl *Control) Add(role Role) {
	ctl.mutex.Lock()
	ctl.roles = append(ctl.roles, role)
	ctl.mutex.Unlock()
	log.Debug1f(" |- add role %s", role.Id())
}

func (ctl *Control) Roles() []Role {
	return ctl.roles
}

func (ctl *Control) Stop() {
	if ctl.cancel != nil {
		ctl.cancel()
	}
	for _, r := range ctl.roles {
		r.Stop()
	}
}

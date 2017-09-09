package bifrost

import (
	"fmt"

	log "gitlab.meitu.com/gocommons/logbunny"
)

type FakeLog struct {
}

func (l *FakeLog) Debug(v ...interface{}) {
	log.Debug(fmt.Sprint(v...))
}

func (l *FakeLog) Info(v ...interface{}) {
	log.Info(fmt.Sprint(v...))
}

func (l *FakeLog) Error(v ...interface{}) {
	log.Error(fmt.Sprint(v...))
}

func (l *FakeLog) Warn(v ...interface{}) {
	log.Warn(fmt.Sprint(v...))
}

func (l *FakeLog) Fatal(v ...interface{}) {
	log.Fatal(fmt.Sprint(v...))

}

func (l *FakeLog) Debugf(format string, v ...interface{}) {
	log.Debug(fmt.Sprintf(format, v...))
}

func (l *FakeLog) Infof(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))

}

func (l *FakeLog) Errorf(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))

}

func (l *FakeLog) Warnf(format string, v ...interface{}) {
	log.Warn(fmt.Sprintf(format, v...))

}

func (l *FakeLog) Fatalf(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func (l *FakeLog) SetAdditionalStackDepth(depth int) {
}

func (l *FakeLog) Flush() {
}

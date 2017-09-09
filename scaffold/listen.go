package scaffold

import (
	"net"
	"time"
)

type Listener struct {
	l     net.Listener
	ready bool
}

func Listen(t, laddr string) (*Listener, error) {
	l, err := net.Listen(t, laddr)
	if err != nil {
		return nil, err
	}
	return &Listener{l: l}, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.l.Accept()
	l.ready = true
	return c, err
}

func (l *Listener) Close() error {
	return l.l.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.l.Addr()
}

func (l *Listener) WaitConnect() {
	for {
		if l.ready {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

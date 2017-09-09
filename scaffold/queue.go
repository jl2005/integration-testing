package scaffold

import (
	"sync/atomic"

	log "gitlab.meitu.com/gocommons/logbunny"
	"golang.org/x/net/context"
)

type Queue interface {
	Push(val interface{})
	Pop(ctx context.Context) interface{}
	Size() int32
	Stop()
}

type StorQueue struct {
	ch   chan interface{}
	size int32

	isStat bool

	ctx    context.Context
	cancel context.CancelFunc
}

const chLen = 4

func NewStorQueue(isStat bool) Queue {
	ctx, cancel := context.WithCancel(context.Background())
	sq := &StorQueue{
		ch:   make(chan interface{}, chLen),
		size: 0,

		isStat: isStat,

		ctx:    ctx,
		cancel: cancel,
	}
	return sq
}

func (sq *StorQueue) Push(val interface{}) {
	log.Debug("push into queue", log.Object("val", val))
	atomic.AddInt32(&sq.size, 1)
	if sq.isStat {
		return
	}
	select {
	case sq.ch <- val:
		return
	default:
		//TODO 这里可能会起很多的goroutine，后期再优化一下
		go func() {
			select {
			case <-sq.ctx.Done():
				return
			case sq.ch <- val:
			}
		}()
	}

}

func (sq *StorQueue) Pop(ctx context.Context) interface{} {
	if sq.isStat {
		return nil
	}
	select {
	case <-ctx.Done():
		return nil
	case <-sq.ctx.Done():
		return nil
	case val := <-sq.ch:
		return val
	}
}

func (sq *StorQueue) Size() int32 {
	return sq.size
}

func (sq *StorQueue) Stop() {
	sq.cancel()
}

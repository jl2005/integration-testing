package bifrost

import (
	"fmt"
	"time"

	"gitlab.meitu.com/platform/bifrost/grpc/conn"
	"gitlab.meitu.com/platform/bifrost/grpc/push"
	"gitlab.meitu.com/platform/go-radar/radar"
	"golang.org/x/net/context"
	//	"google.golang.org/grpc"
	"integration-testing/log"
	sca "integration-testing/scaffold"
)

func init() {
	sca.Register("connd", NewConnd, "fake connd object")
}

type Connd struct {
	conf *Config
	id   string
	Addr string

	listen *sca.Listener
	reg    radar.Registration

	pushd push.PushServiceClient

	packetid uint64

	queue    sca.Queue
	msgQueue sca.Queue

	ctx    context.Context
	cancel context.CancelFunc
}

func NewConnd(ctx context.Context, index int, param *sca.Param) (sca.Role, error) {
	//TODO
	//	c := conf.(*Config)
	//var err error
	ctx, cancel := context.WithCancel(context.Background())
	connd := &Connd{
		//		conf:     c,
		id: fmt.Sprintf("%s%d", param.GetString("id"), index),
		//Addr:     c.ConndGrpcAddr,
		queue:    sca.NewStorQueue(false),
		msgQueue: sca.NewStorQueue(true),
		ctx:      ctx,
		cancel:   cancel,
	}
	/*

		//1.1 start grpc server
		connd.listen, err = sca.Listen("tcp", c.ConndGrpcAddr)
		if err != nil {
			return nil, err
		}
		gs := grpc.NewServer()
		conn.RegisterConnServiceServer(gs, connd)
		go func() {
			if err := gs.Serve(connd.listen); err != nil {
				log.Errorf("connd grpc serve error %s", err)
			}
		}()

		//1.2 注册地址
		cli, err := radar.NewClient(c.Etcd.Cluster, "", "")
		if err != nil {
			return nil, err
		}
		connd.reg, err = cli.CreateRegistration(c.ConndService)
		if err != nil {
			return nil, err
		}
		connd.reg.Register(c.ConndGrpcAddr, "connd")

		//2. connect to pushd
		connect, err := grpc.Dial(c.PushdAddr, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		connd.pushd = push.NewPushServiceClient(connect)

	*/
	go connd.run()
	return connd, nil
}

func (c *Connd) Id() string {
	return c.id
}

func (c *Connd) Run(ctx context.Context, method string, p *sca.Param) error {
	log.Debug1f(" |- connd run %s", c.Id(), method, p.String())
	switch method {
	case "sub":
		return c.Sub(ctx, p)
	case "recvn":
		return c.Recvn(ctx, p)
	default:
		return fmt.Errorf("connd not suport method '%s'", method)
	}
	return nil
}

func (c *Connd) Stop() {
	if c.reg != nil {
		c.reg.Deregister()
	}
	c.cancel()
}

func (c *Connd) Sub(ctx context.Context, p *sca.Param) error {
	topic := p.GetString("topic")
	clientid := p.GetString("clientid")
	qos := p.GetInt("qos")
	req := &push.SubscribeReq{
		ClientID:     clientid,
		Cookie:       nil,
		Topics:       []string{topic},
		Qoss:         []uint32{uint32(qos)},
		ConndAddress: c.Addr,
		TraceID:      "****",
		Service:      "test",
	}
	if resp, err := c.pushd.Subscribe(ctx, req); err != nil {
		return err
	} else {
		c.msgQueue.Push(resp)
	}
	return nil
}

func (c *Connd) Recvn(ctx context.Context, p *sca.Param) error {
	num := p.GetInt("num")
	tCtx, cancel := context.WithTimeout(ctx, c.conf.WaitTimeout)
	defer cancel()
	var n int
	for {
		select {
		case <-tCtx.Done():
			return fmt.Errorf("recvn want %d get %d", num, n)
		default:
		}
		n = int(c.msgQueue.Size())
		if n >= num {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (c *Connd) Disconnect(ctx context.Context, req *conn.DisconnectReq) (*conn.DisconnectResp, error) {
	return &conn.DisconnectResp{Code: 0}, nil
}

func (c *Connd) Notify(ctx context.Context, nr *conn.NotifyReq) (*conn.NotifyResp, error) {
	//TODO get notify
	c.queue.Push(nr)
	return &conn.NotifyResp{Code: 0}, nil
}

func (c *Connd) run() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		nr := c.queue.Pop(c.ctx)
		if nr == nil {
			return
		}
		if err := c.pull(nr.(*conn.NotifyReq)); err != nil {
			log.Errorf("pull msg error %s", err)
			return
		}
	}
}

func (c *Connd) pull(nr *conn.NotifyReq) error {
	req := &push.PullReq{
		Topic:        nr.Topic,
		Index:        nr.Index,
		PacketID:     c.packetid,
		Count:        1, //TODO 使用参数控制
		CleanSession: true,
	}
	for i := 0; i < len(nr.ClientIDs); i++ {
		req.ClientID = nr.ClientIDs[i]
		req.Qos = nr.Qoss[i]
		if resp, err := c.pushd.Pull(c.ctx, req); err != nil {
			return err
		} else {
			c.packetid = resp.PacketID
			c.msgQueue.Push(resp)
		}
	}
	return nil
}

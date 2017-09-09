package bifrost

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.meitu.com/platform/bifrost/grpc/conn"
	"gitlab.meitu.com/platform/bifrost/grpc/publish"
	pb "gitlab.meitu.com/platform/bifrost/grpc/push"
	"gitlab.meitu.com/platform/tardis-go/tardis"
	"gitlab.meitu.com/platform/tardis-go/tardis/register"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"integration-testing/log"
	sca "integration-testing/scaffold"
)

func init() {
	sca.Register("pushd", NewPushd, "fake pushd object")
}

type Pushd struct {
	conf *Config

	id string

	listen *sca.Listener
	server *tardis.GrpcServer

	addr   string
	client conn.ConnServiceClient

	ch chan *pb.Message
}

func NewPushd(ctx context.Context, index int, param *sca.Param) (sca.Role, error) {
	pushd := &Pushd{
		//	conf: conf.(*Config),
		id: fmt.Sprintf("%s%d", param.GetString("id"), index),
		ch: make(chan *pb.Message, 8),
	}

	// connect to connd
	c, err := grpc.Dial(pushd.conf.ConndGrpcAddr, grpc.WithInsecure(), grpc.WithBackoffMaxDelay(3*time.Second))
	if err != nil {
		return nil, err
	}
	pushd.client = conn.NewConnServiceClient(c)

	// start pushd and publish service
	pushd.listen, err = sca.Listen("tcp", pushd.conf.PushdAddr)
	if err != nil {
		return nil, err
	}

	r, err := register.NewEtcdRegister(pushd.conf.Etcd.Cluster, pushd.conf.Etcd.Username, pushd.conf.Etcd.Password)
	if err != nil {
		return nil, err
	}
	pushd.server = tardis.NewGrpcServer("", "", pushd.conf.PushdRegion, tardis.WithRegisters(r), tardis.IsAutoRegister(false))
	pushd.server.RegHandler(pb.RegisterPushServiceServer, pushd)
	pushd.server.RegHandler(publish.RegisterPublishServiceServer, pushd)

	err = pushd.server.RegisterService(pushd.conf.PushdPushdName, pushd.conf.PushdPushdGroup, pushd.conf.PushdAddr)
	if err != nil {
		return nil, fmt.Errorf("register pushd pushd service failed %s", err)
	}
	err = pushd.server.RegisterService(pushd.conf.PushdPublishName, pushd.conf.PushdPublishGroup, pushd.conf.PushdAddr)
	if err != nil {
		return nil, fmt.Errorf("register pushd publish service failed %s", err)
	}

	go func() {
		if err := pushd.server.Serve(pushd.listen); err != nil {
			log.Errorf("pushd server error %s", err)
		}
	}()

	log.Debug2f("wait connd connect")
	pushd.listen.WaitConnect()
	return pushd, nil
}

func (p *Pushd) Id() string {
	return p.id
}

func (pushd *Pushd) Run(ctx context.Context, method string, p *sca.Param) error {
	log.Debug1f(" |- pushd run %s %s %s", pushd.Id(), method, p.String())
	switch method {
	case "notify":
		return pushd.Notify(ctx, p)
	case "gen":
		return pushd.Gen(ctx, p)
	default:
		return fmt.Errorf("pushd not suport method '%s'", method)
	}
	return nil
}

func (pushd *Pushd) Stop() {
	pushd.listen.Close()
	//TODO close pushd.client
}

func (pushd *Pushd) Notify(ctx context.Context, p *sca.Param) error {
	topic := p.GetString("topic")
	clientid := p.GetString("clientid")
	qos := p.GetInt("qos")
	num := p.GetInt("num")

	req := &conn.NotifyReq{
		Topic:     topic,
		ClientIDs: []string{clientid},
		Qoss:      []uint32{uint32(qos)},
		Index:     1,
	}

	for i := 1; i <= num; i++ {
		req.Index = uint64(i)
		resp, err := pushd.client.Notify(context.Background(), req)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			log.Errorf("notify result code isn't zero code=%d", resp.Code)
		}
	}
	return nil
}

func (pushd *Pushd) Gen(ctx context.Context, p *sca.Param) error {
	topic := p.GetString("topic")
	size := p.GetInt("size")
	num := p.GetInt("num")
	qos := p.GetInt("qos")

	rand.Seed(time.Now().UnixNano())
	const maxSize = 64 * 1024
	buf := genData(maxSize)
	for i := 1; i <= num; i++ {
		msg := &pb.Message{
			Topic:     topic,
			Qos:       uint32(qos),
			MessageID: uint64(i),
			Payload:   buf[:size],
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("pushd want gen data %d, cur %d", num, i)
		case pushd.ch <- msg:
		}
	}
	return nil
}

func genData(n int) []byte {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = uint8('a' + rand.Intn(26))
	}
	return buf
}

func (pushd *Pushd) Pull(ctx context.Context, req *pb.PullReq) (*pb.PullResp, error) {
	select {
	case msg := <-pushd.ch:
		resp := &pb.PullResp{
			Code:     0,
			Index:    msg.MessageID, //TODO 这里应该是一个递增的数字
			Messages: []*pb.Message{msg},
		}
		return resp, nil
	default:
		resp := &pb.PullResp{
			Code:     0,
			Index:    0,
			Messages: nil,
		}
		return resp, nil
	}
	return nil, nil
}

func (pushd *Pushd) Connect(ctx context.Context, req *pb.ConnectReq) (*pb.ConnectResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.ConnectResp{Code: 0}, nil
}
func (pushd *Pushd) Disconnect(ctx context.Context, req *pb.DisconnectReq) (*pb.DisconnectResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.DisconnectResp{Code: 0}, nil
}
func (pushd *Pushd) Puback(ctx context.Context, req *pb.PubackReq) (*pb.PubackResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.PubackResp{Code: 0}, nil
}
func (pushd *Pushd) Pubrec(ctx context.Context, req *pb.PubrecReq) (*pb.PubrecResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.PubrecResp{Code: 0}, nil
}
func (pushd *Pushd) Pubrel(ctx context.Context, req *pb.PubrelReq) (*pb.PubrelResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.PubrelResp{Code: 0}, nil
}
func (pushd *Pushd) Pubcomp(ctx context.Context, req *pb.PubcompReq) (*pb.PubcompResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.PubcompResp{Code: 0}, nil
}
func (pushd *Pushd) Publish(ctx context.Context, req *pb.PublishReq) (*pb.PublishResp, error) {
	//time.Sleep(time.Duration(s.opt.delay) * time.Millisecond)
	return &pb.PublishResp{Code: 0}, nil
}
func (pushd *Pushd) Subscribe(ctx context.Context, req *pb.SubscribeReq) (*pb.SubscribeResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	stored := make([]bool, len(req.Topics))
	for i := range stored {
		stored[i] = true
	}
	return &pb.SubscribeResp{Code: 0, Stored: stored}, nil
}

func (pushd *Pushd) Unsubscribe(ctx context.Context, req *pb.UnsubscribeReq) (*pb.UnsubscribeResp, error) {
	//time.Sleep(time.Duration(delay) * time.Millisecond)
	return &pb.UnsubscribeResp{Code: 0}, nil
}

func (pushd *Pushd) ResumeSession(ctx context.Context, req *pb.ResumeSessionReq) (*pb.ResumeSessionResp, error) {
	return &pb.ResumeSessionResp{Code: 0}, nil
}

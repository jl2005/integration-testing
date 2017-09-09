package bifrost

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pb "gitlab.meitu.com/platform/bifrost/grpc/callback"
	publish "gitlab.meitu.com/platform/bifrost/grpc/publish"
	"gitlab.meitu.com/platform/go-radar/radar"
	"gitlab.meitu.com/platform/tardis-go/tardis"
	"gitlab.meitu.com/platform/tardis-go/tardis/ctxsys"
	"gitlab.meitu.com/platform/tardis-go/tardis/register"
	"gitlab.meitu.com/platform/tardis-go/tardis/utils"
	"golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"integration-testing/log"
	sca "integration-testing/scaffold"
)

func init() {
	sca.Register("callback", NewCallback, "callback object")
}

type Callback struct {
	conf *Config

	id string

	Addr   string
	listen *sca.Listener
	reg    radar.Registration

	pushd publish.PublishServiceClient
}

func NewCallback(ctx context.Context, index int, param *sca.Param) (sca.Role, error) {
	//TODO get config
	//	c := conf.(*Config)
	callback := &Callback{
		//		conf: c,
		id: fmt.Sprintf("%s%d", param.GetString("id"), index),
		//Addr: c.CallbackAddr,
	}
	if err := callback.init(); err != nil {
		return nil, err
	}
	return callback, nil
}

//TODO 当失败的时候需要回收一些资源
func (cb *Callback) init() error {
	if err := cb.connectPushd(); err != nil {
		return err
	}

	if err := cb.registerCallback(); err != nil {
		return err
	}
	return nil
}

func (cb *Callback) connectPushd() error {
	conn, err := grpc.Dial(cb.conf.PushdAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	cb.pushd = publish.NewPublishServiceClient(conn)
	return nil
}

func (cb *Callback) registerCallback() error {
	var err error
	if cb.listen, err = sca.Listen("tcp", cb.Addr); err != nil {
		return fmt.Errorf("callback listen failed %s", err)
	}

	//TODO 将这些变成配置
	fnames := "OnConnect,OnDisconnect,OnSubscribe,OnPublish,OnOffline,OnUnsubscribe,PostSubscribe"
	value := struct {
		ServiceName string   `json:"service-name"`
		Funcs       []string `json:"callback"`
	}{
		ServiceName: cb.conf.CallbackServiceName,
		Funcs:       strings.Split(fnames, ","),
	}
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}

	etcd, err := register.NewEtcdRegister(cb.conf.Etcd.Cluster, "", "")
	if err != nil {
		return err
	}

	var opts []tardis.ServiceOption
	opts = append(opts, tardis.WithRegisterNodeMsg(string(val)))
	opts = append(opts, tardis.WithRegisters(etcd))

	gs := tardis.NewGrpcServer(cb.conf.CallbackName, cb.conf.CallbackGroup, cb.conf.CallbackRegion, opts...)
	gs.RegHandler(pb.RegisterOnConnectServer, cb)
	gs.RegHandler(pb.RegisterOnDisconnectServer, cb)
	gs.RegHandler(pb.RegisterOnOfflineServer, cb)
	gs.RegHandler(pb.RegisterOnPublishServer, cb)
	gs.RegHandler(pb.RegisterOnSubscribeServer, cb)
	gs.RegHandler(pb.RegisterOnUnsubscribeServer, cb)
	gs.RegHandler(pb.RegisterPostSubscribeServer, cb)

	go func() {
		if err = gs.Serve(cb.listen); err != nil {
			log.Debug2f("callback serve failed %s", err)
		}
	}()
	log.Debug1f("wait pushd connect callback")
	cb.listen.WaitConnect()
	return nil
}

func (cb *Callback) Id() string {
	return cb.id
}

func (cb *Callback) Run(ctx context.Context, method string, param *sca.Param) error {
	log.Debug1f(" |- callback run %s %s %s", cb.Id(), method, param.String())
	switch method {
	case "onconnect":
	case "onsub":
	case "onpub":
	case "pub":
		return cb.Publish(ctx, param)
	default:
		return fmt.Errorf("callback not suport method '%s'", method)
	}
	return nil
}

func (cb *Callback) Stop() {
	if cb.reg != nil {
		cb.reg.Deregister()
	}
	if cb.listen != nil {
		cb.listen.Close()
		time.Sleep(10 * time.Millisecond)
	}
}

func (cb *Callback) OnConnect(ctx context.Context, req *pb.OnConnectRequest) (*pb.OnConnectReply, error) {
	//	log.Debug("OnConnect", log.Object("req", req))
	return &pb.OnConnectReply{Code: pb.ErrCode_ErrOK, Cookie: []byte("connect")}, nil
}

func (cb *Callback) OnDisconnect(ctx context.Context, req *pb.OnDisconnectRequest) (*pb.OnDisconnectReply, error) {
	//log.Debug("OnDisconnect", log.Object("req", req))
	return &pb.OnDisconnectReply{Code: pb.ErrCode_ErrOK}, nil
}

func (cb *Callback) OnSubscribe(ctx context.Context, req *pb.OnSubscribeRequest) (*pb.OnSubscribeReply, error) {
	//log.Debug("OnSubscribe", log.Object("req", req))
	var success []bool = make([]bool, len(req.SubTopics))
	for i := 0; i < len(success); i++ {
		success[i] = true
	}

	return &pb.OnSubscribeReply{
		Code:      pb.ErrCode_ErrOK,
		Successes: success,
		Cookie:    []byte("subscribe"),
	}, nil
}

func (cb *Callback) PostSubscribe(ctx context.Context, req *pb.PostSubscribeRequest) (*pb.PostSubscribeReply, error) {
	//log.Debug("PostSubscribe", log.Object("req", req))
	return &pb.PostSubscribeReply{Code: pb.ErrCode_ErrOK}, nil
}
func (cb *Callback) OnUnsubscribe(ctx context.Context, req *pb.OnUnsubscribeRequest) (*pb.OnUnsubscribeReply, error) {
	//log.Debug("OnUnsubscribe", log.Object("req", req))
	return &pb.OnUnsubscribeReply{Code: pb.ErrCode_ErrOK}, nil
}
func (cb *Callback) OnPublish(ctx context.Context, req *pb.OnPublishRequest) (*pb.OnPublishReply, error) {
	//log.Debug("OnPublish", log.Object("req", req))
	cb.publish(ctx, req)
	return &pb.OnPublishReply{Code: pb.ErrCode_ErrSkip, Cookie: []byte("publish")}, nil
}
func (cb *Callback) OnOffline(ctx context.Context, req *pb.OnOfflineRequest) (*pb.OnOfflineReply, error) {
	//log.Debug("OnOffline", log.Object("req", req))
	return &pb.OnOfflineReply{Code: pb.ErrCode_ErrOK}, nil
}

func (cb *Callback) publish(ctx context.Context, req *pb.OnPublishRequest) {
	target := &publish.Target{
		Topic:    req.Topic,
		Address:  "",
		ClientID: "",
		QoS:      req.QoS,
	}
	pubReq := &publish.PublishRequest{
		MessageID: []byte{1},
		Payload:   req.Message,
		Targets:   []*publish.Target{target},
		//cookie
	}
	_, err := cb.pushd.Publish(ctx, pubReq)
	log.Debug2f(" callback pubulish %v", req)
	if err != nil {
		log.Errorf("publish error %s", err)
	}
}

//Publish $msg to $topic $qos in $num
func (cb *Callback) Publish(ctx context.Context, param *sca.Param) error {
	topic := param.GetString("topic")
	num := param.GetInt("num")
	msg := param.GetString("msg")
	qos := param.GetInt("qos")
	sleep := param.GetDuration("sleep")

	topics := strings.Split(topic, "|")
	var targets []*publish.Target
	for _, t := range topics {
		targets = append(targets, &publish.Target{
			Topic:    t,
			Address:  "",
			ClientID: "",
			QoS:      int32(qos),
		})
	}
	pubReq := &publish.PublishRequest{
		MessageID: []byte{1},
		Payload:   []byte(msg),
		Targets:   targets,
		//cookie
	}

	for i := 1; i <= num; i++ {
		pubReq.MessageID[0] = byte(i)
		tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		id := fmt.Sprintf("%s-%d", utils.TraceID(), i)
		_, err := cb.pushd.Publish(ctxsys.WithTraceID(tCtx, id), pubReq)
		log.Debug2f("publish  msg '%s' to topic '%s' traceid %s", msg, topic, id)
		if err != nil {
			cancel()
			log.Errorf("publish failed id=%s %s", id, err)
			return err
		}
		cancel()
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
	return nil
}

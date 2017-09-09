package bifrost

import (
	"fmt"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/net/context"
	"integration-testing/log"
	sca "integration-testing/scaffold"
)

func init() {
	sca.Register("user", NewUser, "fake user object")
}

type User struct {
	conf *Config
	ctx  context.Context

	index    int
	Clientid string
	Username string
	password string

	client MQTT.Client

	isStat   bool
	queueMap map[string]sca.Queue
	mu       sync.Mutex
}

func NewUser(ctx context.Context, index int, param *sca.Param) (sca.Role, error) {
	fs := &flag.FlagSet{}
	id := fs.String("id", "user", "user id prefix")
	username := fs.String("username", "name", "user name for client")
	password := fs.String("password", "password", "password for client")
	stat := fs.Bool("stat", false, "status")
	fs.Parse(args)
	//TODO
	//	c := conf.(*Config)
	user := &User{
		//		conf: c,
		ctx: ctx,

		index:    index,
		Clientid: fmt.Sprintf("%s%d", param.GetString("id"), index),
		Username: param.GetString("username"),
		password: param.GetString("password"),

		isStat:   param.GetBool("stat"),
		queueMap: make(map[string]sca.Queue),
	}
	if err := user.connect(); err != nil {
		return nil, err
	}
	return user, nil
}

func (user *User) Run(ctx context.Context, method string, param *sca.Param) error {
	log.Debug1f(" |- user run %s %s %s", user.Id(), method, param.String())
	switch method {
	case "sub":
		return user.Sub(ctx, param)
	case "unsub":
		return user.Unsub(ctx, param)
	case "pub":
		return user.Pub(ctx, param)
	case "recv":
		return user.Recv(ctx, param)
	case "recvn":
		return user.RecvN(ctx, param)
	case "stat":
		return user.Stat(ctx, param)
	default:
		return fmt.Errorf("user not suport method '%s'", method)
	}
	return nil
}

func (user *User) Id() string {
	return user.Clientid
}

func (user *User) connect() error {
	opts := MQTT.NewClientOptions()
	opts.SetProtocolVersion(4)
	opts.AddBroker(user.conf.ConndAddr)
	opts.SetClientID(user.Clientid)
	opts.SetUsername(user.Username)
	opts.SetPassword(user.password)
	opts.SetConnectTimeout(user.conf.WaitTimeout)
	opts.SetDefaultPublishHandler(user.messageHandle)
	//opts.SetCleanSession(false)
	//TODO set ssl connection

	user.client = MQTT.NewClient(opts)
	token := user.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (user *User) messageHandle(client MQTT.Client, msg MQTT.Message) {
	user.put(msg)
}

func (user *User) Sub(ctx context.Context, param *sca.Param) error {
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}
	if len(topic) == 0 {
		return fmt.Errorf("sub topic is empty")
	}

	token := user.client.Subscribe(topic, 1, user.messageHandle)
	if token.WaitTimeout(user.conf.WaitTimeout) {
		if token.Error() != nil {
			return token.Error()
		}
	} else {
		return fmt.Errorf("user sub topic '%s' timeout", topic)
	}
	user.mu.Lock()
	if q, exist := user.queueMap[topic]; !exist {
		//不关注重复订阅的问题
		q = sca.NewStorQueue(user.isStat)
		user.queueMap[topic] = q
	}
	user.mu.Unlock()
	return nil
}

func (user *User) Unsub(ctx context.Context, param *sca.Param) error {
	//TODO 这个是否应该清空一下queue中的统计
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}
	if len(topic) == 0 {
		return fmt.Errorf("unsub topic is empty")
	}

	//TODO 将topic拆分成多个topic
	token := user.client.Unsubscribe(topic)
	if token.WaitTimeout(user.conf.WaitTimeout) {
		if token.Error() != nil {
			return token.Error()
		}
	} else {
		return fmt.Errorf("user sub topic '%s' timeout", topic)
	}
	user.mu.Lock()
	if q, exist := user.queueMap[topic]; exist {
		q.Stop()
		delete(user.queueMap, topic)
	}
	user.mu.Unlock()
	return nil
}

func (user *User) replaceVar(s string) (string, error) {
	if len(s) == 0 || !strings.Contains(s, "$") {
		return s, nil
	}
	s = strings.Replace(s, "$clientid", user.Clientid, -1)
	s = strings.Replace(s, "$index", fmt.Sprintf("%d", user.index), -1)
	if strings.Contains(s, "$") {
		return "", fmt.Errorf("some variable not replace '%s'", s)
	}
	return s, nil
}

// Pub msg '$msg' to topic '$topic'
func (user *User) Pub(ctx context.Context, param *sca.Param) error {
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}
	if len(topic) == 0 {
		return fmt.Errorf("pub topic is empty")
	}

	token := user.client.Publish(topic, 1, false, param.GetString("msg"))
	if token.WaitTimeout(user.conf.WaitTimeout) {
		if token.Error() != nil {
			return token.Error()
		}
	} else {
		return fmt.Errorf("user sub topic '%s' timeout", topic)
	}
	return nil
}

//Recv recv $msg from $topic
func (user *User) Recv(ctx context.Context, param *sca.Param) error {
	tCtx, cancel := context.WithTimeout(ctx, user.conf.WaitTimeout)
	defer cancel()
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}

	msg := user.get(tCtx, topic)
	if msg == nil {
		return fmt.Errorf("not recv msg from topic %s", topic)
	}

	if string(msg.Payload()) != param.GetString("msg") {
		return fmt.Errorf("recv msg from topic '%s' not match. want '%s' get '%s'",
			param.GetString("topic"), param.GetString("msg"), msg.Payload())
	}
	return nil
}

//RecvN recv $msg from $topic $num
func (user *User) RecvN(ctx context.Context, param *sca.Param) error {
	topic := param.GetString("topic")
	num := param.GetInt("num")
	if len(topic) == 0 {
		return fmt.Errorf("RecvN topic is empty")
	}
	if num == 0 {
		return fmt.Errorf("RecvN not set num param")
	}
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}
	tCtx, cancel := context.WithTimeout(ctx, user.conf.WaitTimeout)
	defer cancel()
	if err := user.waitN(tCtx, topic, num); err != nil {
		return err
	}
	return nil
}

//Stat $topic -num $num
func (user *User) Stat(ctx context.Context, param *sca.Param) error {
	topic := param.GetString("topic")
	if len(topic) == 0 {
		return fmt.Errorf("Stat topic is empty")
	}
	topic, err := user.replaceVar(param.GetString("topic"))
	if err != nil {
		return err
	}
	expect := param.GetInt("num")
	var cur, num int
	t := time.NewTimer(1 * time.Second)
	for num < expect {
		num = user.getSize(topic)
		t.Reset(1 * time.Second)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			log.Infof("%s recv msg %d in 1s, total %d", user.Clientid, num-cur, num)
		}
		cur = num
	}
	return nil
}

func (user *User) put(msg MQTT.Message) {
	user.mu.Lock()
	q, exist := user.queueMap[msg.Topic()]
	if !exist {
		q = sca.NewStorQueue(user.isStat)
		user.queueMap[msg.Topic()] = q
	}
	q.Push(msg)
	user.mu.Unlock()
}

func (user *User) get(ctx context.Context, topic string) MQTT.Message {
	user.mu.Lock()
	q, exist := user.queueMap[topic]
	user.mu.Unlock()
	if !exist {
		return nil
	}
	i := q.Pop(ctx)
	if i == nil {
		return nil
	}
	if msg, ok := i.(MQTT.Message); ok {
		return msg
	}
	return nil
}

const waitTime = 10 * time.Millisecond

func (user *User) waitN(ctx context.Context, topic string, n int) error {
	var num int
	user.mu.Lock()
	q, exist := user.queueMap[topic]
	user.mu.Unlock()
	if !exist {
		return fmt.Errorf("not sub topic %s", topic)
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waitn for %s want %d get %d", topic, n, num)
		default:
		}
		num = int(q.Size())
		if num >= n {
			return nil
		}
		time.Sleep(waitTime)
	}
	return nil

}

func (user *User) getSize(key string) int {
	user.mu.Lock()
	q, exist := user.queueMap[key]
	user.mu.Unlock()
	if !exist {
		return 0
	}
	return int(q.Size())
}

func (user *User) Stop() {
	user.client.Disconnect(10)
	user.mu.Lock()
	for _, q := range user.queueMap {
		q.Stop()
	}
	user.mu.Unlock()
}

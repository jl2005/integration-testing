package bifrost

import (
	"time"

	"github.com/shafreeck/configo"
	sca "integration-testing/scaffold"
)

func init() {
	sca.RegisterConf("bifrost", func(file string) (sca.Config, error) {
		return LoadConf(file)
	})
}

type Config struct {
	ScriptTimeout time.Duration `cfg:"script-timeout; 10s; ; wait script timeout"`
	WaitTimeout   time.Duration `cfg:"wait-timeout; 1s; ; wait one script timeout"`

	ConndAddr     string `cfg:"connd-addr; tcp://127.0.0.1:1883; ; connd address"`
	ConndGrpcAddr string `cfg:"connd-grpc-addr; 127.0.0.1:50051; ; connd grpc address"`
	ConndService  string `cfg:"connd-service; connd; ; connd service name"`

	PushdAddr         string `cfg:"pushd-addr; 127.0.0.1:50054; ; pushd address"`
	PushdPublishName  string `cfg:"pushd-publish-name; test_publish; ; pushd publish service name"`
	PushdPublishGroup string `cfg:"pushd-publish-group; test_publish; ; pushd publish group"`
	PushdPushdName    string `cfg:"pushd-pushd-name; test_pushd; ; pushd pushdservice name"`
	PushdPushdGroup   string `cfg:"pushd-pushd-group; test_pushd; ; pushd pushd group"`
	PushdRegion       string `cfg:"pushd-region; local; ; pushd region"`

	CallbackAddr        string `cfg:"callback-addr; 127.0.0.1:24321; ; callback address"`
	CallbackName        string `cfg:"callback-name; test_callback; ; callback service name"`
	CallbackGroup       string `cfg:"callback-group; test_group; ; callback group"`
	CallbackRegion      string `cfg:"callback-region; local; ; callback region"`
	CallbackServiceName string `cfg:"callback-servicename; pushd-live; ; callback service name"`

	Etcd Etcd `cfg:"etcd"`
}

type Etcd struct {
	Cluster  []string `cfg:"cluster; required; url; addresses of etcd cluster"`
	Username string   `cfg:"username;;; username of etcd"`
	Password string   `cfg:"password;;; password of the user"`
}

func LoadConf(file string) (*Config, error) {
	conf := new(Config)
	if err := configo.Load(file, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

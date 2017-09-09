`integration-testing` 是一个用于做集成测试的框架，它具有简单易于上手。

## 命令格式

```
Usage of ./integration-testing:
  -c string
        specify the config file (default "conf.toml")
  -d string
        script data file (default "data")
  -i    interactive model
  -v    debug message
  -vv
        more debug message
```

为了更加详细的获取调用信息，这里增加了`-vv`级别的日志输出，以便获取更加详细的信息。

## 脚本格式

其主要由`new` `call` `pcall`三个基础命令组成。

1. new

```
new [class name] -id [object id prefix]  [-num n]
```

创建类对应的对象，需要通过`-id`指定对象的前缀，这个前缀用来区分类的不同对象。可以用num指定创建多个对象。

**PS:**在创建对象的时候，框架会在id后添加一个从0开始的递增数字用于区分不同的多个对象。

2. call

```
call [object id regex] [method name] [params]
```

通过id匹配对象，如果匹配则调用其方法。


3. pcall

```
pcall [object id regex] [method name] [params]
```

异步调用对象的方法，`pcall`会对调用生成一个`goroutine`。

## 如何接入框架

如果想扩展框架，只需要做以下两步

1. 实现Role接口

```
type Role interface {
    Id() string
    Run(ctx context.Context, method string, p *Param) error
    Stop()
}
```
* `Id()`用来获取类的id，此id将作为调用时的正则匹配
* `Run()`作为调用的入口，需要通过method区分不同方法的调用，param是对调用中的参数的封装。

2. 注册类创建函数

```
func Register(name string, f NewFunc)
```

需要注册创建函数，创建函数的格式如下：

```
type NewFunc func(context.Context, int, Config, *Param) (Role, error)
```

2. 注册配置文件生成函数和类创建函数

```
func RegisterConf(name string, f LoadConfFunc)
```

在交互模式下，需要先读取配置文件，此时需要注册对应的配置文件创建接口。配置文件读取接口格式如下：

```
type LoadConfFunc func(file string) (Config, error)
```


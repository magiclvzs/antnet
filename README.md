# antnet
A game server framework in Golang    一个Golang游戏服务器框架     
有疑问可联系qq：441707528   
## 第一次使用
   在介绍antnet之前，我们先使用一次，看看antnet如何构建一个echo服务器
```
package main

import (
	"antnet"
)

func main() {
	antnet.StartServer("tcp://:6666", antnet.MsgTypeCmd, &antnet.EchoMsgHandler{}, nil)
	antnet.WaitForSystemExit()
}
```
通过上面的代码我们就实现了一个最简单的echo服务器。   
现在打开命令行，执行telent 127.0.0.1 6666，输入一个字符串，回车后你将收到原样的回复消息。    

## 代码组织    
antnet尽可能把功能相关的代码组织到一块，让你能快速找到代码，比如parser打头的文件表示解析器相关，msgque打头的文件表示消息队列相关。    

## 依赖项
github.com/golang/protobuf   
github.com/vmihailenco/msgpack   
github.com/go-redis/redis   

## 消息头
对于一个网络服务器，我们首先需要定义的是消息头，antnet的消息头长度为12个字节，定义如下
```
type MessageHead struct {
	Len   uint32 //数据长度
	Error uint16 //错误码
	Cmd   uint8  //命令
	Act   uint8  //动作
	Index uint16 //序号
	Flags uint16 //标记
}
```
错误码用于快速返回一个错误，往往服务器返回错误时是不需要跟任何数据的。  
其中cmd和act标识了消息的用途，cmd一般按照大功能分类，比如建筑模块，士兵模块，而act则是这些模块下面的活动，比如升级，建造等。  
index用于标识唯一的一次请求，客户端应该自增方式增加index，服务器会原样返回，这样就让客户端可以唯一标识每次请求。  
cmd和act，index共同组成了一个消息的tag，服务器在返回时往往需要原样返回一个消息的tag。  
flags是一个消息的选项，比如消息是否压缩，是否加密等等。   

## 消息
```
type Message struct {
	Head       *MessageHead //消息头，可能为nil
	Data       []byte       //消息数据
	IMsgParser              //解析器
	User       interface{}  //用户自定义数据
}
```
消息是对一次交互的抽象，每个消息都会用自己的解析器，有消息头部分（可能为nil），和数据部分。  

## 消息队列
antnet将数据流抽象为消息队列，无论是来自tcp，udp还是websocket的数据流，都会被抽象为消息队列。    
根据是否带有消息头，antnet将消息队列分为两种类型：   
```
type MsgType int

const (
	MsgTypeMsg MsgType = iota //消息基于确定的消息头
	MsgTypeCmd                //消息没有消息头，以\n分割
)  
```

## 解析器
antnet目前有三种解析器类型：  
```
type ParserType int

const (
	ParserTypePB   ParserType = iota //protobuf类型，用于和客户端交互
	ParserTypeJson                   //json类型，可以用于客户端或者服务器之间交互
	ParserTypeCmd                    //cmd类型，类似telnet指令，用于直接和程序交互
)   
```
这三种类型的解析器，都可以用antnet.Parser来创建。    
每个解析器需要一个Type字段和一个ErrType字段定义，Type字段表示了消息解析器的类型，而ErrType字段则决定了消息解析失败之后默认的行为,ErrType目前有4中方式：    
```
type ParseErrType int

const (
	ParseErrTypeSendRemind ParseErrType = iota //消息解析失败发送提醒消息
	ParseErrTypeContinue                       //消息解析失败则跳过本条消息
	ParseErrTypeAlways                         //消息解析失败依然处理
	ParseErrTypeClose                          //消息解析失败则关闭连接
)
```

默认的解析器Type是pb类型的，而错误处理是一旦解析出错给客户端发送提示消息。    
比如我们现在有一个需求是根据玩家id获取玩家等级，那么我们可以建立一个cmd类型的解析器，这样我们就能直接通过telent连接到服务器进行查询了，使用如下代码
创建一个cmd类型的解析器。
```
pf := &antnet.Parser{Type: antnet.ParserTypeCmd}
```
上面的代码就定义了一个基于cmd模式的解析器。    

定义好解析之后，就需要注册解析器需要解析的消息，解析器支持两种模式：  
1. 基于MsgTypeMsg的，根据cmd和act进行解析，支持上面三种类型，使用Register进行注册。
2. 基于MsgTypeCmd的，可以支持ParserTypeCmd和ParserTypeCmd类型，这种消息往往没有消息头，使用RegisterMsg进行注册。  

#### 命令行解析器
命令行解析器用于解析命令行输入，类似telnet，我希望但服务器运行起来之后有一个非常简单的交流接口，直接基于telnet是最好了，而这个解析器就是为此准备，他可以接收不完整的输入，只要你最终输入完整即可，也可以接收你错误的输入，直到你输入正确为止。    
命令行解析器目前支持两种tag：   
1. `match:"k"`表示只需要匹配字段名即可，为了减少输入的大小写切换，在匹配的时候会将字段名默认作为小写匹配。    
2. `match:"kv"`表示需要匹配字段名和字段值   
命令行解析器的注册需要一个结构体，比如上面例子，需要查询玩家等级的，我们的定义如下： 
```
type GetGamerLevel struct {
	Get   string `match:"k"`
	Gamer int
	Level int `match:"k"`
}
```
   
1. Get字段，表示方法，比如get，set，reload，这种情况我只需要输入方法即可，而tag `match:"k"`则表示只需要匹配字段名即可   
2. Gamer字段，表示玩家id，没有tag，对于没有tag的字段，解析器会认为需要匹配字段名和值，比如输入gamer 1 会被认为合法，而gamer test则不合法，因为test不是int   
3. Level字段，表示玩家等级，有tag，表示只需要匹配level这个字段名即可。    
定义好结构体之后我们需要注册到解析器，使用如下代码注册即可：   
```
pf.RegisterMsg(&GetGamerLevel{}, nil)
```
这样我们就把这个消息注册到了解析器
#### json解析器
json解析器用于解析json格式的数据，json解析器在MsgTypeCmd类型的消息队列中不接受不完整的输入。
#### protobuf解析器
protobuf解析器用于解析pb类型的数据

## 处理器
处理器用于处理消息，一个处理器应该实现IMsgHandler消息接口：
```
type IMsgHandler interface {
	OnNewMsgQue(msgque IMsgQue) bool                //新的消息队列
	OnDelMsgQueue(msgque IMsgQue)                   //消息队列关闭
	OnProcessMsg(msgque IMsgQue, msg *Message) bool //默认的消息处理函数
	OnConnectComplete(msgque IMsgQue, ok bool) bool //连接成功
	GetHandlerFunc(msg *Message) HandlerFunc        //根据消息获得处理函数
}
```

当然，一般情况，我们并不需要完全实现上面的接口，你只需在你的处理器里面添加antnet.DefMsgHandler定义即可。
在antnet.DefMsgHandler里面，同样定义了Register和RegisterMsg函数，原理和解析器一样，也是为了区分不同的输入。   
如果你没有注册任何消息处理函数，系统会自动调用OnProcessMsg函数，如果你有定义的话。    
在上面根据玩家id获取玩家等级的例子中，我们这样定义处理器：   
```
type Handler struct {
	antnet.DefMsgHandler
}
```
定义好处理器之后我们需要创建处理器以及注册要处理的消息以和理函数：    
```
h := &Handler{}
h.RegisterMsg(&GetGamerLevel{}, func(msgque antnet.IMsgQue, msg *antnet.Message) bool {
	c2s := msg.C2S().(*GetGamerLevel)
	c2s.Level = 8
	msgque.SendStringLn(msg.C2SString())
	return true
})
```
这样我们就创建了一个处理器以及注册处理函数

#### 处理器的调用时机
antnet会为每个tcp链接建立两个goroutine进行服务一个用于读，一个用于写，处理的回调发生在每个链接的读的goroutine之上，为什么要这么设计，是考虑当客户端的一个消息没有处理完成的时候真的有必要立即处理下一个消息吗？  

## 启动服务
启动一个网络服务器使用antnet.StartServer函数，他被定义在msgque.go文件里面，一个服务目前需要一个处理器和一个解析器才可以运行。
在上面根据玩家id获取玩家等级的例子中，我们这样启动服务：   
```
antnet.StartServer("tcp://:6666", antnet.MsgTypeCmd, h, pf)
```
在服务启动后我们需要等待ctrl+C消息以结束服务，使用WaitForSystemExit()函数即可。    
完整示例： 
```
package main

import "antnet"

type GetGamerLevel struct {
	Get   string `match:"k"`
	Gamer int
	Level int `match:"k"`
}

type Handler struct {
	antnet.DefMsgHandler
}

func main() {
	pf := &antnet.Parser{Type: antnet.ParserTypeCmd}
	pf.RegisterMsg(&GetGamerLevel{}, nil)

	h := &Handler{}
	h.RegisterMsg(&GetGamerLevel{}, func(msgque antnet.IMsgQue, msg *antnet.Message) bool {
		c2s := msg.C2S().(*GetGamerLevel)
		c2s.Level = 8
		msgque.SendStringLn(msg.C2SString())
		return true
	})

	antnet.StartServer("tcp://:6666", antnet.MsgTypeCmd, h, pf)
	antnet.WaitForSystemExit()
}
```
在这个示例中，我们建立了一个基于命令的网络应用。  
现在打开命令行，执行telent 127.0.0.1 6666。  
输入字符串 get gamer 1 level，你将收到回复{"Get":"get","Gamer":1,"Level":8}  
上面的例子模拟了一个游戏服务器常见的需求，即命令行式的交互，这对于游戏后台的调试以及某些gm指令的执行非常有效。  

## 全局变量
为了方便使用antnet封装了一些全局变量：  
1. StartTick 用于标识antnet启动的时刻，是一个毫秒级的时间戳    
2. NowTick 用于标识antnet现在的时刻，是一个自动变化的毫秒级时间戳  
3. DefMsgQueTimeout 默认的网络超时，当超过这个时间和客户端没有交互，antnet将断开连接，默认是30s    
4. MaxMsgDataSize 默认的最大数据长度，超过这个长度的消息将会被拒绝并关闭连接，默认为1MB  

## 全局函数
为了方便使用antnet封装了一些全局函数以供调用：  
1. WaitForSystemExit 用于等待用户输入ctrl+C以结束进程。  
2. Go 用于创建可被antnet管理的goroutine   
2. Go2 同Go，不同的是会有个默认的channel，以通知antnet的结束    
3. Stop 结束antnet   
4. Println 再也不想需要打印某些调试信息的时候导入fmt，而打印完成又去删除fmt引用了  
5. Printf 同上  
6. Sprintf 同上  
7. IsStop antnet是否停止  
8. IsRuning antnet是否运行中  
9. PathExists 判断路径是否存在  
10. Daemon 进入精灵进程  
11. GetStatis 获得antnet的统计信息  
12. Atoi 简化字符串到数值  
13. Itoa 简化数值到字符串  
14. ParseBaseKind 字符串到特定类型的转化  
15. CmdAct 将cmd和act转为一个int    
16. SetTomeout 设置一个定时器   
17. DelTimeout 删除定时器   
18. LogXXX 日志系列函数   

## 日志
antnet会默认会产生一个日志系统，通过antnet.Logxxx即可输出不同等级的日志。    
日志等级分类如下：
```
const (
	LogLevelAllOn  LogLevel = iota //开放说有日志
	LogLevelDebug                  //调试信息
	LogLevelInfo                   //资讯讯息
	LogLevelWarn                   //警告状况发生
	LogLevelError                  //一般错误，可能导致功能不正常
	LogLevelFatal                  //严重错误，会导致进程退出
	LogLevelAllOff                 //关闭所有日志
)
```

## redis封装
antnet对redis进行了一下封装。   
antnet.Redis代表了对redis的一个封装，主要记录了对eval指令的处理，能购把预先生成的lua脚本上传到redis得到hash，以后使用evalsha命令进行调用。
RedisManager用于管理一组redis数据库。   

## 定时器
antnet会默认运行一个基于时间轮的计时器，精度是毫秒，用于定时器使用。

## 数据模型
antnet自带了一个基于redis的数据模型处理，使用protobuf作为数据库定义语言，默认情况下，redis内部存储的数据是msgpack格式的，处理的时候你可以非常方便的将他转换为protobuf数据流发给你的客户端。       
你可以使用protobuf产生的go结构体作为数据模型，当存入redis时，存入msgpack字节流，之所以这么做，是为了方便redis里面能直接用lua脚本操作单个字段。    
当从数据库读出后，你可以方便的将他转换为pb字节流，填充到你定义好的pb结构体中，发送给客户端。

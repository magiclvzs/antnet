# antnet
A game server framework in Golang    一个Golang游戏服务器框架   
## 第一次使用
   在介绍antnet之前，我们先使用一次，看看antnet如何构建一个网络程序
```
package main

import (
	"antnet"
)

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
##代码组织
antnet尽可能把功能相关的代码组织到一块，让你能快速找到代码，比如parser打头的文件表示解析器相关，msgque打头的文件表示消息队列相关。
##依赖项
github.com/golang/protobuf   
github.com/vmihailenco/msgpack   
github.com/go-redis/redis   
##消息头
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
##消息
消息是对一次交互的抽象，每个消息都会用自己的解析器，有消息头部分（可能为nil），和数据部分。
##消息队列
antnet将数据流抽象为消息队列，无论是来自tcp，udp还是websocket的数据流，都会被抽象为消息队列。    
根据是否带有消息头，antnet将消息队列分为两种类型：        
1. 有消息头，MsgTypeMsg类型     
2. 没有消息头，MsgTypeCmd类型，数据以\n作为分割     
##解析器
antnet目前有三种解析器类型：  
1. cmd类型，类似telnet指令，用于直接和程序交互，比如上面查询玩家等级的例子。  
2. json类型，可以用于客户端或者服务器之间交互。  
3. protobuf类型，用于和客户端交互。   
这三种类型的解析器，都可以用antnet.Parser来创建，每个解析器需要一个Type字段和一个ErrType字段定义，Type字段表示了消息解析器的类型，而ErrType字段则决定了消息解析失败之后默认的行为。     
定义好解析之后，就需要注册解析器需要解析的消息，解析器支持两种模式：  
1. 基于cmd和act的解析，支持上面三种类型，配合消息头使用，使用Register进行注册。  
2. 基于输入的解析，可以支持json和cmd类型，这种消息往往没有消息头，使用RegisterMsg进行注册。  
####命令行解析器
命令行解析器用于解析命令行输入，类似telnet，我希望但服务器运行起来之后有一个非常简单的交流接口，直接基于telnet是最好了，而这个解析器就是为此准备，他可以接收不完整的输入，只要你最终输入完整即可，也可以接收你错误的输入，直到你输入正确为止。    
命令行解析器目前支持两种tag：   
1. `match:"k"`表示只需要匹配字段名即可，为了减少输入的大小写切换，在匹配的时候会将字段名默认作为小写匹配。    
2. `match:"kv"`表示需要匹配字段名和字段值   
     
命令行解析器的注册需要一个结构体，比如上面例子中的GetGamerLevel，是用来查询玩家等级的，定义了三个字段：    
1. Get字段，表示方法，比如get，set，reload，这种情况我只需要输入方法即可，而tag `match:"k"`则表示只需要匹配字段名即可   
2. Gamer字段，表示玩家id，没有tag，对于没有tag的字段，解析器会认为需要匹配字段名和值，比如输入gamer 1 会被认为合法，而gamer test则不合法，因为test不是int   
3. Level字段，表示玩家等级，有tag，表示只需要匹配level这个字段名即可。    

####json解析器
json用于解析json格式的数据，json解析器在MsgTypeCmd类型的消息队列中不接受不完整的输入。
####protobuf
解析器protobuf用于解析pb类型的数据
##处理器
处理器用于处理消息，一个处理器应该实现IMsgHandler消息接口：
```
type IMsgHandler interface {
	OnNewMsgQue(msgque IMsgQue) bool                //新的消息队列
	OnDelMsgQueue(msgque IMsgQue)                   //消息队列关闭
	OnProcessMsg(msgque IMsgQue, msg *Message) bool //处理消息
	OnConnectComplete(msgque IMsgQue, ok bool) bool //连接成功
	GetHandlerFunc(msg *Message) HandlerFunc
}
```

当然，一般情况，你只需在你的处理器里面添加antnet.DefMsgHandler定义即可。  
在antnet.DefMsgHandler里面，同样定义了Register和RegisterMsg函数，原理和解析器一样，也是为了区分不同的输入。   
如果你没有注册任何消息处理函数，系统会自动调用OnProcessMsg函数，如果你有定义的话。    
####处理器的调用时机
antnet会为每个tcp链接建立两个goroutine进行服务一个用于读，一个用于写，处理的回调发生在每个链接的读的goroutine之上，为什么要这么设计，是考虑当客户端的一个消息没有处理完成的时候真的有必要立即处理下一个消息吗？  
##启动服务
启动一个网络服务器使用antnet.StartServer函数，他被定义在msgque.go文件里面，一个服务目前需要一个处理器和一个解析器才可以运行。
##全局变量
为了方便使用antnet封装了一些全局变量：  
1. StartTick 用于标识antnet启动的时刻，是一个毫秒级的时间戳    
2. NowTick 用于标识antnet现在的时刻，是一个自动变化的毫秒级时间戳  
##全局函数
为了方便使用antnet封装了一些全局函数以供调用：  
1. WaitForSystemExit 用于等待用户输入ctrl+C以结束进程。  
2. Go 用于创建可被antnet管理的goroutine  
2. Go2 通Go，不同的是会有个默认的channel，以通知antnet的结束  
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

##日志
##redis封装
##定时器
antnet会默认运行一个基于时间轮的计时器，精度是毫秒，用于定时器使用。

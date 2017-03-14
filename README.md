# antnet
A game server framework in Golang
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
##解析器
antnet目前有三种解析器类型：  
1. cmd类型，类似telnet指令，用于直接和程序交互，比如上面查询玩家等级的例子。  
2. json类型，可以用于客户端或者服务器之间交互。  
3. protobuf类型，用于和客户端交互。   
这三种类型的解析器，都可以用antnet.Parser来创建，每个解析器需要一个Type字段和一个ErrType字段定义，Type字段表示了消息解析器的类型，而ErrType字段则决定了消息解析失败之后默认的行为。     
定义好解析之后，就需要注册解析器需要解析的消息，解析器支持两种模式：  
1. 基于cmd和act的解析，支持上面三种类型，配合消息头使用，使用Register进行注册。  
2. 基于输入的解析，可以支持json和cmd类型，这种消息往往没有消息头，使用RegisterMsg进行注册。  
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
##启动服务
启动一个网络服务器使用antnet.StartServer函数，他被定义在msgque.go文件里面，一个服务目前需要一个处理器和一个解析器才可以运行。
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

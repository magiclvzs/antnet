package antnet

import "sync"

type Error struct {
	Id  uint16
	Str string
}

func (r *Error) Error() string {
	return r.Str
}

var idErrMap = sync.Map{}
var errIdMap = sync.Map{}

func NewError(str string, id uint16) *Error {
	err := &Error{id, str}
	idErrMap.Store(id, err)
	errIdMap.Store(err, id)
	return err
}

var (
	ErrOk             = NewError("正确", 0)
	ErrDBErr          = NewError("数据库错误", 1)
	ErrProtoPack      = NewError("协议解析错误", 2)
	ErrProtoUnPack    = NewError("协议打包错误", 3)
	ErrMsgPackPack    = NewError("msgpack打包错误", 4)
	ErrMsgPackUnPack  = NewError("msgpack解析错误", 5)
	ErrPBPack         = NewError("pb打包错误", 6)
	ErrPBUnPack       = NewError("pb解析错误", 7)
	ErrJsonPack       = NewError("json打包错误", 8)
	ErrJsonUnPack     = NewError("json解析错误", 9)
	ErrCmdUnPack      = NewError("cmd解析错误", 10)
	ErrMsgLenTooLong  = NewError("数据过长", 11)
	ErrMsgLenTooShort = NewError("数据过短", 12)
	ErrHttpRequest    = NewError("http请求错误", 13)
	ErrCSVParse       = NewError("csv解析错误", 14)
	ErrGobPack        = NewError("gob打包错误", 15)
	ErrGobUnPack      = NewError("gob解析错误", 16)
	ErrServePanic     = NewError("服务器内部错误", 17)
	ErrNeedIntraNet   = NewError("需要内网环境", 18)
	ErrConfigPath     = NewError("配置路径错误", 50)

	ErrFileRead       = NewError("文件读取错误", 100)
	ErrDBDataType     = NewError("数据库数据类型错误", 101)
	ErrNetTimeout     = NewError("网络超时", 200)
	ErrNetUnreachable = NewError("网络不可达", 201)

	ErrClientReserve = NewError("客户端保留，服务器任何情况不会下发这个错误", 254)
	ErrErrIdNotFound = NewError("错误没有对应的错误码", 255)
)

var MinUserError = 256

func GetError(id uint16) *Error {
	if e, ok := idErrMap.Load(id); ok {
		return e.(*Error)
	}
	return ErrErrIdNotFound
}

func GetErrId(err error) uint16 {
	if id, ok := errIdMap.Load(err); ok {
		return id.(uint16)
	}
	id, _ := errIdMap.Load(ErrErrIdNotFound)
	return id.(uint16)
}

type ErrJsonStr struct {
	Error    int    `json:"error"`
	ErrorStr string `json:"errstr"`
}

func GetErrJsonStr(err error) string {
	return string(GetErrJsonData(err))
}
func GetErrJsonData(err error) []byte {
	data, _ := JsonPack(&ErrJsonStr{Error: int(GetErrId(err)), ErrorStr: err.Error()})
	return data
}

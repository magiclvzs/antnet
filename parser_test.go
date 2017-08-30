package antnet

import (
	"testing"
)

type GetGamerLevel struct {
	Get   string `match:"k"`
	Gamer int
	Level int `match:"k"`
}

type GetGamerRmb struct {
	Get   string `match:"k"`
	Gamer int
	Rmb   int `match:"k"`
}

func Test_CmdParser(t *testing.T) {
	pm := Parser{Type: ParserTypeCmd}
	pm.RegisterMsg(&GetGamerLevel{}, nil)

	p := pm.Get()
	m, _ := p.ParseC2S(NewStrMsg("get gamer 1 level"))
	Printf("%#v\n", m.C2S().(*GetGamerLevel))

	Println(m.C2SString())
}

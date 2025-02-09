package posting

import (
	"fmt"

	"github.com/frustra/bbcode"
	"github.com/gochan-org/gochan/pkg/gcutil"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func luaTableToHTMLTag(l *lua.LState, table *lua.LTable) (*bbcode.HTMLTag, error) {
	tag := &bbcode.HTMLTag{
		Name: table.RawGetString("name").String(),
	}
	value := table.RawGetString("value")
	if value.Type() != lua.LTNil {
		tag.Value = value.String()
	}
	attrsLV := table.RawGetString("attrs")
	switch attrsLV.Type() {
	case lua.LTTable:
		attrsLT := attrsLV.(*lua.LTable)
		fmt.Println("attrs size:", attrsLT.Len())
		attrsLT.ForEach(func(key, val lua.LValue) {
			if tag.Attrs == nil {
				tag.Attrs = make(map[string]string)
			}
			tag.Attrs[key.String()] = val.String()
		})
	case lua.LTNil:
	default:
		return nil, fmt.Errorf("expected table or nil for attrs value, got %s", attrsLV.Type().String())
	}
	childrenLV := table.RawGetString("children")
	switch childrenLV.Type() {
	case lua.LTTable:
		childrenT := childrenLV.(*lua.LTable)
		if childrenT.Len() > 0 {
			tag.Children = make([]*bbcode.HTMLTag, childrenT.Len())
			childrenT.ForEach(func(iLV, childLV lua.LValue) {
				childT, err := luaTableToHTMLTag(l, childLV.(*lua.LTable))
				if err != nil {
					l.RaiseError("Error converting child table to HTMLTag: %v", err)
					return
				}
				tag.Children[int(iLV.(lua.LNumber)-1)] = childT
			})
		}
	case lua.LTNil:
	default:
		return nil, fmt.Errorf("expected table or nil for children value, got %s", childrenLV.Type().String())
	}
	return tag, nil
}

func PreloadBBCodeModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"set_tag": func(l *lua.LState) int {
			bbcodeTag := l.CheckString(1)
			bbCodeLV := l.CheckAny(2)
			if bbCodeLV.Type() == lua.LTNil {
				msgfmtr.bbCompiler.SetTag(bbcodeTag, nil)
				return 0
			}
			bbcodeFunc := l.CheckFunction(2)
			msgfmtr.bbCompiler.SetTag(bbcodeTag, func(node *bbcode.BBCodeNode) (*bbcode.HTMLTag, bool) {
				err := l.CallByParam(lua.P{
					Fn:   bbcodeFunc,
					NRet: 2,
				}, luar.New(l, node))
				if err != nil {
					gcutil.LogError(err).Caller().Msg("Error calling bbcode function")
					l.RaiseError("Error calling bbcode function: %v", err)
					return nil, false
				}
				tagRet := l.CheckAny(-2)
				if tagRet.Type() != lua.LTTable {
					l.RaiseError("Invalid return value from bbcode function (expected table)")
					return nil, false
				}
				tagTable := tagRet.(*lua.LTable)
				tag, err := luaTableToHTMLTag(l, tagTable)
				if err != nil {
					gcutil.LogError(err).Caller().Msg("Error converting table to HTMLTag")
					l.RaiseError("Error converting table to HTMLTag: %v", err)
					return nil, false
				}
				return tag, true
			})
			return 0
		},
	})
	l.Push(t)
	return 1
}

package config

import (
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"system_critical_config": func(l *lua.LState) int {
			l.Push(luar.New(l, &Cfg.SystemCriticalConfig))
			return 1
		},
		"site_config": func(l *lua.LState) int {
			l.Push(luar.New(l, &Cfg.SiteConfig))
			return 1
		},
		"board_config": func(l *lua.LState) int {
			numArgs := l.GetTop()
			board := ""
			if numArgs > 0 {
				board = l.CheckString(1)
			}
			l.Push(luar.New(l, GetBoardConfig(board)))
			return 1
		},
	})

	l.Push(t)
	return 1
}

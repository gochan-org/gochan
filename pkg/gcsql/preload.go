package gcsql

import (
	"database/sql"
	"reflect"

	"github.com/gochan-org/gochan/pkg/gcplugin/luautil"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type lvalueScanner struct {
	val   lua.LValue
	state *lua.LState
}

func (lvs *lvalueScanner) Scan(src any) error {
	typeof := reflect.TypeOf(src)
	if typeof != nil && typeof.String() == "[]uint8" {
		src = string(src.([]uint8))
	}
	lvs.val = luar.New(lvs.state, src)
	return nil
}

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"query_rows": func(l *lua.LState) int {
			queryStr := l.CheckString(1)
			queryArgsL := l.CheckAny(2)

			var queryArgs []any
			if queryArgsL.Type() != lua.LTNil {
				table := queryArgsL.(*lua.LTable)
				table.ForEach(func(_ lua.LValue, val lua.LValue) {
					arg := luautil.LValueToInterface(l, val)
					queryArgs = append(queryArgs, arg)
				})
			}

			rows, err := QuerySQL(queryStr, queryArgs...)

			l.Push(luar.New(l, rows))
			l.Push(luar.New(l, err))
			return 2

		},
		"execute_sql": func(l *lua.LState) int {
			execStr := l.CheckString(1)
			execArgsL := l.CheckAny(2)
			var execArgs []any
			if execArgsL.Type() != lua.LTNil {
				table := execArgsL.(*lua.LTable)
				table.ForEach(func(_, val lua.LValue) {
					arg := luautil.LValueToInterface(l, val)
					execArgs = append(execArgs, arg)
				})
			}
			result, err := ExecSQL(execStr)

			l.Push(luar.New(l, result))
			l.Push(luar.New(l, err))
			return 2
		},
		"scan_rows": func(l *lua.LState) int {
			rows := l.CheckUserData(1).Value.(*sql.Rows)
			table := l.CheckTable(2)
			var scanners []any
			colNames, err := rows.Columns()
			if err != nil {
				l.Push(luar.New(l, err))
				return 1
			}

			for range colNames {
				scanners = append(scanners, &lvalueScanner{state: l})
			}

			if err = rows.Scan(scanners...); err != nil {
				l.Push(luar.New(l, err))
				return 1
			}
			for i, name := range colNames {
				table.RawSetString(name, scanners[i].(*lvalueScanner).val)
			}
			l.Push(lua.LNil)
			return 1
		},
	})

	l.Push(t)
	return 1
}

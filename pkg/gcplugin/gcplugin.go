package gcplugin

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"plugin"
	"reflect"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	gluahttp "github.com/vadv/gopher-lua-libs/http"

	async "github.com/CuberL/glua-async"
	luaFilePath "github.com/vadv/gopher-lua-libs/filepath"
	luaStrings "github.com/vadv/gopher-lua-libs/strings"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const (
	tableArgFmt = "invalid value for key %q passed to table, expected %s, got %s"
)

var (
	lState             *lua.LState
	eventPlugins       map[string][]*lua.LFunction
	ErrInvalidInitFunc = errors.New("invalid InitPlugin, expected function with 0 arguments and 1 return value (error type)")
)

func initLua() {
	if lState == nil {
		lState = lua.NewState()
		preloadLua()
	}
}

func ClosePlugins() {
	if lState != nil {
		lState.Close()
	}
}

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

func lvalueToInterface(l *lua.LState, v lua.LValue) interface{} {
	lt := v.Type()
	switch lt {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		return lua.LVAsBool(v)
	case lua.LTNumber:
		return lua.LVAsNumber(v)
	case lua.LTString:
		return lua.LVAsString(v)
	case lua.LTUserData:
		l.Push(v)
		return l.CheckUserData(l.GetTop()).Value
	default:
		gcutil.LogError(nil).Caller(1).
			Interface("lvalue", v).
			Str("type", lt.String()).
			Msg("Unrecognized or unsupported Lua type")
	}
	return nil
}

func createLuaLogFunc(which string) lua.LGFunction {
	return func(l *lua.LState) int {
		switch which {
		case "info":
			l.Push(luar.New(l, gcutil.LogInfo()))
		case "warn":
			l.Push(luar.New(l, gcutil.LogWarning()))
		case "error":
			numArgs := l.GetTop()
			if numArgs == 0 {
				l.Push(luar.New(l, gcutil.LogError(nil)))
			} else {
				errVal := l.CheckAny(-1)
				errI := lvalueToInterface(l, errVal)
				err := fmt.Errorf("%v", errI)

				l.Push(luar.New(l, gcutil.LogError(err)))
			}
		}
		return 1
	}
}

func luaEventRegisterHandlerAdapter(l *lua.LState, fn *lua.LFunction) events.EventHandler {
	return func(trigger string, data ...interface{}) error {
		args := []lua.LValue{
			luar.New(l, trigger),
		}
		for _, i := range data {
			args = append(args, luar.New(l, i))
		}
		l.CallByParam(lua.P{
			Fn:   fn,
			NRet: 1,
			// Protect: true,
		}, args...)
		errStr := lua.LVAsString(l.Get(-1))
		if errStr != "" {
			return errors.New(errStr)
		}
		return nil
	}
}

func preloadLua() {
	luaFilePath.Preload(lState)
	luaStrings.Preload(lState)
	gluahttp.Preload(lState)
	async.Init(lState)

	lState.PreloadModule("config", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"system_critical_config": func(l *lua.LState) int {
				l.Push(luar.New(l, config.GetSystemCriticalConfig()))
				return 1
			},
			"site_config": func(l *lua.LState) int {
				l.Push(luar.New(l, config.GetSiteConfig()))
				return 1
			},
			"board_config": func(l *lua.LState) int {
				numArgs := l.GetTop()
				board := ""
				if numArgs > 0 {
					board = l.CheckString(1)
				}
				l.Push(luar.New(l, config.GetBoardConfig(board)))
				return 1
			},
		})

		l.Push(t)
		return 1
	})

	lState.PreloadModule("events", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"register_event": func(l *lua.LState) int {
				table := l.CheckTable(-2)
				var triggers []string
				table.ForEach(func(i, val lua.LValue) {
					triggers = append(triggers, val.String())
				})
				fn := l.CheckFunction(-1)
				events.RegisterEvent(triggers, luaEventRegisterHandlerAdapter(l, fn))
				return 0
			},
			"trigger_event": func(l *lua.LState) int {
				trigger := l.CheckString(1)
				numArgs := l.GetTop()
				var data []interface{}
				for i := 2; i <= numArgs; i++ {
					v := l.CheckAny(i)
					data = append(data, lvalueToInterface(l, v))
				}
				events.TriggerEvent(trigger, data...)
				return 0
			},
		})
		l.Push(t)
		return 1
	})

	lState.PreloadModule("gclog", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"info_log":  createLuaLogFunc("info"),
			"warn_log":  createLuaLogFunc("warn"),
			"error_log": createLuaLogFunc("error"),
		})
		l.Push(t)
		return 1
	})

	lState.PreloadModule("gcsql", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"query_rows": func(l *lua.LState) int {
				queryStr := l.CheckString(1)
				queryArgsL := l.CheckAny(2)

				var queryArgs []any
				if queryArgsL.Type() != lua.LTNil {
					table := queryArgsL.(*lua.LTable)
					table.ForEach(func(_ lua.LValue, val lua.LValue) {
						arg := lvalueToInterface(l, val)
						queryArgs = append(queryArgs, arg)
					})
				}

				rows, err := gcsql.QuerySQL(queryStr, queryArgs...)

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
						arg := lvalueToInterface(l, val)
						execArgs = append(execArgs, arg)
					})
				}
				result, err := gcsql.ExecSQL(execStr)

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
	})

	lState.PreloadModule("gctemplates", func(l *lua.LState) int {
		t := l.NewTable()

		l.SetFuncs(t, map[string]lua.LGFunction{
			"load_template": func(l *lua.LState) int {
				var tmplPaths []string
				for i := 0; i < l.GetTop(); i++ {
					tmplPaths = append(tmplPaths, l.CheckString(i+1))
				}
				tmpl, err := gctemplates.LoadTemplate(tmplPaths...)
				l.Push(luar.New(l, tmpl))
				l.Push(luar.New(l, err))
				return 2
			},
			"minify_template": func(l *lua.LState) int {
				tmplUD := l.CheckUserData(1)
				tmpl := tmplUD.Value.(*template.Template)
				dataTable := l.CheckTable(2)
				data := map[string]interface{}{}
				dataTable.ForEach(func(l1, l2 lua.LValue) {
					data[l1.String()] = lvalueToInterface(l, l2)
				})
				writer := l.CheckUserData(3).Value.(io.Writer)
				mediaType := l.CheckString(4)
				err := serverutil.MinifyTemplate(tmpl, data, writer, mediaType)
				l.Push(luar.New(l, err))
				return 1
			},
			"parse_template": func(l *lua.LState) int {
				tmplName := l.CheckString(1)
				tmplData := l.CheckString(2)
				tmpl, err := gctemplates.ParseTemplate(tmplName, tmplData)
				l.Push(luar.New(l, tmpl))
				l.Push(luar.New(l, err))
				return 2
			},
		})

		l.Push(t)
		return 1
	})

	lState.PreloadModule("manage", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"ban_ip": func(l *lua.LState) int {
				now := time.Now()
				ban := &gcsql.IPBan{
					IP: l.CheckString(1),
				}
				ban.IsActive = true
				ban.AppealAt = now
				ban.CanAppeal = true

				durOrNil := l.CheckAny(2)
				var err error
				switch durOrNil.Type() {
				case lua.LTNil:
					ban.Permanent = true
				case lua.LTString:
					var duration time.Duration
					duration, err = durationutil.ParseLongerDuration(lua.LVAsString(durOrNil))
					if err != nil {
						l.Push(luar.New(l, err))
						return 1
					}
					ban.ExpiresAt = time.Now().Add(duration)
				default:
					lState.ArgError(2, "Expected string or nil value")
				}

				ban.Message = l.CheckString(3)

				if l.GetTop() > 3 {
					t := l.CheckTable(4)
					var failed bool
					t.ForEach(func(keyLV, val lua.LValue) {
						key := lua.LVAsString(keyLV)
						valType := val.Type()
						switch key {
						case "board":
							fallthrough
						case "BoardID":
							fallthrough
						case "board_id":
							switch valType {
							case lua.LTNil:
								// global
							case lua.LTNumber:
								ban.BoardID = new(int)
								*ban.BoardID = int(lua.LVAsNumber(val))
							case lua.LTString:
								boardDir := lua.LVAsString(val)
								if boardDir != "" {
									var id int
									if id, err = gcsql.GetBoardIDFromDir(boardDir); err != nil {
										l.Push(luar.New(l, err))
										return
									}
									ban.BoardID = new(int)
									*ban.BoardID = id
								}
							default:
								failed = true
								l.Push(lua.LNil)
								l.RaiseError(tableArgFmt, key, "string, number, or nil", valType)
								return
							}
						case "staff":
							fallthrough
						case "staff_id":
							fallthrough
						case "StaffID":
							switch valType {
							case lua.LTString:
								ban.StaffID, err = gcsql.GetStaffID(lua.LVAsString(val))
								if err != nil {
									l.Push(luar.New(l, err))
									failed = true
									return
								}
							case lua.LTNumber:
								ban.StaffID = int(lua.LVAsNumber(val))
							default:
								failed = true
								l.Push(lua.LNil)
								l.RaiseError(tableArgFmt, key, "number or string", valType)
							}
						case "post_id":
							fallthrough
						case "post":
							fallthrough
						case "PostID":
							if valType != lua.LTNumber {
								failed = true
								l.Push(lua.LNil)
								l.RaiseError(tableArgFmt, key, "number", valType)
								return
							}
						case "is_thread_ban":
							fallthrough
						case "IsThreadBan":
							ban.IsThreadBan = lua.LVAsBool(val)
						case "appeal_after":
							fallthrough
						case "AppealAfter":
							str := lua.LVAsString(val)
							dur, err := durationutil.ParseLongerDuration(str)
							if err != nil {
								l.Push(luar.New(l, err))
								failed = true
								return
							}
							ban.AppealAt = now.Add(dur)
						case "can_appeal":
							fallthrough
						case "appealable":
							fallthrough
						case "CanAppeal":
							ban.CanAppeal = lua.LVAsBool(val)
						case "staff_note":
							fallthrough
						case "StaffNote":
							ban.StaffNote = lua.LVAsString(val)
						}
					})
					if failed {
						return 1
					}
				}
				if ban.StaffID < 1 {
					l.Push(luar.New(l, errors.New("missing staff key in table")))
					return 1
				}
				ban.IssuedAt = time.Now()
				err = gcsql.NewIPBan(ban)
				l.Push(luar.New(l, err))
				return 1
			},
			"register_manage_page": func(l *lua.LState) int {
				actionID := l.CheckString(1)
				actionTitle := l.CheckString(2)
				actionPerms := l.CheckInt(3)
				actionJSON := l.CheckInt(4)
				fn := l.CheckFunction(5)
				actionHandler := func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
					if err = l.CallByParam(lua.P{
						Fn:   fn,
						NRet: 2,
						// Protect: true,
					}, luar.New(l, writer), luar.New(l, request), luar.New(l, staff), lua.LBool(wantsJSON), luar.New(l, infoEv), luar.New(l, errEv)); err != nil {
						return "", err
					}
					out := lua.LVAsString(l.Get(-2))
					errStr := lua.LVAsString(l.Get(-1))
					if errStr != "" {
						err = errors.New(errStr)
					}
					return out, err
				}
				manage.RegisterManagePage(actionID, actionTitle, actionPerms, actionJSON, actionHandler)
				return 0
			},
		})

		l.Push(t)
		return 1
	})

	lState.PreloadModule("uploads", func(l *lua.LState) int {
		t := l.NewTable()

		l.SetFuncs(t, map[string]lua.LGFunction{
			"register_handler": func(l *lua.LState) int {
				ext := l.CheckString(1)
				handler := l.CheckFunction(2)
				uploads.RegisterUploadHandler(ext, func(upload *gcsql.Upload, post *gcsql.Post, board, filePath, thumbPath, catalogThumbPath string, infoEv, accessEv, errEv *zerolog.Event) error {
					l.CallByParam(lua.P{
						Fn:   handler,
						NRet: 1,
						// Protect: true,
					}, luar.New(l, upload), luar.New(l, post), lua.LString(board), lua.LString(filePath), lua.LString(thumbPath), lua.LString(catalogThumbPath))

					errRet := l.CheckAny(-1)
					if errRet != nil && errRet.Type() != lua.LTNil {
						return errors.New(errRet.String())
					}
					return nil
				})
				return 0
			},
			"get_thumbnail_ext": func(l *lua.LState) int {
				fileExt := l.CheckString(1)
				l.Push(luar.New(l, uploads.GetThumbnailExtension(fileExt)))
				return 1
			},
			"set_thumbnail_ext": func(l *lua.LState) int {
				fileExt := l.CheckString(1)
				thumbExt := l.CheckString(2)
				uploads.SetThumbnailExtension(fileExt, thumbExt)
				return 0
			},
		})

		l.Push(t)
		return 1
	})

	lState.SetGlobal("_GOCHAN_VERSION", lua.LString(config.GetVersion().String()))
}

func registerEventFunction(name string, fn *lua.LFunction) {
	switch name {
	case "onStartup":
		fallthrough
	case "onPost":
		fallthrough
	case "onDelete":
		eventPlugins[name] = append(eventPlugins[name], fn)
	}
}

func LoadPlugins(paths []string) error {
	var err error
	var luaInitialized bool
	for _, pluginPath := range paths {
		ext := path.Ext(pluginPath)
		gcutil.LogInfo().Str("pluginPath", pluginPath).Msg("Loading plugin")
		switch ext {
		case ".lua":
			if !luaInitialized {
				initLua()
				luaInitialized = true
			}
			if err = lState.DoFile(pluginPath); err != nil {
				return err
			}
			pluginTable := lState.NewTable()
			pluginTable.ForEach(func(key, val lua.LValue) {
				keyStr := key.String()
				fn, ok := val.(*lua.LFunction)
				if !ok {
					return
				}
				registerEventFunction(keyStr, fn)
			})
		case ".so":
			nativePlugin, err := plugin.Open(pluginPath)
			if err != nil {
				return err
			}
			initFuncSymbol, err := nativePlugin.Lookup("InitPlugin")
			if err != nil {
				return err
			}
			initFunc, ok := initFuncSymbol.(func() error)
			if !ok {
				return ErrInvalidInitFunc
			}
			if err = initFunc(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized plugin type (expected .lua or .so extension): %s", pluginPath)
		}
	}
	return nil
}

package gcplugin

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"plugin"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	luar "layeh.com/gopher-luar"

	"github.com/cjoudrey/gluahttp"

	async "github.com/CuberL/glua-async"
	luaFilePath "github.com/vadv/gopher-lua-libs/filepath"
	luaStrings "github.com/vadv/gopher-lua-libs/strings"
	lua "github.com/yuin/gopher-lua"
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

func preloadLua() {
	luaFilePath.Preload(lState)
	luaStrings.Preload(lState)
	async.Init(lState)

	lState.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	lState.PreloadModule("url", func(l *lua.LState) int {
		t := l.NewTable()
		l.SetFuncs(t, map[string]lua.LGFunction{
			"join_path": func(l *lua.LState) int {
				argc := l.GetTop()
				base := l.CheckString(1)
				var pathArgs []string
				for i := 2; i <= argc; i++ {
					pathArgs = append(pathArgs, l.CheckString(i))
				}
				result, err := url.JoinPath(base, pathArgs...)
				l.Push(lua.LString(result))
				l.Push(luar.New(l, err))
				return 2
			},
			"query_escape": func(l *lua.LState) int {
				query := l.CheckString(1)
				l.Push(lua.LString(url.QueryEscape(query)))
				return 1
			},
			"query_unescape": func(l *lua.LState) int {
				query := l.CheckString(1)
				result, err := url.QueryUnescape(query)
				l.Push(lua.LString(result))
				l.Push(luar.New(l, err))
				return 1
			},
		})
		l.Push(t)
		return 1
	})

	lState.PreloadModule("config", config.PreloadModule)
	lState.PreloadModule("events", events.PreloadModule)
	lState.PreloadModule("gclog", gcutil.PreloadModule)
	lState.PreloadModule("gcsql", gcsql.PreloadModule)
	lState.PreloadModule("gctemplates", gctemplates.PreloadModule)
	lState.PreloadModule("manage", manage.PreloadModule)
	lState.PreloadModule("uploads", uploads.PreloadModule)

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

package geoip

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type luaHandler struct {
	lState         *lua.LState
	initFunc       lua.LValue
	getCountryFunc lua.LValue
	closeFunc      lua.LValue
}

func (lh *luaHandler) Init(options map[string]any) error {
	if lh.initFunc == lua.LNil {
		return nil
	}
	optionsT := lh.lState.NewTable()
	for k, v := range options {
		optionsT.RawSetString(k, luar.New(lh.lState, v))
	}
	p := lua.P{
		Fn:   lh.initFunc,
		NRet: 1,
	}
	err := lh.lState.CallByParam(p, optionsT)
	if err != nil {
		return err
	}
	errStr := lua.LVAsString(lh.lState.Get(-1))
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

func (lh *luaHandler) GetCountry(request *http.Request, board string, errEv *zerolog.Event) (*Country, error) {
	p := lua.P{
		Fn:   lh.getCountryFunc,
		NRet: 2,
	}
	err := lh.lState.CallByParam(p,
		luar.New(lh.lState, request),
		lua.LString(board),
		luar.New(lh.lState, errEv))

	if err != nil {
		return nil, err
	}
	countryTable, ok := lh.lState.Get(-2).(*lua.LTable)
	if !ok {
		return nil, errors.New("invalid value returned by get_country (expected table)")
	}
	errStr := lua.LVAsString(lh.lState.Get(-1))
	if errStr != "" {
		return nil, errors.New(errStr)
	}
	name, ok := countryTable.RawGetString("name").(lua.LString)
	if !ok {
		return nil, errors.New("invalid name value in table returned gy bet_country (expected string)")
	}
	flag, ok := countryTable.RawGetString("flag").(lua.LString)
	if !ok {
		return nil, errors.New("invalid flag value in table returned gy bet_country (expected string)")
	}
	return &Country{
		Name: name.String(),
		Flag: flag.String(),
	}, nil
}

func (lh *luaHandler) Close() error {
	if lh.closeFunc == lua.LNil {
		return nil
	}
	p := lua.P{
		Fn:   lh.closeFunc,
		NRet: 1,
	}
	err := lh.lState.CallByParam(p)
	if err != nil {
		return err
	}
	errStr := lua.LVAsString(lh.lState.Get(-1))
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()
	l.SetFuncs(t, map[string]lua.LGFunction{
		"register_handler": func(l *lua.LState) int {
			name := l.CheckString(1)
			handlerTable := l.CheckTable(2)
			initFuncVal := handlerTable.RawGetString("init")
			lookupFunc := handlerTable.RawGetString("get_country")
			closeFuncVal := handlerTable.RawGetString("close")
			handler := &luaHandler{
				lState:         l,
				initFunc:       initFuncVal,
				getCountryFunc: lookupFunc,
				closeFunc:      closeFuncVal,
			}

			l.Push(luar.New(l, RegisterGeoIPHandler(name, handler)))
			return 1
		},
		"country_name": func(l *lua.LState) int {
			abbr := l.CheckString(1)
			name, err := GetCountryName(abbr)
			l.Push(lua.LString(name))
			l.Push(luar.New(l, err))
			return 2
		},
	})
	l.Push(t)
	return 1
}

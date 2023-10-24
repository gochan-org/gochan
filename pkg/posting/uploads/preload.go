package uploads

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func PreloadModule(l *lua.LState) int {
	t := l.NewTable()

	l.SetFuncs(t, map[string]lua.LGFunction{
		"register_handler": func(l *lua.LState) int {
			ext := l.CheckString(1)
			handler := l.CheckFunction(2)
			RegisterUploadHandler(ext, func(upload *gcsql.Upload, post *gcsql.Post, board, filePath, thumbPath, catalogThumbPath string, infoEv, accessEv, errEv *zerolog.Event) error {
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
			l.Push(luar.New(l, GetThumbnailExtension(fileExt)))
			return 1
		},
		"set_thumbnail_ext": func(l *lua.LState) int {
			fileExt := l.CheckString(1)
			thumbExt := l.CheckString(2)
			SetThumbnailExtension(fileExt, thumbExt)
			return 0
		},
	})

	l.Push(t)
	return 1
}

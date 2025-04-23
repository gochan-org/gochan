package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/rs/zerolog"
)

const (
	infoPage = `<b>Gochan version:</b> %s<br/>
<b>Go version:</b> %s<br/>
<b>GOOS:</b> %s<br/>
<b>DB type:</b> %s<br/>
<b>Loaded plugins:</b><br/>
<ul><li>%s</li></ul>`
)

func InitPlugin() error {
	var err error
	manage.RegisterManagePage("gochaninfo", "Gochan info", manage.AdminPerms, manage.NoJSON,
		func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			return fmt.Sprintf(infoPage,
					config.GochanVersion,
					runtime.Version(),
					runtime.GOOS,
					config.GetSystemCriticalConfig().DBtype,
					strings.Join(config.GetSystemCriticalConfig().Plugins, "</li><li>")),
				nil
		},
	)
	return err
}

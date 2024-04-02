package gctemplates_test

import "github.com/gochan-org/gochan/pkg/config"

var (
	jsConstsCases = []templateTestCase{
		{
			desc: "base test",
			data: map[string]any{
				"styles": []config.Style{
					{Name: "Pipes", Filename: "pipes.css"},
					{Name: "Yotsuba A", Filename: "yotsuba.css"},
				},
				"defaultStyle": "pipes.css",
				"webroot":      "/",
				"timezone":     -1,
			},
			expectedOutput: `var styles=[{Name:"Pipes",Filename:"pipes.css"},{Name:"Yotsuba A",Filename:"yotsuba.css"}];var defaultStyle="pipes.css";var webroot="/";var serverTZ=-1;`,
		},
		{
			desc: "empty values",
			data: map[string]any{
				"defaultStyle": "",
				"webroot":      "",
				"timezone":     0,
			},
			expectedOutput: `var styles=[];var defaultStyle="";var webroot="";var serverTZ=0;`,
		},
	}
)

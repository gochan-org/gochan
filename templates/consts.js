{{/*
	This will be used for storing configuration-dependent JS variables,
	instead of loading them on every HTML page.
*/ -}}
var styles = [{{range $ii, $style := .Styles -}}
	{{if gt $ii 0}}, {{end -}}
	{Name: "{{js $style.Name}}", Filename: "{{js $style.Filename}}"}
{{- end}}];
var defaultStyle = "{{.DefaultStyle}}";
var webroot = "{{.SiteWebfolder}}";

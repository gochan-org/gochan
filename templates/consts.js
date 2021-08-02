{{/*
	This will be used for storing configuration-dependent JS variables,
	instead of loading them on every HTML page.
*/ -}}
var styles = [
	{{- range $ii, $style := .styles -}}
		{{if gt $ii 0}},{{end -}}
		{Name: "{{js $style.Name}}", Filename: "{{js $style.Filename}}"}
	{{- end -}}
];
var defaultStyle = "{{js .default_style}}";
var webroot = "{{js .webroot}}";
var serverTZ = {{.timezone}};

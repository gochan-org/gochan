var styles = [
	{{- range $ii, $style := .styles -}}
		{{if gt $ii 0}},{{end -}}
		{Name: "{{js $style.Name}}", Filename: "{{js $style.Filename}}"}
	{{- end -}}
];
var defaultStyle = "{{js .defaultStyle}}";
var webroot = "{{js .webroot}}";
var serverTZ = {{js .timezone}};
var fileTypes = [
	{{- range $ext, $_ := .fileTypes -}}
		"{{js $ext}}",
	{{- end -}}
];
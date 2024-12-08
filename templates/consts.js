const styles = [
	{{- range $ii, $style := .styles -}}
		{{if gt $ii 0}},{{end -}}
		{Name: "{{js $style.Name}}", Filename: "{{js $style.Filename}}"}
	{{- end -}}
];
const defaultStyle = "{{js .defaultStyle}}";
const webroot = "{{js .webroot}}";
const serverTZ = {{js .timezone}};
const fileTypes = [
	{{- range $ext, $_ := .fileTypes -}}
		"{{js $ext}}",
	{{- end -}}
];
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	compositeStructTypes = []string{
		"SystemCriticalConfig", "SQLConfig", "SiteConfig", "CaptchaConfig", "BoardConfig", "PostConfig", "UploadConfig",
	}
	explicitlyNamedStructTypes = []string{
		"PageBanner", "BoardCooldowns",
	}
)

type columnLengths struct {
	fieldLength   int
	typeLength    int
	defaultLength int
	docLength     int
}

func (c *columnLengths) setLengths(strs ...structType) {
	c.fieldLength = 6
	c.typeLength = 5
	c.defaultLength = 0
	c.docLength = 4
	for _, str := range strs {
		for _, field := range str.fields {
			if len(field.name) > c.fieldLength {
				c.fieldLength = len(field.name)
			}
			if len(field.fType) > c.typeLength {
				c.typeLength = len(field.fType)
			}
			if len(field.defaultVal) > c.defaultLength {
				c.defaultLength = len(field.defaultVal)
			}
			if len(field.doc) > c.docLength {
				c.docLength = len(field.doc)
			}
		}
	}
	if c.defaultLength > 0 && c.defaultLength < 8 {
		c.defaultLength = 8
	}
}

func mustParse(fset *token.FileSet, filename, filePath string) *ast.File {
	ba, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	f, err := parser.ParseFile(fset, filename, string(ba), parser.ParseComments|parser.DeclarationErrors)
	if err != nil {
		panic(err)
	}
	return f
}

type structType struct {
	name   string
	doc    string
	fields []fieldType
}

type fieldType struct {
	composite  string
	name       string
	fType      string
	defaultVal string
	doc        string
}

func docStructs(dir string) (map[string]structType, error) {
	structMap := make(map[string]structType)
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		var structName string
		var structDoc string
		file := mustParse(fset, d.Name(), path)

		structDocs := make(map[string]string)

		ast.Inspect(file, func(n ast.Node) bool {
			switch t := n.(type) {
			case *ast.TypeSpec:
				structName = t.Name.String()
				// log.Println(structName, "doc:", t)
				if t.Doc == nil {
					structDoc = structDocs[structName]
				} else {
					structDoc = t.Doc.Text()
				}
			case *ast.StructType:
				st := structType{
					name: structName,
					doc:  structDoc,
				}
				for _, field := range t.Fields.List {
					var fieldT fieldType
					if field.Names == nil {
						fieldT.composite = field.Type.(*ast.Ident).Obj.Name
					} else {
						fieldT.name = field.Names[0].String()
					}
					if field.Doc.Text() == "" {
						// field has no documentation, skip it
						continue
					}

					if field.Doc != nil {
						fieldT.doc = field.Doc.Text()
					}
					docLines := strings.Split(fieldT.doc, "\n")

					for _, line := range docLines {
						if strings.HasPrefix(strings.ToLower(line), "default: ") && fieldT.defaultVal == "" {
							fieldT.defaultVal = line[9:]
							break
						}
					}

					switch tt := field.Type.(type) {
					case *ast.Ident:
						fieldT.fType = tt.Name
					case *ast.ArrayType:
						// fmt.Printf("%v - %#v\n", tt.Elt, tt.Elt)
						if selectorExpr, ok := tt.Elt.(*ast.SelectorExpr); ok {
							fieldT.fType = "[]" + fmt.Sprintf("%v.%v", selectorExpr.X, selectorExpr.Sel)
						} else {
							fieldT.fType = "[]" + fmt.Sprint(tt.Elt)
						}
						// fieldT.fType = "[]" + fmt.Sprint(tt.Elt)
					case *ast.MapType:
						fieldT.fType = fmt.Sprintf("map[%v]%v", tt.Key, tt.Value)
					case *ast.StarExpr:
						fieldT.fType = fmt.Sprint(tt.X)
					default:
						panic(fmt.Sprintf("%#v", field.Type))
					}
					st.fields = append(st.fields, fieldT)
				}
				structMap[structName] = st
			case *ast.File:
				// log.Println("file name", t.Name)
			case *ast.ImportSpec:
			case *ast.BasicLit:
				// log.Println("basiclit:", t.Kind)
			case *ast.ValueSpec:
				// log.Println("valuespec:", t)
				if t.Doc != nil {
					log.Println("ValueSpec doc:", t.Doc.Text())
				}
			case *ast.StarExpr:
				// log.Println("starexpr:", t)
			case *ast.CompositeLit:
				// log.Println("compositelit:", t)
			case *ast.MapType:
				// log.Println("maptype:", t)
			case *ast.ArrayType:
				// log.Println("arraytype:", t)
			case *ast.FieldList:
				// log.Println("fieldlist:", t)
			case *ast.Field:
				// log.Println("field:", t)
			case *ast.BlockStmt:
				// log.Println("blockstmt:", t)
			case *ast.GenDecl:
				doc := t.Doc.Text()
				if doc != "" {
					firstSpace := strings.Index(doc, " ")
					if firstSpace > 0 {
						probableName := doc[:firstSpace]
						structDocs[probableName] = doc
					}
				}
			}
			return true
		})

		return nil
	})
	return structMap, err
}

func fieldsAsMarkdownTable(str *structType, builder *strings.Builder, named bool, showColumnHeaders bool, lengths *columnLengths) {
	if named {
		builder.WriteString("## " + str.name + "\n")
		if str.doc != "" {
			builder.WriteString(str.doc)
		}
	}
	if lengths == nil {
		lengths = &columnLengths{}
		lengths.setLengths(*str)
	}

	if showColumnHeaders {
		builder.WriteString("Field")
		for range lengths.fieldLength - 4 {
			builder.WriteRune(' ')
		}
		builder.WriteString("|Type")
		for range lengths.typeLength - 3 {
			builder.WriteRune(' ')
		}
		if lengths.defaultLength > 0 {
			builder.WriteString("|Default")
			for range lengths.defaultLength - 4 {
				builder.WriteRune(' ')
			}
		}
		builder.WriteString("|Info\n")
		for range lengths.fieldLength + 1 {
			builder.WriteRune('-')
		}
		builder.WriteRune('|')
		for range lengths.typeLength + 1 {
			builder.WriteRune('-')
		}
		if lengths.defaultLength > 0 {
			builder.WriteRune('|')
			for range lengths.defaultLength + 3 {
				builder.WriteRune('-')
			}
		}
		builder.WriteString("|--------------\n")
	}

	for _, field := range str.fields {
		if strings.Contains(field.doc, "Deprecated:") {
			continue
		}
		builder.WriteString(field.name)
		for range lengths.fieldLength - len(field.name) + 1 {
			builder.WriteRune(' ')
		}
		builder.WriteRune('|')
		builder.WriteString(field.fType)
		for range lengths.typeLength - len(field.fType) + 1 {
			builder.WriteRune(' ')
		}
		builder.WriteRune('|')
		if lengths.defaultLength > 0 {
			builder.WriteString(field.defaultVal)
			for range lengths.defaultLength - len(field.defaultVal) + 3 {
				builder.WriteRune(' ')
			}
			builder.WriteRune('|')
		}
		builder.WriteString(strings.ReplaceAll(field.doc, "\n", " "))
		builder.WriteRune('\n')
	}
}

func main() {
	log.SetFlags(0)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s /path/to/gochan/", os.Args[0])
	}
	// cfgDir := os.Args[1]
	gochanRoot := os.Args[1]
	cfgDir := path.Join(gochanRoot, "pkg/config")
	configStructs, err := docStructs(cfgDir)
	if err != nil {
		log.Fatalf("Error parsing package in %s: %s", cfgDir, err)
	}

	geoipDir := path.Join(gochanRoot, "pkg/posting/geoip")
	geoipStructs, err := docStructs(geoipDir)
	if err != nil {
		log.Fatalf("Error parsing package in %s: %s", geoipDir, err)
	}

	var builder strings.Builder
	builder.WriteString("# Configuration\n\n")

	cfgColumnLengths := columnLengths{}
	configStructsArray := make([]structType, 0, len(configStructs))
	for _, str := range compositeStructTypes {
		configStructsArray = append(configStructsArray, configStructs[str])
	}
	cfgColumnLengths.setLengths(configStructsArray...)

	for s, structName := range compositeStructTypes {
		str := configStructs[structName]
		fieldsAsMarkdownTable(&str, &builder, false, s == 0, &cfgColumnLengths)
	}
	builder.WriteString("\n")
	for _, structName := range explicitlyNamedStructTypes {
		str := configStructs[structName]
		if str.name == "" {
			fmt.Println(structName, str)
			continue
		}
		fieldsAsMarkdownTable(&str, &builder, true, true, nil)
		builder.WriteString("\n")
	}

	country := geoipStructs["Country"]
	country.name = "geoip.Country"
	cfgColumnLengths.setLengths(country)
	fieldsAsMarkdownTable(&country, &builder, true, true, &cfgColumnLengths)
	log.Println(builder.String())

	// fi, err := os.OpenFile("cfgdoc.md", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	// if err != nil {
	// 	log.Fatalln("Unable to open cfgdoc.md:", err.Error())
	// }
	// if _, err = io.WriteString(fi, builder.String()); err != nil {
	// 	fi.Close()
	// 	log.Fatalln("Unable to write to cfgdoc.md:", err.Error())
	// }
	// if err = fi.Close(); err != nil {
	// 	log.Fatalln("Unable to close cfgdoc.md:", err.Error())
	// }
	// log.Println("Wrote to cfgdoc.md successfully")
}

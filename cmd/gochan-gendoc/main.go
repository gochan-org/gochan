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
		ast.Inspect(file, func(n ast.Node) bool {
			switch t := n.(type) {
			case *ast.TypeSpec:
				structName = t.Name.String()
				log.Println(structName, "doc:", t.Doc)
				structDoc = t.Doc.Text()
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
						if strings.HasPrefix(line, "default: ") {
							fieldT.defaultVal = strings.Replace(line, "default: ", "", 1)
							break
						}
					}

					switch tt := field.Type.(type) {
					case *ast.Ident:
						fieldT.fType = tt.Name
					case *ast.ArrayType:
						fieldT.fType = fmt.Sprint(tt.Elt)
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
				// log.Println("gendecl:", t.Doc)
			}
			return true
		})

		return nil
	})
	return structMap, err
}

func printFields(str *structType) {
	for _, field := range str.fields {
		log.Println(field)
		if field.composite != "" {
			log.Println("")
		}
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
	structs, err := docStructs(cfgDir)
	if err != nil {
		log.Fatalf("Error parsing package in %s: %s", cfgDir, err)
	}
	var builder strings.Builder

	longestName := 4
	longestType := 0
	longestDefault := 5
	for _, str := range structs {
		for _, field := range str.fields {
			if len(field.name) > longestName {
				longestName = len(field.name)
			}
			if len(field.fType) > longestType {
				longestType = len(field.fType)
			}
			if len(field.defaultVal) > longestDefault {
				longestDefault = len(field.defaultVal)
			}
		}
	}

	builder.WriteString("# Configuration\n\nKey")
	for range longestName - 2 {
		builder.WriteRune(' ')
	}
	builder.WriteString("|Type")
	for range longestType - 3 {
		builder.WriteRune(' ')
	}
	builder.WriteString("|Default")
	for range longestDefault - 4 {
		builder.WriteRune(' ')
	}
	builder.WriteString("|Info\n")
	for range longestName + 1 {
		builder.WriteRune('-')
	}
	builder.WriteRune('|')
	for range longestType + 1 {
		builder.WriteRune('-')
	}
	builder.WriteRune('|')
	for range longestDefault + 3 {
		builder.WriteRune('-')
	}
	builder.WriteString("|----------\n")
	for _, str := range structs {
		// log.Println("struct name:", str.name)
		if str.doc != "" {
			log.Println("struct doc:", str.doc)
		}
		for _, field := range str.fields {
			builder.WriteString(field.name)
			for range longestName - len(field.name) + 1 {
				builder.WriteRune(' ')
			}
			builder.WriteRune('|')
			builder.WriteString(field.fType)
			for range longestType - len(field.fType) + 1 {
				builder.WriteRune(' ')
			}
			builder.WriteRune('|')
			builder.WriteString(field.defaultVal)
			for range longestDefault - len(field.defaultVal) + 3 {
				builder.WriteRune(' ')
			}
			builder.WriteRune('|')
			builder.WriteString(strings.ReplaceAll(field.doc, "\n", " "))
			builder.WriteRune('\n')
		}
	}
	log.Println(builder.String())
	// log.Println("Structs:", structs)
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

package gen

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/alecthomas/go-thrift/parser"
)

// Arg is a named function argument.
type Arg struct {
	Name string
	Type string
}

// Method is a method call on a service.
type Method struct {
	Name         string
	Request      []*Arg
	ResponseType string // empty string => void
}

// Service is a thrift service.
type Service struct {
	Name    string
	Methods []*Method
}

// Thrift is a single thrift file.
type Thrift struct {
	Package       string   // The go package to write the generated file to.
	ThriftPackage string   // The package of the thrift gen code.
	ThriftImport  string   // The import path to the thrift gen code.
	Imports       []string // All imports used in the servies.
	Services      []*Service
}

// ArgDeclarations returns the declarations for all args.
func (m *Method) ArgDeclarations() string {
	results := make([]string, len(m.Request))
	for i, argument := range m.Request {
		results[i] = argument.Name + " " + argument.Type
	}
	return strings.Join(results, ", ")
}

// Args returns a list of all args, without types.
func (m *Method) Args() string {
	results := make([]string, len(m.Request))
	for i, argument := range m.Request {
		results[i] = argument.Name
	}
	return strings.Join(results, ", ")
}

// Parser parses thrift files into a Thrift object.  It is not threadsafe.
type Parser struct {
	pkg string

	// these only exist during a parse run
	imports  map[string]bool
	thrift   map[string]*parser.Thrift
	mainFile string
}

// NewParser creates a new parser.  pkg is the package to write to.
func NewParser(pkg string) *Parser {
	return &Parser{
		pkg: pkg,
	}
}

// Parse parses the given thrift file.
func (p *Parser) Parse(thriftFile string) (*Thrift, error) {
	var err error
	p.thrift, p.mainFile, err = parser.New().ParseFile(thriftFile)
	if err != nil {
		return nil, err
	}
	p.imports = map[string]bool{p.absPathToImport(p.mainFile): true} // clean imports for next time
	services := []*Service{}
	for _, service := range p.thrift[p.mainFile].Services {
		services = append(services, p.parseService(service))
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	imports := p.getUsedImports()
	return &Thrift{
		Package:       p.pkg,
		Services:      services,
		ThriftImport:  p.absPathToImport(p.mainFile),
		ThriftPackage: p.absPathToPkg(p.mainFile),
		Imports:       imports,
	}, nil
}

func (p *Parser) parseService(service *parser.Service) *Service {
	//TODO(mdee) handle extends.
	methods := make([]*Method, 0, len(service.Methods))
	for _, method := range service.Methods {
		methods = append(methods, p.parseMethod(method))
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	return &Service{Name: titleCase(service.Name), Methods: methods}
}

func (p *Parser) parseMethod(method *parser.Method) *Method {
	returnType := ""
	ret := method.ReturnType
	if ret != nil {
		returnType = p.parseType(ret)
	}
	args := make([]*Arg, len(method.Arguments))
	for i, arg := range method.Arguments {
		typeName := p.parseType(arg.Type)
		args[i] = &Arg{Name: arg.Name, Type: typeName}
	}

	return &Method{Name: titleCase(method.Name), ResponseType: returnType, Request: args}
}

// absPathToImport converts an absolute path into the go import path for that file.
func (p *Parser) absPathToImport(path string) string {
	return strings.Replace(p.thrift[path].Namespaces["go"], ".", "/", -1)
}

// absPathToPkg converts an absolute path into the go package for that file.
func (p *Parser) absPathToPkg(path string) string {
	imported := p.absPathToImport(path)
	p.imports[imported] = true
	return imported[strings.LastIndex(p.absPathToImport(path), "/")+1:]
}

// typeToAbsPath converts a thrift type into the absolute path of the file it came from.
func (p *Parser) typeToAbsPath(typeName string) string {
	split := strings.Split(typeName, ".")
	typeFile := p.mainFile
	if len(split) == 2 {
		typeFile = p.thrift[p.mainFile].Includes[split[0]]
	}
	return typeFile
}

// typeToPackage converts a type into the go package that it is imported with.
func (p *Parser) typeToPackage(typeName string) string {
	imported := p.absPathToPkg(p.typeToAbsPath(typeName))
	return imported
}

// getUsedImports returns the imports needed for all the types in service signatures.
func (p *Parser) getUsedImports() []string {
	imports := make([]string, 0, len(p.imports))
	for imported := range p.imports {
		imports = append(imports, imported)
	}
	sort.Strings(imports)
	return imports
}

// parseType convers the type into its go representation.
func (p *Parser) parseType(parserType *parser.Type) string {
	if parserType.ValueType != nil {
		switch parserType.Name {
		case "list":
			return fmt.Sprintf("[]%s", p.parseType(parserType.ValueType))
		case "map":
			return fmt.Sprintf("map[%s]%s",
				p.parseType(parserType.KeyType),
				p.parseType(parserType.ValueType))
		case "set":
			return fmt.Sprintf("map[%s]bool", p.parseType(parserType.ValueType))
		default:
			panic("unknown type name")
		}
	} else {
		name := p.parseName(parserType.Name)
		return name
	}
}

var (
	primitiveTypes = map[string]string{
		"string": "string",
		"bool":   "bool",
		"i16":    "int16",
		"i32":    "int32",
		"i64":    "int64",
		"double": "float64",
		"byte":   "int8",
		"binary": "[]byte",
	}
)

// parseName converts a thrift base type name into its go equivalent.
// TODO handle enums, optional
func (p *Parser) parseName(typeName string) string {
	if val, ok := primitiveTypes[typeName]; ok {
		return val
	}
	if strings.Contains(typeName, ".") {
		split := strings.Split(typeName, ".")
		ns := p.typeToPackage(typeName)

		return fmt.Sprintf("*%s.%s", ns, titleCase(split[1]))
	}
	return fmt.Sprintf("*%s.%s", p.typeToPackage(typeName), titleCase(typeName))
}

// titleCase converts a name into UpperCamelCase. It takes into account a subset of edge cases from
// https://github.com/apache/thrift/blob/master/compiler/cpp/src/thrift/generate/t_go_generator.cc.
// THIS IS OVERSIMPLIFIED AND WILL FAIL FOR CERTAIN EDGE CASES
func titleCase(s string) string {
	prev := '_'
	s = fixForInitialismCase(s)
	titleCased := strings.Map(
		func(r rune) rune {
			if r == '_' {
				prev = r
				return -1
			}
			if prev == '_' {
				prev = r
				return unicode.ToUpper(r)
			}
			prev = r
			return r
		}, s)
	// special cases when a struct name ends in 'Args'/'Result', leads to go name having '_' appended
	// https://github.com/apache/thrift/blob/master/compiler/cpp/src/thrift/generate/t_go_generator.cc#L495
	if (len(titleCased) >= 4 && titleCased[len(titleCased)-4:] == "Args") ||
		(len(titleCased) >= 6 && s[len(titleCased)-6:] == "Result") {
		titleCased = titleCased + "_"
	}
	return titleCased
}

// lifted from https://github.com/apache/thrift/blob/master/compiler/cpp/src/thrift/generate/t_go_generator.cc#L671
var initialismList = []string{
	"api",
	"ascii",
	"cpu",
	"css",
	"dns",
	"eof",
	"guid",
	"html",
	"http",
	"https",
	"id",
	"ip",
	"json",
	"lhs",
	"qps",
	"ram",
	"rhs",
	"rpc",
	"sla",
	"smtp",
	"ssh",
	"tcp",
	"tls",
	"ttl",
	"udp",
	"ui",
	"uid",
	"uuid",
	"uri",
	"url",
	"utf8",
	"vm",
	"xml",
	"xsrf",
	"xss",
}

// fixForInitialismCase handles cases in the supplied string where snakecase initialism is given and uppercases it
// as the go gen will do https://github.com/apache/thrift/blob/master/compiler/cpp/src/thrift/generate/t_go_generator.cc#L449
func fixForInitialismCase(s string) string {
	for _, initialism := range initialismList {
		parts := strings.Split(s, "_")
		for i, part := range parts {
			if part == initialism {
				parts[i] = strings.ToUpper(initialism)
			}
		}
		s = strings.Join(parts, "_")
	}
	return s
}

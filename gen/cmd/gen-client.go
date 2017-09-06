package main

// gen-client generates an intermediate go handler for common rpc client behavior around a thrift generated
// client

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/oscarhealth/thriftgowrap/gen"
)

var (
	thriftFile  = flag.String("thrift", "", "Thrift file to generate clients for, relative to $DATA_REPO")
	outFileName = flag.String("out", "", "Location to write the output to")
)

const goTemplate = `{{- $tPkg := .ThriftPackage -}}
// Package {{.Package}} wraps {{.ThriftImport}} with RPC-specific logic.
// @generated
package {{.Package}}

import (
	"git.apache.org/thrift.git/lib/go/thrift"
{{range $import := .Imports}}
	"{{$import}}"
{{- end}}
	"utils/rpc"
)
{{range $service := .Services}}
// {{$service.Name}}RPCClient implements {{$service.Name}} with RPC-specific logic.
type {{$service.Name}}RPCClient rpc.Client

// New{{$service.Name}}RPCClient returns a new {{$service.Name}}RPCClient.
func New{{$service.Name}}RPCClient(transportFactory rpc.TransportFactory, options ...rpc.ClientOption) *{{$service.Name}}RPCClient {
	client := rpc.NewClient(transportFactory, options...)
	return (*{{$service.Name}}RPCClient)(client)
}

// getThriftClient returns a {{$tPkg}}.{{$service.Name}}Client and thrift.TTransport.
// The caller is expected to close the transport.
func (c *{{$service.Name}}RPCClient) getThriftClient() ({{$tPkg}}.{{$service.Name}}, thrift.TTransport, error) {
	transport, protocolFactory, err := c.TransportFactory.GetTransport()
	if err != nil {
		return nil, nil, err
	}
	return {{$tPkg}}.New{{$service.Name}}ClientFactory(transport, protocolFactory), transport, nil
}
{{- range $method := $service.Methods}}

// {{$method.Name}} wraps the underlying method.
func (c *{{$service.Name}}RPCClient) {{$method.Name}}({{$method.ArgDeclarations}}) (
	{{- if $method.ResponseType}}resp {{$method.ResponseType}}, {{end}}err error) {
	err = c.Retrier.Do(func() error {
		client, transport, innerErr := c.getThriftClient()
		if innerErr != nil {
			return innerErr
		}
		defer transport.Close()

		{{if $method.ResponseType}}resp, {{end}}err = client.{{$method.Name}}({{$method.Args}})
		return err
	})

	return
}
{{- end}}
{{end -}}
`

func generate(args *gen.Thrift, w io.Writer) error {
	t := template.Must(template.New("template").Parse(goTemplate))
	if err := t.Execute(w, args); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	fileName := *thriftFile
	if !filepath.IsAbs(fileName) {
		var err error
		fileName, err = filepath.Abs(fileName)
		if err != nil {
			log.Fatal(err)
		}
	}
	goThrift, err := gen.NewParser(os.Getenv("GOPACKAGE")).Parse(fileName)
	if err != nil {
		log.Fatal(err)
	}
	usedFileName := *outFileName
	println(usedFileName)
	if usedFileName == "" {
		baseFile := filepath.Base(fileName)
		usedFileName = baseFile[0:strings.LastIndex(baseFile, "thrift")] + "go"
	}
	outfile, err := os.Create(usedFileName)
	if err != nil {
		log.Fatal(err)
	}
	err = generate(goThrift, outfile)
	if err != nil {
		log.Fatal(err)
	}
}

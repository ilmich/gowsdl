// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"
)

// GoWSDL defines the struct for WSDL generator.
type GoWSDL struct {
	loc                   *Location
	rawWSDL               []byte
	pkg                   string
	ignoreTLS             bool
	makePublicFn          func(string) string
	wsdl                  *WSDL
	resolvedXSDExternals  map[string]bool
	currentRecursionLevel uint8
	currentNamespace      string
	resolveCollisions     map[string]string
}

// Method setNS sets (and returns) the currently active XML namespace.
func (g *GoWSDL) setNS(ns string) string {
	g.currentNamespace = ns
	return ns
}

// Method setNS returns the currently active XML namespace.
func (g *GoWSDL) getNS() string {
	return g.currentNamespace
}

var cacheDir = filepath.Join(os.TempDir(), "gowsdl-cache")

func init() {
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		log.Println("Create cache directory", "error", err)
		os.Exit(1)
	}
}

var timeout = time.Duration(30 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func downloadFile(url string, ignoreTLS bool) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: ignoreTLS,
		},
		Dial: dialTimeout,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Received response code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// NewGoWSDL initializes WSDL generator.
func NewGoWSDL(file, pkg string, ignoreTLS bool, exportAllTypes bool) (*GoWSDL, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, errors.New("WSDL file is required to generate Go proxy")
	}

	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		pkg = "generated_from_wsdl"
	}
	makePublicFn := func(id string) string { return id }
	if exportAllTypes {
		makePublicFn = makePublic
	}

	r, err := ParseLocation(file)
	if err != nil {
		return nil, err
	}

	return &GoWSDL{
		loc:          r,
		pkg:          pkg,
		ignoreTLS:    ignoreTLS,
		makePublicFn: makePublicFn,
	}, nil
}

// Start starts the GoWSDL code generation process. It unmarshals the WSDL document, resolves complex type name collisions,
// and generates the necessary code for types, operations, and server based on the WSDL structure. The output is returned as a
// map of byte slices, where the keys represent different code files and the values contain the corresponding generated code.
// In case of any error during the generation process, an error is returned.
func (g *GoWSDL) Start() (map[string][]byte, error) {
	gocode := make(map[string][]byte)
	var mu sync.Mutex

	g.resolveCollisions = make(map[string]string)

	err := g.unmarshal()
	if err != nil {
		return nil, err
	}

	// Resolve complex type name collisions
	seen := map[string]int{}
	for _, schema := range g.wsdl.Types.Schemas {
		for _, complexType := range schema.ComplexTypes {
			seen[complexType.Name]++
		}
		for _, simpleType := range schema.SimpleType {
			seen[simpleType.Name]++
		}
	}

	// Filter out non-colliding names
	for k, v := range seen {
		if v < 2 {
			delete(seen, k)
		}
	}

	// Log collisions and update colliding names
	for i := range g.wsdl.Types.Schemas {
		schema := g.wsdl.Types.Schemas[i]
		for j := range schema.ComplexTypes {
			complexType := schema.ComplexTypes[j]
			if num := seen[complexType.Name]; num > 0 {
				org := complexType.Name
				update := fmt.Sprintf("%s%d", org, num)
				g.resolveCollisions[fmt.Sprintf("%s/%s", schema.TargetNamespace, org)] = update
				complexType.Name = update
				seen[org]--
				log.Printf("Collision detected: ComplexType '%s' renamed to '%s' in namespace '%s'", org, update, schema.TargetNamespace)
			}
		}
		for j := range schema.SimpleType {
			simpleType := schema.SimpleType[j]
			if num := seen[simpleType.Name]; num > 0 {
				org := simpleType.Name
				update := fmt.Sprintf("%s%d", org, num)
				g.resolveCollisions[fmt.Sprintf("%s/%s", schema.TargetNamespace, org)] = update
				simpleType.Name = update
				seen[org]--
				log.Printf("Collision detected: SimpleType '%s' renamed to '%s' in namespace '%s'", org, update, schema.TargetNamespace)
			}
		}
	}

	// Process WSDL nodes
	for _, schema := range g.wsdl.Types.Schemas {
		newTraverser(schema, g.wsdl.Types.Schemas, g.resolveCollisions).traverse()
	}

	var wg sync.WaitGroup
	var genErr error

	runWithMutex := func(key string, genFunc func() ([]byte, error)) {
		defer wg.Done()
		code, err := genFunc()
		if err != nil {
			mu.Lock()
			genErr = err
			mu.Unlock()
			return
		}
		mu.Lock()
		gocode[key] = code
		mu.Unlock()
	}

	wg.Add(3)
	go runWithMutex("types", g.genTypes)
	go runWithMutex("operations", g.genOperations)
	go runWithMutex("server", g.genServer)

	wg.Wait()

	if genErr != nil {
		return nil, genErr
	}

	// Generate header code
	gocode["header"], err = g.genHeader()
	if err != nil {
		return nil, err
	}

	// Generate server header code
	gocode["server_header"], err = g.genServerHeader()
	if err != nil {
		return nil, err
	}

	// Generate server WSDL
	gocode["server_wsdl"] = []byte("var wsdl = `" + string(g.rawWSDL) + "`")

	return gocode, nil
}

func (g *GoWSDL) fetchFile(loc *Location) (data []byte, err error) {
	if loc.f != "" {
		log.Println("Reading", "file", loc.f)
		data, err = os.ReadFile(loc.f)
	} else {
		log.Println("Downloading", "file", loc.u.String())
		data, err = downloadFile(loc.u.String(), g.ignoreTLS)
	}
	return
}

func (g *GoWSDL) unmarshal() error {
	data, err := g.fetchFile(g.loc)
	if err != nil {
		return err
	}

	g.wsdl = new(WSDL)
	err = xml.Unmarshal(data, g.wsdl)
	if err != nil {
		return err
	}
	g.rawWSDL = data

	for _, schema := range g.wsdl.Types.Schemas {
		err = g.resolveXSDExternals(schema, g.loc)
		if err != nil {
			return err
		}
	}

	return nil
}

// resolveXSDExternals downloads and resolves external XSD imports and includes for a given XSD schema.
// It downloads the XSD file, parses it into an XSDSchema struct, and recursively resolves any additional imports or includes if present.
// The resolved schemas are then appended to the wsdl.Types.Schemas slice.
// It returns an error if there is any issue with downloading, parsing, or resolving the external XSDs.
func (g *GoWSDL) resolveXSDExternals(schema *XSDSchema, loc *Location) error {
	download := func(base *Location, ref string) error {
		location, err := base.Parse(ref)
		if err != nil {
			return err
		}
		schemaKey := location.String()
		if g.resolvedXSDExternals[schemaKey] {
			return nil
		}
		if g.resolvedXSDExternals == nil {
			g.resolvedXSDExternals = make(map[string]bool)
		}
		g.resolvedXSDExternals[schemaKey] = true

		var data []byte
		if data, err = g.fetchFile(location); err != nil {
			return err
		}

		newschema := new(XSDSchema)

		err = xml.Unmarshal(data, newschema)
		if err != nil {
			return err
		}

		// Risolvi ricorsivamente solo se ci sono ulteriori importazioni o inclusioni
		if len(newschema.Includes) > 0 || len(newschema.Imports) > 0 {
			err = g.resolveXSDExternals(newschema, location)
			if err != nil {
				return err
			}
		}

		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)

		return nil
	}

	// Scarica e risolvi le importazioni
	for _, impts := range schema.Imports {
		if impts.SchemaLocation == "" {
			log.Printf("[WARN] Don't know where to find XSD for %s", impts.Namespace)
			continue
		}

		if e := download(loc, impts.SchemaLocation); e != nil {
			return e
		}
	}

	// Scarica e risolvi le inclusioni
	for _, incl := range schema.Includes {
		if e := download(loc, incl.SchemaLocation); e != nil {
			return e
		}
	}

	return nil
}

func (g *GoWSDL) genTypes() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":                 toGoType,
		"stripns":                  stripns,
		"replaceReservedWords":     replaceReservedWords,
		"replaceAttrReservedWords": replaceAttrReservedWords,
		"normalize":                normalize,
		"makePublic":               g.makePublicFn,
		"makeFieldPublic":          makePublic,
		"comment":                  comment,
		"removeNS":                 removeNS,
		"goString":                 goString,
		"findNameByType":           g.findNameByType,
		"removePointerFromType":    removePointerFromType,
		"setNS":                    g.setNS,
		"getNS":                    g.getNS,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
	err := tmpl.Execute(data, g.wsdl.Types)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genOperations() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"normalize":            normalize,
		"makePublic":           g.makePublicFn,
		"makePrivate":          makePrivate,
		"findType":             g.findType,
		"findSOAPAction":       g.findSOAPAction,
		"findServiceAddress":   g.findServiceAddress,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("operations").Funcs(funcMap).Parse(opsTmpl))
	err := tmpl.Execute(data, g.wsdl.PortTypes)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genServer() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           g.makePublicFn,
		"findType":             g.findType,
		"findSOAPAction":       g.findSOAPAction,
		"findServiceAddress":   g.findServiceAddress,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("server").Funcs(funcMap).Parse(serverTmpl))
	err := tmpl.Execute(data, g.wsdl.PortTypes)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genHeader() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"normalize":            normalize,
		"makePublic":           g.makePublicFn,
		"findType":             g.findType,
		"comment":              comment,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("header").Funcs(funcMap).Parse(headerTmpl))
	err := tmpl.Execute(data, g.pkg)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genServerHeader() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           g.makePublicFn,
		"findType":             g.findType,
		"comment":              comment,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("server_header").Funcs(funcMap).Parse(serverHeaderTmpl))
	err := tmpl.Execute(data, g.pkg)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

var reservedWords = map[string]string{
	"break":       "break_",
	"default":     "default_",
	"func":        "func_",
	"interface":   "interface_",
	"select":      "select_",
	"case":        "case_",
	"defer":       "defer_",
	"go":          "go_",
	"map":         "map_",
	"struct":      "struct_",
	"chan":        "chan_",
	"else":        "else_",
	"goto":        "goto_",
	"package":     "package_",
	"switch":      "switch_",
	"const":       "const_",
	"fallthrough": "fallthrough_",
	"if":          "if_",
	"range":       "range_",
	"type":        "type_",
	"continue":    "continue_",
	"for":         "for_",
	"import":      "import_",
	"return":      "return_",
	"var":         "var_",
}

var reservedWordsInAttr = map[string]string{
	"break":       "break_",
	"default":     "default_",
	"func":        "func_",
	"interface":   "interface_",
	"select":      "select_",
	"case":        "case_",
	"defer":       "defer_",
	"go":          "go_",
	"map":         "map_",
	"struct":      "struct_",
	"chan":        "chan_",
	"else":        "else_",
	"goto":        "goto_",
	"package":     "package_",
	"switch":      "switch_",
	"const":       "const_",
	"fallthrough": "fallthrough_",
	"if":          "if_",
	"range":       "range_",
	"type":        "type_",
	"continue":    "continue_",
	"for":         "for_",
	"import":      "import_",
	"return":      "return_",
	"var":         "var_",
	"string":      "astring",
}

var specialCharacterMapping = map[string]string{
	"+": "Plus",
	"@": "At",
}

// Replaces Go reserved keywords to avoid compilation issues
func replaceReservedWords(identifier string) string {
	value := reservedWords[identifier]
	if value != "" {
		return value
	}
	return normalize(identifier)
}

// Replaces Go reserved keywords to avoid compilation issues
func replaceAttrReservedWords(identifier string) string {
	value := reservedWordsInAttr[identifier]
	if value != "" {
		return value
	}
	return normalize(identifier)
}

// Normalizes value to be used as a valid Go identifier, avoiding compilation issues
func normalize(value string) string {
	for k, v := range specialCharacterMapping {
		value = strings.ReplaceAll(value, k, v)
	}

	mapping := func(r rune) rune {
		if r == '.' || r == '-' {
			return '_'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return -1
	}

	return strings.Map(mapping, value)
}

func goString(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

var xsd2GoTypes = map[string]string{
	"string":             "string",
	"token":              "string",
	"float":              "float32",
	"double":             "float64",
	"decimal":            "float64",
	"integer":            "int32",
	"int":                "int32",
	"short":              "int16",
	"byte":               "int8",
	"long":               "int64",
	"boolean":            "bool",
	"datetime":           "string",
	"date":               "string",
	"time":               "string",
	"base64binary":       "[]byte",
	"hexbinary":          "[]byte",
	"unsignedint":        "uint32",
	"nonnegativeinteger": "uint32",
	"unsignedshort":      "uint16",
	"unsignedbyte":       "byte",
	"unsignedlong":       "uint64",
	"anytype":            "AnyType",
	"ncname":             "NCName",
	"anyuri":             "AnyURI",
	// customz.
	"sdpstring":     "string",
	"sdpboolean":    "bool",
	"sdpbyte":       "int8",
	"sdpbigdecimal": "float64",
	"sdplong":       "int64",
	"sdpinteger":    "int32",
	"sdpbiginteger": "int32",
	"sdpfloat":      "float32",
	"sdpdouble":     "float64",
	"sdpshort":      "int16",
	"sdpdate":       "string",
	"sdptime":       "string",
	"sdpdatetime":   "string",
	"timestamp":     "int64",
}

func removeNS(xsdType string) string {
	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")

	if len(r) == 2 {
		return r[1]
	}

	return r[0]
}

func toGoType(xsdType string, nillable bool) string {
	// Rimuove il namespace, ad esempio xsd:string diventa string
	r := strings.Split(xsdType, ":")

	t := r[0]
	if len(r) == 2 {
		t = r[1]
	}

	// Cerca il tipo nel dizionario `xsd2GoTypes`
	value := xsd2GoTypes[strings.ToLower(t)]

	if value != "" {
		if nillable {
			value = "*" + value
		}
		return value
	}

	// Se il tipo non è trovato, ritorna un tipo pubblico generico Go
	return "*" + replaceReservedWords(makePublic(t))
}
func removePointerFromType(goType string) string {
	return regexp.MustCompile("^\\s*\\*").ReplaceAllLiteralString(goType, "")
}

// Given a message, finds its type.
//
// I'm not very proud of this function but
// it works for now and performance doesn't
// seem critical at this point
func (g *GoWSDL) findType(message string) string {
	message = stripns(message)

	for _, msg := range g.wsdl.Messages {
		if msg.Name != message {
			continue
		}

		// Assumes document/literal wrapped WS-I
		if len(msg.Parts) == 0 {
			// Message does not have parts. This could be a Port
			// with HTTP binding or SOAP 1.2 binding, which are not currently
			// supported.
			log.Printf("[WARN] %s message doesn't have any parts, ignoring message...", msg.Name)
			continue
		}

		part := msg.Parts[0]
		if part.Type != "" {
			return stripns(part.Type)
		}

		elRef := stripns(part.Element)

		for _, schema := range g.wsdl.Types.Schemas {
			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					if el.Type != "" {
						return stripns(el.Type)
					}
					return el.Name
				}
			}
		}
	}
	return ""
}

// Given a type, check if there's an Element with that type, and return its name.
func (g *GoWSDL) findNameByType(name string) string {
	return newTraverser(nil, g.wsdl.Types.Schemas, g.resolveCollisions).findNameByType(name)
}

// TODO(c4milo): Add support for namespaces instead of striping them out
// TODO(c4milo): improve runtime complexity if performance turns out to be an issue.
func (g *GoWSDL) findSOAPAction(operation, portType string) string {
	for _, binding := range g.wsdl.Binding {
		if strings.ToUpper(stripns(binding.Type)) != strings.ToUpper(portType) {
			continue
		}

		for _, soapOp := range binding.Operations {
			if soapOp.Name == operation {
				return soapOp.SOAPOperation.SOAPAction
			}
		}
	}
	return ""
}

func (g *GoWSDL) findServiceAddress(name string) string {
	for _, service := range g.wsdl.Service {
		for _, port := range service.Ports {
			if port.Name == name {
				return port.SOAPAddress.Location
			}
		}
	}
	return ""
}

// TODO(c4milo): Add namespace support instead of stripping it
func stripns(xsdType string) string {
	r := strings.Split(xsdType, ":")
	t := r[0]

	if len(r) == 2 {
		t = r[1]
	}

	return t
}

func makePublic(identifier string) string {
	if isBasicType(identifier) {
		return identifier
	}
	if identifier == "" {
		return "EmptyString"
	}
	field := []rune(identifier)
	if len(field) == 0 {
		return identifier
	}

	field[0] = unicode.ToUpper(field[0])
	return string(field)
}

var basicTypes = map[string]string{
	"string":      "string",
	"float32":     "float32",
	"float64":     "float64",
	"int":         "int",
	"int8":        "int8",
	"int16":       "int16",
	"int32":       "int32",
	"int64":       "int64",
	"bool":        "bool",
	"time.Time":   "time.Time",
	"[]byte":      "[]byte",
	"byte":        "byte",
	"uint16":      "uint16",
	"uint32":      "uint32",
	"uinit64":     "uint64",
	"interface{}": "interface{}",
}

func isBasicType(identifier string) bool {
	if _, exists := basicTypes[identifier]; exists {
		return true
	}
	return false
}

func makePrivate(identifier string) string {
	field := []rune(identifier)
	if len(field) == 0 {
		return identifier
	}

	field[0] = unicode.ToLower(field[0])
	return string(field)
}

func comment(text string) string {
	lines := strings.Split(text, "\n")

	var output string
	if len(lines) == 1 && lines[0] == "" {
		return ""
	}

	// Helps to determine if there is an actual comment without screwing newlines
	// in real comments.
	hasComment := false

	for _, line := range lines {
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line != "" {
			hasComment = true
		}
		output += "\n// " + line
	}

	if hasComment {
		return output
	}
	return ""
}

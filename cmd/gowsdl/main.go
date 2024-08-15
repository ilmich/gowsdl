// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
/*

Gowsdl generates Go code from a WSDL file.

This project is originally intended to generate Go clients for WS-* services.

Usage: gowsdl [options] myservice.wsdl
  -o string
        File where the generated code will be saved (default "myservice.go")
  -p string
        Package under which code will be generated (default "myservice")
  -v    Shows gowsdl version

Features

Supports only Document/Literal wrapped services, which are WS-I (http://ws-i.org/) compliant.

Attempts to generate idiomatic Go code as much as possible.

Supports WSDL 1.1, XML Schema 1.0, SOAP 1.1.

Resolves external XML Schemas

Supports providing WSDL HTTP URL as well as a local WSDL file.

Not supported

UDDI.

TODO

Add support for filters to allow the user to change the generated code.

If WSDL file is local, resolve external XML schemas locally too instead of failing due to not having a URL to download them from.

Resolve XSD element references.

Support for generating namespaces.

Make code generation agnostic so generating code to other programming languages is feasible through plugins.

*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"

	gen "github.com/Andrea-Cavallo/gowsdl"
)

// Version and Name are initialized during compilation by go build.
var Version string
var Name string

// Flags
var (
	vers       = flag.Bool("v", false, "Shows gowsdl version")
	pkg        = flag.String("p", "myservice", "Package under which code will be generated")
	outFile    = flag.String("o", "myservice.go", "File where the generated code will be saved")
	dir        = flag.String("d", "./", "Directory under which package directory will be created")
	insecure   = flag.Bool("i", false, "Skips TLS Verification")
	makePublic = flag.Bool("make-public", true, "Make the generated types public/exported")
)

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	log.SetPrefix("👽  ")
}

func main() {
	// Setup command-line usage
	setupUsage()

	flag.Parse()

	// Show app version and exit
	if *vers {
		showVersion()
		return
	}

	// Check for sufficient arguments
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	wsdlPath := os.Args[len(os.Args)-1]

	// Prevent overwriting the WSDL file
	if *outFile == wsdlPath {
		log.Fatalln("Output file cannot be the same as the WSDL file")
	}

	// Load WSDL and generate code
	gowsdl, err := gen.NewGoWSDL(wsdlPath, *pkg, *insecure, *makePublic)
	handleError(err)

	gocode, err := gowsdl.Start()
	handleError(err)

	// Create the output directory if it doesn't exist
	outputDir := filepath.Join(*dir, *pkg)
	err = os.MkdirAll(outputDir, 0744)
	handleError(err)

	// Write the generated code to files
	writeGeneratedCode(outputDir, gocode, *outFile)

	log.Println("Daje 🚀🚀🚀️")
}

// setupUsage configures the usage message for the command-line tool.
func setupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] myservice.wsdl\n", os.Args[0])
		flag.PrintDefaults()
	}
}

// showVersion displays the version of the tool.
func showVersion() {
	log.Println(Name, Version)
	os.Exit(0)
}

// handleError is a utility function for error handling.
func handleError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// writeGeneratedCode handles writing the generated code to the output files.
func writeGeneratedCode(outputDir string, gocode map[string][]byte, outFile string) {
	// Write main generated code
	writeFile(filepath.Join(outputDir, outFile), formatCode(gocode["header"], gocode["types"], gocode["operations"], gocode["soap"]))

	// Write server generated code
	serverFileName := "server_" + outFile
	writeFile(filepath.Join(outputDir, serverFileName), formatCode(gocode["server_header"], gocode["server_wsdl"], gocode["server"]))
}

// writeFile creates a file and writes the content to it.
func writeFile(filePath string, content []byte) {
	file, err := os.Create(filePath)
	handleError(err)
	defer file.Close()

	_, err = file.Write(content)
	handleError(err)
}

// formatCode formats the source code using gofmt.
func formatCode(parts ...[]byte) []byte {
	data := new(bytes.Buffer)
	for _, part := range parts {
		data.Write(part)
	}

	formattedSource, err := format.Source(data.Bytes())
	if err != nil {
		log.Printf("Error formatting source, writing unformatted source: %v", err)
		return data.Bytes()
	}

	return formattedSource
}

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
// Flags
var (
	vers       = flag.Bool("v", false, "Mostra la versione di gowsdl")
	pkg        = flag.String("p", "mioServizio", "Pacchetto sotto il quale verrà generato il codice")
	outFile    = flag.String("o", "mioServizio.go", "File in cui verrà salvato il codice generato")
	dir        = flag.String("d", "./", "Directory in cui verrà creato il pacchetto")
	insecure   = flag.Bool("i", false, "Salta la verifica TLS")
	makePublic = flag.Bool("make-public", true, "Rende i tipi generati pubblici/esportati")
)

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	log.SetPrefix("👽  ")
}

func main() {
	// Configura l'uso della riga di comando
	setupUsage()

	flag.Parse()

	// Mostra la versione dell'applicazione ed esci
	if *vers {
		mostraVersione()
		return
	}

	// Controlla se ci sono abbastanza argomenti
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	wsdlPath := os.Args[len(os.Args)-1]

	// Impedisce di sovrascrivere il file WSDL
	if *outFile == wsdlPath {
		log.Fatalln("Il file di output non può essere lo stesso del file WSDL")
	}

	// Carica WSDL e genera il codice
	gowsdl, err := gen.NewGoWSDL(wsdlPath, *pkg, *insecure, *makePublic)
	gestisciErrore(err)

	gocode, err := gowsdl.Start()
	gestisciErrore(err)

	// Crea la directory di output se non esiste
	outputDir := filepath.Join(*dir, *pkg)
	err = os.MkdirAll(outputDir, 0744)
	gestisciErrore(err)

	// Scrivi il codice generato nei file
	scriviCodiceGenerato(outputDir, gocode, *outFile)

	log.Println("Daje 🚀🚀🚀️")
}

// setupUsage configura il messaggio di utilizzo per lo strumento da riga di comando.
func setupUsage() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [opzioni] mioServizio.wsdl\n", os.Args[0])
		flag.PrintDefaults()
	}
}

// mostraVersione visualizza la versione dello strumento.
func mostraVersione() {
	log.Println(Name, Version)
	os.Exit(0)
}

// gestisciErrore è una funzione di utilità per la gestione degli errori.
func gestisciErrore(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// scriviCodiceGenerato gestisce la scrittura del codice generato nei file di output.
func scriviCodiceGenerato(outputDir string, gocode map[string][]byte, outFile string) {
	// Scrivi il codice generato principale
	scriviFile(filepath.Join(outputDir, outFile), formattaCodice(gocode["header"], gocode["types"], gocode["operations"], gocode["soap"]))

	// Scrivi il codice generato del server
	nomeFileServer := "server_" + outFile
	scriviFile(filepath.Join(outputDir, nomeFileServer), formattaCodice(gocode["server_header"], gocode["server_wsdl"], gocode["server"]))
}

// scriviFile crea un file e vi scrive il contenuto.
func scriviFile(filePath string, content []byte) {
	file, err := os.Create(filePath)
	gestisciErrore(err)
	defer file.Close()

	_, err = file.Write(content)
	gestisciErrore(err)
}

// formattaCodice formatta il codice sorgente usando gofmt.
func formattaCodice(parts ...[]byte) []byte {
	data := new(bytes.Buffer)
	for _, part := range parts {
		data.Write(part)
	}

	codiceFormattato, err := format.Source(data.Bytes())
	if err != nil {
		log.Printf("Errore nella formattazione del sorgente, scrittura del sorgente non formattato: %v", err)
		return data.Bytes()
	}

	return codiceFormattato
}

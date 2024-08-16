# WSDL to Go

![wsdl](./soap.png)

## Italiano

Capace di generare codice Go direttamente da un file WSDL, per coloro che si trovano a dover affrontare l'arduo compito di lavorare con SOAP.
Implementazione customizzata.


## Installazione

* [Scarica la release](https://github.com/hooklift/gowsdl/releases)
* Scarica e compila localmente:
    * Go 1.15: `go get github.com/Andrea-Cavallo/gowsdl/...`
    * Go 1.20: `go install github.com/Andrea-Cavallo/gowsdl/cmd/gowsdl@latest`
* Installa con Homebrew: `brew install gowsdl`

## Obiettivi

* Generare codice Go idiomatico il più possibile
* Supportare solo servizi Document/Literal wrapped, che sono conformi a [WS-I](http://ws-i.org/)
* Supportare:
    * WSDL 1.1
    * XML Schema 1.0
    * SOAP 1.1
* Risolvere schemi XML esterni
* Supportare WSDL esterni e locali

## Avvertenze

* Tieni presente che il codice generato è solo un riflesso di com'è il WSDL. Se il tuo WSDL ha definizioni di tipi duplicati, il codice Go generato avrà gli stessi e potrebbe non compilare.

## Uso

```sh
Uso: gowsdl [opzioni] myservice.wsdl
  -o string
        File dove verrà salvato il codice generato (predefinito "myservice.go")
  -p string
        Pacchetto sotto il quale verrà generato il codice (predefinito "myservice")
  -i    Salta la verifica TLS
  -v    Mostra la versione di gowsdl
```

## Esempio

Per generare il codice Go da un file WSDL denominato `example.wsdl` e salvare l'output in `example.go`:

```sh
gowsdl -o example.go example.wsdl
```

## Esempio per richiamare il metodo client.callWithResponse..

- **ctx** - il context se non lo passi viene preso il background.ctx
- **soapAction** - l'azione soap una stringa
- **la request** che sara' poi parsata nell'envelope soap
- **custoMheaders** una mappa di headers custom
- **responseObject** - un puntatore alla response che ti aspetti ( se non sei sicuro della risposta potresti anche usare : &rawResponse,un puntatore a xml.RawMessage)
- **useTLS** se false skippi il tls
- **XlmnNamespace** altra mappa per le nostre esigenze vedi esempio


```go

// Crei una mappa rappresentante gli XLMNS ( adesso accetta quelli custom )
xmlNamespaces := map[string]string{
"soap": "http://schemas.xmlsoap.org/soap/envelope/",
"custom1": "http://custom1.example.it",
"custom2" : "http://custom2.example.it"
}

response, err := client.callWithResponse(
ctx,
soapAction,
request,
responseObj,
customHeaders,
useTLS,
xmlNamespaces,
)
```



## Ringraziamenti

Un sincero ringraziamento agli autori originali di questa libreria e a `IlMich` che mi ha illuminato su questi trick .
Questo fork è stato creato per soddisfare specifiche esigenze di sviluppo personali.


---

## English
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/hooklift/gowsdl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![GoDoc](https://godoc.org/github.com/hooklift/gowsdl?status.svg)](https://godoc.org/github.com/hooklift/gowsdl)
[![Build Status](https://travis-ci.org/hooklift/gowsdl.svg?branch=master)](https://travis-ci.org/hooklift/gowsdl)

Generate Go code from a WSDL file.

## Installation

* [Download the release](https://github.com/hooklift/gowsdl/releases)
* Download and build locally:
    * Go 1.15: `go get github.com/Andrea-Cavallo/gowsdl/...`
    * Go 1.20: `go install github.com/Andrea-Cavallo/gowsdl/cmd/gowsdl@latest`
* Install with Homebrew: `brew install gowsdl`

## Goals

* Generate idiomatic Go code as much as possible
* Support only Document/Literal wrapped services that are WS-I compliant
* Support:
    * WSDL 1.1
    * XML Schema 1.0
    * SOAP 1.1
* Resolve external XML schemas
* Support both external and local WSDLs

## Warnings

* Please note that the generated code is only a reflection of how the WSDL is. If your WSDL has duplicate type definitions, the generated Go code will have the same and may not compile.

## Usage

```sh
Usage: gowsdl [options] myservice.wsdl
  -o string
        The file where the generated code will be saved (default "myservice.go")
  -p string
        The package under which the code will be generated (default "myservice")
  -i    Skip TLS verification
  -v    Show the version of gowsdl
```

## Example

To generate Go code from a WSDL file named `example.wsdl` and save the output to `example.go`:

```sh
gowsdl -o example.go example.wsdl
```

## Acknowledgements

A sincere thank you to the original authors of this library. This fork was created to meet my own development needs.

## Contributing

If you wish to contribute to the project, follow these steps:

1. Fork the repository.
2. Create a branch for your changes (`git checkout -b feature/feature-name`).
3. Commit your changes (`git commit -am 'Added a new feature'`).
4. Push the branch (`git push origin feature/feature-name`).
5. Create a new Pull Request.

## Support

If you have questions or need help, join our chat on [Gitter](https://gitter.im/hooklift/gowsdl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) or check the documentation on [GoDoc](https://godoc.org/github.com/hooklift/gowsdl).

## License

This project is released under the MIT License. See the [LICENSE](LICENSE) file for details.

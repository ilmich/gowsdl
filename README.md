# WSDL to Go

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/hooklift/gowsdl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![GoDoc](https://godoc.org/github.com/hooklift/gowsdl?status.svg)](https://godoc.org/github.com/hooklift/gowsdl)
[![Build Status](https://travis-ci.org/hooklift/gowsdl.svg?branch=master)](https://travis-ci.org/hooklift/gowsdl)

Genera codice Go da un file WSDL.

## Installazione

* [Scarica la release](https://github.com/hooklift/gowsdl/releases)
* Scarica e compila localmente
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
gowsdl [opzioni] myservice.wsdl -o string File dove verrà salvato il codice generato (predefinito "myservice.go") -p string Pacchetto sotto il quale verrà generato il codice (predefinito "generated_wsdl..")
-i Salta la verifica TLS -v Mostra la versione di gowsdl






## Esempio

Per generare il codice Go da un file WSDL denominato `example.wsdl` e salvare l'output in `example.go`:

```sh
gowsdl -o example.go example.wsdl
```

## Contribuire

Se desideri contribuire al progetto, segui questi passaggi:

1. Fai un fork del repository.
2. Crea un branch per la tua modifica (`git checkout -b feature/nome-funzionalità`).
3. Fai commit delle tue modifiche (`git commit -am 'Aggiunta di una nuova funzionalità'`).
4. Fai push del branch (`git push origin feature/nome-funzionalità`).
5. Crea una nuova Pull Request.

## Supporto

Se hai domande o hai bisogno di aiuto, unisciti alla nostra chat su [Gitter](https://gitter.im/hooklift/gowsdl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) oppure consulta la documentazione su [GoDoc](https://godoc.org/github.com/hooklift/gowsdl).

## Licenza

Questo progetto è rilasciato sotto la licenza MIT. Consulta il file [LICENSE](LICENSE) per i dettagli.

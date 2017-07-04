package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

var BUILD_DATE string
var VERSION string

func main() {
	app := kingpin.New("havana", "Tool for building anything inside containers")
	app.Version(fmt.Sprintf("%s, built %s", VERSION, BUILD_DATE))

	AddMainCmd(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

package main

import (
	"flag"
	"fmt"
	"os"

	app "integration-testing/bifrost"

	"integration-testing/log"
	sca "integration-testing/scaffold"
)

func usage() {
	fmt.Printf("Usage: %v\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Println("command:")
	fmt.Println("  new\t:create new object")
	fmt.Println("  call\t:call object method")
	fmt.Println("  pcall\t:asyn call object method")
	fmt.Println("object:")
	for name, desc := range sca.ObjectsDesc() {
		fmt.Printf("  %s", name)
		if len(desc) > 0 {
			fmt.Printf("\t: %s", desc)
		}
		fmt.Println()
	}
}

func main() {
	fileName := flag.String("f", "data", "script data file name")
	i := flag.Bool("i", false, "interactive model")
	v := flag.Bool("v", false, "debug message")
	vv := flag.Bool("vv", false, "more debug message")
	flag.Usage = usage
	flag.Parse()

	if *v {
		log.SetLevel(log.Debug1)
	} else if *vv {
		log.SetLevel(log.Debug2)
	}

	app.Init()
	defer app.Clear()

	if *i {
		sca.Interactive()
		return
	}

	scens, err := sca.LoadScens(*fileName)
	if err != nil {
		log.Fatalf("load scen failed %s", err)
	}
	sca.Run(scens)
}

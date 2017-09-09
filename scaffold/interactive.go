package scaffold

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func InteractiveUsage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\tloadconf pakcet-name conf-file\n")
	fmt.Printf("\tloadscen scen-file-name\n")
	fmt.Printf("\tlistscen [id]\n")
	fmt.Printf("\trun scen-id\n")
	fmt.Printf("\treset\n")
	fmt.Printf("\tnew class-name id-prefix\n")
	fmt.Printf("\tcall id-regex method param...\n")
	fmt.Printf("\tpcall id-regex method param...\n")
	fmt.Printf("\tlistobj\n")
	fmt.Printf("\thistory\n")
}

func Interactive() {
	fmt.Println("Welcome into test interactive model")
	var err error
	var conf Config
	var scens []*Scenario
	var ss []string
	ctl := NewControl()
	defer ctl.Stop()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("> ")
		token, detail := readCommand(reader)
		switch token[0] {
		case "loadconf":
			if len(token) != 3 {
				fmt.Println("loadconf packet-name conf-file")
				continue
			}
			confsMutex.Lock()
			loadFunc, ok := confs[token[1]]
			confsMutex.Unlock()
			if !ok {
				fmt.Printf("not found load config func for '%s'\n", token[1])
				continue
			}
			if conf, err = loadFunc(token[2]); err != nil {
				fmt.Printf("load conf failed %s\n", err)
				continue
			}
			fmt.Printf("Load conf %s Success\n", token[2])
		case "loadscen":
			if len(token) != 2 {
				fmt.Printf("\tloadscen scen-file-name\n")
				continue
			}
			if scens, err = LoadScens(token[1]); err != nil {
				fmt.Printf("load scen failed %s\n", err)
				continue
			}
			fmt.Printf("Load scen %s Success\n", token[1])
		case "listscen":
			if len(scens) == 0 {
				fmt.Printf("NOT load scen\n")
				continue
			}
			if len(token) == 1 {
				for i := range scens {
					fmt.Printf("#%d %s\n", i, scens[i].String())
				}
				continue
			}
			id, _ := strconv.Atoi(token[1])
			if id >= len(scens) {
				fmt.Printf("Invalid id for scens, id in [0, %d]\n", len(scens))
				continue
			}
			fmt.Printf("#%d %s\n", id, scens[id].String())
			for _, s := range scens[id].Scripts() {
				fmt.Printf("  %s\n", s.String())
			}
		case "r", "run":
			if len(token) != 2 {
				fmt.Printf("\trun scen-id\n")
				continue
			}
			id, _ := strconv.Atoi(token[1])
			if id >= len(scens) {
				fmt.Printf("Invalid id for scens, id in [0, %d]\n", len(scens))
				continue
			}
			if conf == nil {
				fmt.Printf("not set config\n")
				continue
			}
			fmt.Println(scens[id].String())
			ctl := NewControl()
			if err = ctl.Run(scens[id]); err != nil {
				fmt.Printf(" |- run failed %s\n", err)
				continue
			}
			fmt.Printf("run scen Success\n")
		case "reset":
			conf = nil
			scens = nil
			if ctl != nil {
				ctl.Stop()
				ctl = NewControl()
			}
			ss = nil
		case "new", "call", "pcall":
			if conf == nil {
				fmt.Printf("not set config\n")
				continue
			}
			var script *Script
			if script, err = ParseScript(detail); err != nil {
				fmt.Printf("parse script faild %s\n", err)
				continue
			}
			if err = ctl.RunScript(script); err != nil {
				fmt.Printf(" |- run failed %s\n", err)
				continue
			}
			ss = append(ss, detail)
		case "listobj":
			roles := ctl.Roles()
			if len(roles) == 0 {
				fmt.Println("not add roles")
				continue
			}
			for i := range roles {
				fmt.Printf("role %s\n", roles[i].Id())
			}
			fmt.Printf("current exist %d role\n", len(roles))
		case "history":
			if len(ss) == 0 {
				fmt.Printf("No history\n")
				continue
			}
			for i := range ss {
				fmt.Println(ss[i])
			}
		case "h", "help":
			InteractiveUsage()
		case "q", "quit":
			fmt.Println("Thanks for use test interactive model")
			return
		default:
			fmt.Printf("NOT found command '%s'\n", token[0])
			InteractiveUsage()
		}
	}
}

func readCommand(reader *bufio.Reader) (token []string, detail string) {
	var err error
	for {
		if detail, err = reader.ReadString('\n'); err != nil {
			fmt.Printf("read command failed. %s\n", err)
			os.Exit(1)
		}
		detail = strings.Trim(detail, " \n\r\t")
		if token = split(detail); len(token) != 0 {
			return
		}
		fmt.Println("inpute not contain usefull info")
		InteractiveUsage()
	}
}

func split(str string) []string {
	var res []string
	t := strings.Split(strings.Trim(str, " "), " ")
	for i := range t {
		t[i] = strings.Trim(t[i], "\r\n ")
		if len(t[i]) > 0 {
			res = append(res, t[i])
		}
	}
	return res
}

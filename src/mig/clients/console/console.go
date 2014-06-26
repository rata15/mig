/* Mozilla InvestiGator Console

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mig"
	migdb "mig/database"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"syscall"

	"code.google.com/p/gcfg"
	"github.com/jvehent/cljs"
)

type Context struct {
	API struct {
		URL string
	}
}

var ctx Context

func main() {
	fmt.Println("\x1b[32;1m" + `
## ##                                     _.---._     .---.
# # # /-\ ---||  |    /\         __...---' .---. '---'-.   '.
#   #|   | / ||  |   /--\    .-''__.--' _.'( | )'.  '.  '._ :
#   # \_/ ---| \_ \_/    \ .'__-'_ .--'' ._'---'_.-.  '.   '-'.
     ###                         ~ -._ -._''---. -.    '-._   '.
      # |\ |\    /---------|          ~ -.._ _ _ _ ..-_ '.  '-._''--.._
      # | \| \  / |- |__ | |                       -~ -._  '-.  -. '-._''--.._.--''.
     ###|  \  \/  ---__| | |                            ~ ~-.__     -._  '-.__   '. '.
          #####                                               ~~ ~---...__ _    ._ .' '.
          #      /\  --- /-\ |--|                                        ~  ~--.....--~
          # ### /--\  | |   ||-\
          #####/    \ |  \_/ |  \
` + "\x1b[0m")

	homedir := findHomedir()
	// command line options
	var config = flag.String("c", homedir+"/.migconsole", "Load configuration from file")
	flag.Parse()

	err := gcfg.ReadFileInto(&ctx, *config)
	if err != nil {
		panic(err)
	}

	// create a channel that receives signals, and capture
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for signal := range c {
			switch signal {
			case syscall.SIGINT:
				// signal is a ^C, handle it
				fmt.Printf("\nGators are going back underwater. KThksBye.\n")
				os.Exit(0)
			}
		}
	}()

	printStatus()
	fmt.Printf("\nConnected to %s. Use ctrl+c to exit.\n", ctx.API.URL)
	for {
		fmt.Printf("mig> ")
		r := bufio.NewReader(os.Stdin)
		// read command line input, split on newlines
		input, err := r.ReadString(0x0A)
		if err != nil {
			panic(err)
		}
		fmt.Println(input)
	}
}

func findHomedir() string {
	if runtime.GOOS == "darwin" {
		return os.Getenv("HOME")
	} else {
		// find keyring in default location
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		return u.HomeDir
	}
}

func printStatus() {
	resp, err := http.Get(ctx.API.URL + "dashboard")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	st := cljs.New("")
	err = json.Unmarshal(body, &st)
	if err != nil {
		panic(err)
	}
	agtout := make([]string, 3)
	agtout[0] = "Agents Summary:"
	actout := make([]string, 1)
	actout[0] = "Latest Actions:"
	for _, item := range st.Collection.Items {
		for _, data := range item.Data {
			switch data.Name {
			case "action":
				bData, err := json.Marshal(data.Value)
				if err != nil {
					panic(err)
				}
				var a mig.Action
				err = json.Unmarshal(bData, &a)
				if err != nil {
					panic(err)
				}
				var investigators string
				for ctr, i := range a.Investigators {
					if ctr > 0 {
						investigators += "; "
					}
					investigators += i.Name
				}
				if len(investigators) > 30 {
					investigators = investigators[0:27] + "..."
				}
				if len(a.Name) > 30 {
					a.Name = a.Name[0:27] + "..."
				}
				str := fmt.Sprintf("* %s, %s launched '%s' with id %.0f on target '%s'",
					a.LastUpdateTime.Format("On Mon Jan 2 at 3:04pm (MST)"),
					investigators, a.Name, a.ID, a.Target)
				actout = append(actout, str)
			case "active agents":
				agtout[1] = fmt.Sprintf("* %.0f agents have checked in during the last 5 minutes", data.Value)
			case "agents versions count":
				bData, err := json.Marshal(data.Value)
				if err != nil {
					panic(err)
				}
				var sum []migdb.AgentsSum
				err = json.Unmarshal(bData, &sum)
				if err != nil {
					panic(err)
				}
				for _, asum := range sum {
					s := fmt.Sprintf("* %.0f agents run version %s", asum.Count, asum.Version)
					agtout = append(agtout, s)
				}
			case "agents started in the last 24 hours":
				agtout[2] = fmt.Sprintf("* %.0f agents (re)started in the last 24 hours", data.Value)
			}
		}
	}
	fmt.Println("\x1b[31;1m+------\x1b[0m")
	for _, s := range agtout {
		fmt.Println("\x1b[31;1m|\x1b[0m " + s)
	}
	fmt.Println("\x1b[31;1m|\x1b[0m")
	for _, s := range actout {
		fmt.Println("\x1b[31;1m|\x1b[0m " + s)
		if len(actout) < 2 {
			fmt.Println("\x1b[31;1m|\x1b[0m * None")
			break
		}
	}
	fmt.Println("\x1b[31;1m+------\x1b[0m")
}
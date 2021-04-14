package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/TKMAX777/panda"
)

var Slack *SlackHandler

const CheckFile = "check.json"

func init() {
	b, err := ioutil.ReadFile("Settings.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &Settings)
	if err != nil {
		panic(err)
	}

	Panda := panda.NewClient()
	err = Panda.Login(Settings.Panda.ECS_ID, Settings.Panda.PASSWORD)
	if err != nil {
		panic(err)
	}

	info := Panda.GetOwnInfo()
	fmt.Printf("PandA login sccess!\nHello, %s.\n", info.Author)

	Slack, err = NewSlackHandler(Settings.Slack.Token, Panda, CheckFile)
	if err != nil {
		panic(err)
	}
}

func main() {
	Slack.Start()
}

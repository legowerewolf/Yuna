package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/legowerewolf/cryptowrapper"
)

//All fields are exported because of the JSON package
type database struct {
	APITokens map[string]string
	People    []person
	Intents   map[string]intent
	Errors    map[string][]string
	local     bool
}

type person struct {
	DiscordID       string
	PermissionLevel int
	Names           []string
}

type intent struct {
	Models          []string
	Responses       []string
	Extra1          []string
	PermissionLevel int
}

func getData() database {
	var raw []byte
	var err error

	raw, err = ioutil.ReadFile("./data/config.json")
	if err != nil { // If config file not found in proper location
		raw, err = ioutil.ReadFile("./onboarding/config.json")
		if err != nil { // If config file not found in onboarding location
			//grab a version from the url contained in an enironment variable and decrypt with the key found in an enviroment variable
			if os.Getenv("CONFIG_URL") != "" && os.Getenv("CONFIG_KEY") != "" {
				return getDataFromRemote(os.Getenv("CONFIG_URL"), os.Getenv("CONFIG_KEY"))
			}
			fmt.Println("Last-ditch database load effort failed. Unable to start.")
			os.Exit(1)
		}
	}

	return buildDatabaseFromRaw(raw, true)
}

func saveData() {
	if reflect.DeepEqual(rundata, getData()) && rundata.local {
		return
	}
	dat, err := json.Marshal(rundata)
	if err != nil {
		fmt.Println(err.Error())
	}
	ioutil.WriteFile("./data/config.json", dat, 0644)
}

func (db database) getPersonFromAlias(alias string) (person, int, error) {
	for index, person := range rundata.People { //Scan through people the bot is aware of
		if alias[2:len(alias)-1] == person.DiscordID || alias == person.DiscordID { //See if this is a mention (<@ ... >) of the person
			return person, index, nil
		}
		for _, name := range person.Names { //Scan through names the bot knows for this person, ignoring case
			if strings.ToLower(name) == strings.ToLower(alias) {
				return person, index, nil
			}
		}

	}

	//The bot doesn't know this person. Add it to the registry and return the new person.
	matched, _ := regexp.MatchString("(:?)([0-9])+", alias[2:len(alias)-1])
	if len(alias[2:len(alias)-1]) == 18 && matched {
		rundata.People = append(rundata.People, person{DiscordID: alias[2 : len(alias)-1]})
	}
	return person{}, 0, errors.New("Could not find person with alias: " + alias)
}

func (db database) getPeopleFromSlice(s []string) []person {
	ret := []person{}
	if indexOf("and", s) != -1 {
		andindex := indexOf("and", s)
		for _, alias := range append(s[:andindex+1], s[andindex+1]) {
			nperson, _, err := db.getPersonFromAlias(alias)
			if err == nil {
				ret = append(ret, nperson)
			}
		}
	} else {
		nperson, _, err := db.getPersonFromAlias(s[0])
		if err == nil {
			ret = append(ret, nperson)
		}
	}
	return ret
}

func getDataFromRemote(configURL, key string) database {
	resp, err := http.Get(configURL)
	checkErr(err, "get config from remote")

	var contents []byte
	contents, err = ioutil.ReadAll(resp.Body)
	checkErr(err, "read remote config")

	raw, err := cryptowrapper.SymmetricDecrypt(string(contents), os.Getenv("CONFIG_KEY"))
	checkErr(err, "decrypt config")

	return buildDatabaseFromRaw(raw, false)

}

func buildDatabaseFromRaw(raw []byte, local bool) database {
	var c database
	json.Unmarshal(raw, &c)
	c.local = local
	return c
}

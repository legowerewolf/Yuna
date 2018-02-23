package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/legowerewolf/cryptowrapper"
)

//All fields are exported because of the JSON package
type database struct {
	RoleName  string
	APITokens map[string]string
	People    []person
	Models    map[string]string
	Responses map[string][]string
	local     bool
}

type person struct {
	DiscordID string   `json:"discordID"`
	Names     []string `json:"names"`
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

func (db database) getRandomResponse(intent string) string {
	responses := db.Responses[intent]
	return responses[rand.Intn(len(responses))]
}

func (db database) getPersonFromAlias(alias string) (person, error) {
	for _, person := range rundata.People { //Scan through people the bot is aware of
		if alias[2:len(alias)-1] == person.DiscordID { //See if this is a mention (<@ ... >) of the person
			return person, nil
		}
		for _, name := range person.Names { //Scan through names the bot knows for this person, ignoring case
			if strings.ToLower(name) == strings.ToLower(alias) {
				return person, nil
			}
		}

	}

	//The bot doesn't know this person. Add it to the registry and return the new person.
	matched, _ := regexp.MatchString("(:?)([0-9])+", alias[2:len(alias)-1])
	if len(alias[2:len(alias)-1]) == 18 && matched {
		rundata.People = append(rundata.People, person{DiscordID: alias[2 : len(alias)-1]})
	}
	return person{}, errors.New("Could not find person with alias: " + alias)
}

func (db database) getPeopleFromSlice(s []string) []person {
	ret := []person{}
	if indexOf("and", s) != -1 {
		andindex := indexOf("and", s)
		for _, alias := range append(s[:andindex+1], s[andindex+1]) {
			nperson, err := db.getPersonFromAlias(alias)
			if err == nil {
				ret = append(ret, nperson)
			}
		}
	} else {
		nperson, err := db.getPersonFromAlias(s[0])
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

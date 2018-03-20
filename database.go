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

	"github.com/legowerewolf/cryptowrapper/wrapper"
)

//All fields are exported because of the JSON package
type database struct {
	APITokens map[string]string
	People    []person
	Intents   map[string]intent
	Errors    map[string][]string
	local     bool
	SourceURL string
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

func getDataFromRemote(configURL, key string) database {
	resp, err := http.Get(configURL)
	checkErr(err, "get config from remote")

	var contents []byte
	contents, err = ioutil.ReadAll(resp.Body)
	checkErr(err, "read remote config")

	raw, err := wrapper.SymmetricDecrypt(string(contents), os.Getenv("CONFIG_KEY"))
	checkErr(err, "decrypt config")

	db := buildDatabaseFromRaw(raw, false)
	db.SourceURL = configURL

	return db

}

func (db database) checkForUpdates() {
	if db.SourceURL == "" {
		return
	}
	if os.Getenv("CONFIG_URL") != db.SourceURL {
		fmt.Println(os.Getenv("CONFIG_URL"))
		fmt.Println(db.SourceURL)
		db = getDataFromRemote(os.Getenv("CONFIG_URL"), os.Getenv("CONFIG_KEY"))
		fmt.Println("Config updated.")
	}
}

func buildDatabaseFromRaw(raw []byte, local bool) database {
	var c database
	json.Unmarshal(raw, &c)
	c.local = local
	return c
}

func (db database) save(path string) {
	if reflect.DeepEqual(db, getData()) && db.local {
		return
	}
	dat, err := json.Marshal(db)
	if err != nil {
		fmt.Println(err.Error())
	}
	ioutil.WriteFile(path, dat, 0644)
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

// Find the best-matching intent from a list of models and extract data from it.
func (db database) intentOf(command string) (intent string, data map[string]string) {
	maxScore := 0.0
	var model, match string

	for _intent, _intentdata := range db.Intents {
		for _, _model := range _intentdata.Models {
			r := regexp.MustCompile("(?i)%\\w+%")
			newModel := r.ReplaceAllString(_model, "[A-Za-z0-9 ]+")
			r = regexp.MustCompile("(?i)" + newModel)
			r.Longest()
			if float64(len(r.FindString(command)))/float64(len(command)) > maxScore {
				intent = _intent
				model = _model
				match = r.FindString(command)
			}
		}
	}

	data = make(map[string]string)
	if submodels := strings.Split(model, "%"); len(submodels) > 1 && len(submodels)%2 == 1 {
		if submodels[len(submodels)-1] == "" {
			submodels[len(submodels)-1] = "$"
		}
		offset := 0
		for i := 0; i < len(submodels)-2; i += 2 {
			//find the end of the first match
			r := regexp.MustCompile("(?i)" + submodels[i])
			r.Longest()
			startIndex := r.FindStringIndex(match[offset:])[1]

			//find the beginning of the second match
			r = regexp.MustCompile("(?i)" + submodels[i+2])
			r.Longest()
			endIndex := r.FindStringIndex(match[offset:])[0]

			offset = startIndex

			//find and append the data
			data[submodels[i+1]] = match[startIndex:endIndex]
		}
	}
	return intent, data
}

// Check if the user has permission to run the command
func (db database) checkAuthorized(userID, intent string) bool {
	person, _, _ := db.getPersonFromAlias(userID)
	if person.PermissionLevel >= db.Intents[intent].PermissionLevel {
		return true
	}
	return false
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"reflect"
)

//All fields are exported because of the JSON package
type database struct {
	Guild        string
	VoiceChannel string
	RoleName     string
	APITokens    map[string]string
	People       []person
	Models       map[string]string
	Responses    map[string][]string
	local        bool
}

func getData() database {
	var raw []byte
	var err error
	local := true

	raw, err = ioutil.ReadFile("./data/config.json")
	if err != nil { // If config file not found in proper location
		local = false
		raw, err = ioutil.ReadFile("./onboarding/config.json")
		if err != nil { // If config file not found in onboarding location
			//grab a version from the url contained in an enironment variable and decrypt with the key found in an enviroment variable
			if os.Getenv("CONFIG_URL") != "" && os.Getenv("CONFIG_KEY") != "" {
				resp, err := http.Get(os.Getenv("CONFIG_URL"))
				checkErr(err)

				var contents []byte
				contents, err = ioutil.ReadAll(resp.Body)
				checkErr(err)

				raw, err = symmetricDecrypt(string(contents), os.Getenv("CONFIG_KEY"))
			} else {
				fmt.Println("Last-ditch database load effort failed. Unable to start.")
				os.Exit(1)
			}
		}
	}

	var c database
	json.Unmarshal(raw, &c)
	c.local = local
	return c
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

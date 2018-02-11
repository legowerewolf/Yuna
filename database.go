package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
)

//All fields are exported because of the JSON package
type database struct {
	Guild        string            `json:"guild"`
	VoiceChannel string            `json:"voiceChannel"`
	RoleName     string            `json:"roleName"`
	APITokens    map[string]string `json:"apitokens"`
	People       []person          `json:"people"`
	Models       map[string]string `json:"models"`
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

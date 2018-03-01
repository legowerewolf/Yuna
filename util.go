package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

//get the most accurate intent of a command from a map of regex models
func intentOf(command string, intents map[string]intent) (intent string, data []string) {
	maxScore := 0.0
	var model, match string

	for _intent, _intentdata := range intents {
		for _, _model := range _intentdata.Models {
			r := regexp.MustCompile("(?i)" + strings.Replace(_model, "%PARAM%", "[A-Za-z0-9 ]+", -1))
			r.Longest()
			if float64(len(r.FindString(command)))/float64(len(command)) > maxScore {
				intent = _intent
				model = _model
				match = r.FindString(command)
			}
		}
	}

	if submodels := strings.Split(model, "%PARAM%"); len(submodels) > 1 {
		if submodels[len(submodels)-1] == "" {
			submodels[len(submodels)-1] = "$"
		}
		for i := 0; i < len(submodels)-1; i++ {
			//find the length of the first match
			r := regexp.MustCompile("(?i)" + submodels[i])
			r.Longest()
			startIndex := r.FindStringIndex(match)[1]

			//find the length of the second match
			r = regexp.MustCompile("(?i)" + submodels[i+1])
			r.Longest()
			endIndex := r.FindStringIndex(match)[0]

			//find and append the data
			data = append(data, match[startIndex:endIndex])
		}
	}
	return intent, data
}

//get the index of any value of any type in any list - to be added to as necessary
func indexOf(value interface{}, list interface{}) int {
	switch list.(type) {
	case []string:
		list := []string(list.([]string))
		for i, v := range list {
			if v == value {
				return i
			}
		}
	case []*discordgo.User:
		list := []*discordgo.User(list.([]*discordgo.User))
		value2 := value.(*discordgo.User)
		for i, v := range list {
			if value2.ID == v.ID {
				return i
			}
		}
	default:
		fmt.Print(value)
		fmt.Println(reflect.TypeOf(value))
		fmt.Print(list)
		fmt.Println(reflect.TypeOf(list))
	}
	return -1
}

//Turns a computer-format list of strings into a regular english list of things (with oxford comma!)
func toEnglishList(elements []string) string {
	ret := ""
	for i, str := range elements {
		if len(elements)-i >= 3 {
			ret += str + ", "
		} else if len(elements)-i == 2 {
			ret += str
			if len(elements) > 2 {
				ret += ", "
			} else {
				ret += " "
			}
		} else {
			if len(elements) > 1 {
				ret += "and "
			}
			ret += str
		}
	}
	return ret
}

func getRandomString(s []string) string {
	return s[rand.Intn(len(s))]
}

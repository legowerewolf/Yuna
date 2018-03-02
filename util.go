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
func intentOf(command string, intents map[string]intent) (intent string, data map[string]string) {
	maxScore := 0.0
	var model, match string

	for _intent, _intentdata := range intents {
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

package main

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

//get the most accurate intent of a command from a map of regex models
func intentOf(command string, models map[string]string) string {
	intent := ""
	maxScore := 0.0
	for i, m := range models {
		r := regexp.MustCompile(m)
		r.Longest()
		if float64(len(r.FindString(command)))/float64(len(command)) > maxScore {
			intent = i
		}
	}
	return intent
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

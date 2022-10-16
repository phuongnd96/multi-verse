package controllers

import (
	"reflect"
)

func addmap(a map[string]string, b map[string]string) map[string]string {
	if isNil(a) {
		return b
	}
	for k, v := range b {
		a[k] = v
	}
	return a
}

func getLabelSelector(id string) map[string]string {
	return map[string]string{"universe": id}
}

func isNil(p interface{}) bool {
	return p == nil || reflect.ValueOf(p).IsNil()
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

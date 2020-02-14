package main

import (
	"fmt"
	simplejson "github.com/bitly/go-simplejson"
	semver "github.com/openSUSE-zh/node-semver"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func parse(f string) *simplejson.Json {
	f1, _ := ioutil.ReadFile(f)
	json, _ := simplejson.NewJson(f1)
	return json
}

func main() {
	args := os.Args[1:]
	//f, _ := ioutil.ReadFile("1.txt")
	//args := strings.Split(string(f), "\n")
	var files []string
	// old style rpm dependency generator, all files are passed through stdin, filter the package.json
	if len(args) > 1 {
		for _, v := range args {
			// both package.json in node_modules and bower_components
			if strings.HasSuffix(v, "package.json") {
				files = append(files, v)
			}
		}
	} else {
		files = args
	}
	provided := make(map[string]string)
	required := make(map[string]semver.Range)

	for _, v := range files {
		if strings.Contains(v, "bower_components") {

		} else if strings.Contains(filepath.Base(filepath.Dir(v)), "@") {
			json := parse(v)
			// provided
			a := strings.Split(filepath.Base(filepath.Dir(v)), "@")
			if _, ok := provided[a[0]]; !ok {
				provided[a[0]] = a[1]
			}
			// required
			for k, v := range json.Get("dependencies").MustMap() {
				if _, ok := required[k]; !ok {
					v1, _ := v.(string)
					required[k] = semver.NewRange(v1)
				}
			}
		}
	}

	isolated := make(map[string][]string)
	for k, v := range required {
		if _, ok := provided[k]; ok {
			if v.Satisfy(semver.NewSemver(provided[k])) {
				continue
			} else {
				//fmt.Println(k)
				//fmt.Println(provided[k])
				//fmt.Println(v.String())
			}
		}
		if _, ok := isolated[k]; !ok {
			isolated[k] = strings.Split(v.String(), " ")
		}
	}

	if ! (isolated == nil) {
		for k, v := range isolated {
                   for _, v1 := range v {
			fmt.Println("node_module("+k+") "+v1)
		   }
		}
	}
}

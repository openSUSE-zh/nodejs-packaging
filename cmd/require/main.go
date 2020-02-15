package main

import (
	"bufio"
	"fmt"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/marguerite/util/dir"
	semver "github.com/openSUSE-zh/node-semver"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func parse(f string) *simplejson.Json {
	f1, _ := ioutil.ReadFile(f)
	json, _ := simplejson.NewJson(f1)
	return json
}

func findDependencies(files []string, idx int) (map[string]map[string]struct{}, map[string][]semver.Range) {
	provided := make(map[string]map[string]struct{})
	required := make(map[string][]semver.Range)
	if idx > 1 {
		// old style generator
		for _, f := range files {
			if filepath.Dir(filepath.Dir(f)) == "bower_components" {
				//bower
			} else if strings.Contains(filepath.Base(filepath.Dir(f)), "@") {
				// required
				json := parse(f)
				for k, v := range json.Get("dependencies").MustMap() {
					v1, _ := v.(string)
					if _, ok := required[k]; !ok {
						required[k] = []semver.Range{semver.NewRange(v1)}
					} else {
						a := required[k]
						r := semver.NewRange(v1)
						found := false
						for _, v2 := range a {
							if v2.String() == r.String() {
								found = true
								break
							}
						}
						if !found {
							a = append(a, r)
							required[k] = a
						}
					}
				}

				// provided
				a := strings.Split(filepath.Base(filepath.Dir(f)), "@")
				if _, ok := provided[a[0]]; !ok {
					provided[a[0]] = make(map[string]struct{})
					provided[a[0]][a[1]] = struct{}{}
				} else {
					if _, ok := provided[a[0]][a[1]]; !ok {
						provided[a[0]][a[1]] = struct{}{}
					}

				}

			}
		}
		return provided, required
	}

	f := files[0]
	if filepath.Dir(filepath.Dir(f)) == "bower_componenets" {
		//bower
	} else if strings.Contains(filepath.Base(filepath.Dir(f)), "@") {
		// required
		json := parse(f)
		for k, v := range json.Get("dependencies").MustMap() {
			v1, _ := v.(string)
			required[k] = []semver.Range{semver.NewRange(v1)}
		}
		// provided
		re := regexp.MustCompile("^(.*\\/usr\\/lib\\/node_modules\\/)")
		d := re.FindStringSubmatch(f)[1]
		directories, _ := dir.Ls(d, "dir")
		for _, v := range directories {
			v1 := filepath.Base(v)
			if strings.Contains(v1, "@") {
				a := strings.Split(v1, "@")
				if _, ok := provided[a[0]]; !ok {
					provided[a[0]] = make(map[string]struct{})
					provided[a[0]][a[1]] = struct{}{}
				} else {
					if _, ok := provided[a[0]][a[1]]; !ok {
						provided[a[0]][a[1]] = struct{}{}
					}
				}
			}
		}
	}

	return provided, required
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var files []string
	idx := 0
	// old style rpm dependency generator, all files are passed through stdin, filter the package.json
	for scanner.Scan() {
		idx += 1
		// both package.json in node_modules and bower_components
		text := scanner.Text()
		if strings.HasSuffix(text, "package.json") {
			files = append(files, text)
		}
	}

	provided, required := findDependencies(files, idx)

	isolated := make(map[string][]string)
	for k, v := range required {
		for _, v1 := range v {
			//v1 semver.Range
			found := false
			if _, ok := provided[k]; ok {
				for k1 := range provided[k] {
					if v1.Satisfy(semver.NewSemver(k1)) {
						found = true
					}
				}
			}
			if found {
				continue
			} else {
				v2 := strings.Split(v1.String(), " || ")[0]
				if _, ok := isolated[k]; ok {
					a := isolated[k]
					for _, v3 := range strings.Split(v2, " ") {
						a = append(a, v3)
					}
					isolated[k] = a
				} else {
					isolated[k] = strings.Split(v2, " ")
				}
			}
		}
	}

	if !(isolated == nil) {
		re := regexp.MustCompile("([>=<]+)(\\d.*$)")
		for k, v := range isolated {
			for _, v1 := range v {
				m := re.FindStringSubmatch(v1)
				fmt.Println("node_module(" + k + ") " + m[1] + " " + m[2])
			}
		}
	}
}

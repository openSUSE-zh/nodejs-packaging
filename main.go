package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/marguerite/util/dir"
)

//Prefix where the build locates
const (
	Prefix    = "/home/zhou/Packages/home:MargueriteSu:branches:devel:languages:nodejs/gulp"
	SourceDir = Prefix
)

func main() {
	var prep, install, clean, filelist bool
	flag.BoolVar(&prep, "prep", false, "do pre-build checks")
	flag.BoolVar(&install, "install", false, "install modules from json")
	flag.BoolVar(&clean, "clean", false, "clean the build environment")
	flag.BoolVar(&filelist, "filelist", false, "generate filelist")
	flag.Parse()

	files, err := dir.Glob(SourceDir, regexp.MustCompile(`\.json$`))
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile(files[0])
	if err != nil {
		log.Fatal(err)
	}

	json, err := simplejson.NewJson(data)
	if err != nil {
		log.Fatal(err)
	}

	if prep {
		chkSourceTarballs(json)
	}

	if install {
		installModules(json)
	}
}

func chkSourceTarballs(json *simplejson.Json) {
	// check if every package specified in the .json file exists in sourcedir
	for k := range json.MustMap() {
		a := strings.Split(k, ":")
		f := a[0] + "-" + a[1] + ".tgz"
		if _, err := os.Stat(filepath.Join(SourceDir, f)); os.IsNotExist(err) {
			fmt.Printf("%s is required in .json but not found under %s\n", f, SourceDir)
			fmt.Printf("You may use an incorrect .json file or your source tarballs were not fully uploaded.\n")
			fmt.Printf("We suggest you re-generate the .json file with node2rpm.\n")
			os.Exit(1)
		}
		chkSourceTarballs(json.Get(k).Get("child"))
	}
}

func installModules(json *simplejson.Json) {

}

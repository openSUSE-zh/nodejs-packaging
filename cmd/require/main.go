package main

import (
	"fmt"
	"os"
	"strings"
	//"github.com/marguerite/util/dir"
	//simplejson "github.com/bitly/go-simplejson"
)

func main() {
  for _, v := range os.Args[1:] {
	  fmt.Println(v)
  }
  args := os.Args[1:]
  //args, _ := dir.Ls("/home/marguerite/home:MargueriteSu:branches:devel:languages:nodejs/gulp/usr/lib/node_modules")
  var files []string
  // old style rpm dependency generator, all files are passed through stdin, filter the package.json
  if len(args) > 1 {
	  for _, v := range args {
		  // both package.json in node_modules and bower_components
		  if strings.HasSuffix(v, "package.json") {
			  files = append(files, v)
		  }
	  }
  }

  for _, v := range files {
    if strings.Contains(v, "bower_components") {

    } else {
      fmt.Println(v)
    }
  }

}

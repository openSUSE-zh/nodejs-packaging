package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var files []string
	// old style rpm dependency generator, all files are passed through stdin, filter the package.json
	for scanner.Scan() {
		// both package.json in node_modules and bower_components
		text := scanner.Text()
		if strings.HasSuffix(text, "package.json") {
			files = append(files, text)
		}
	}

	for _, f := range files {
		f1 := filepath.Base(filepath.Dir(f))
		if strings.Contains(f1, "@") {
			a := strings.Split(f1, "@")
			if len(strings.Split(strings.Split(f, "/usr/lib/node_modules/")[1], "/")) > 2 {
				// bundled
				fmt.Println("node_bundled_module(" + a[0] + ") = " + a[1])
			} else {
				fmt.Println("node_module(" + a[0] + ") = " + a[1])
			}
		}
	}
}

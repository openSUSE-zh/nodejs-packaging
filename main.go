package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/marguerite/util/dir"
)

//Prefix where the build locates
var (
	Prefix    = "/home/marguerite/home:MargueriteSu:branches:devel:languages:nodejs/gulp"
	SourceDir = Prefix
	DestDir   = Prefix
	SiteLib   = filepath.Join(DestDir, "/usr/lib/node_modules")
)

func main() {
	var prep, inst, clean, filelist bool
	flag.BoolVar(&prep, "prep", false, "do pre-build checks")
	flag.BoolVar(&inst, "install", false, "install modules from json")
	flag.BoolVar(&clean, "clean", false, "clean the build environment")
	flag.BoolVar(&filelist, "filelist", false, "generate filelist")
	flag.Parse()

	files, err := dir.Glob(SourceDir, regexp.MustCompile(`\.json$`), regexp.MustCompile(`/\.osc/`))
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
		ChkSourceTarballs(json)
	}

	if inst {
		install(json, SiteLib)
	}
}

func ChkSourceTarballs(json *simplejson.Json) {
	notFound := make(map[string]struct{})
	chkSourceTarballs(json, notFound)
	if len(notFound) > 0 {
		fmt.Printf("These modules are specified in .json but not found under %s:\n", SourceDir)
		var s string
		for v := range notFound {
			s += v + "\n"
		}
		fmt.Println(s)
		fmt.Println("You may use an incorrect .json file or your sources were not handled/uploaded well")
		fmt.Println("We recommend to re-generate .json file with node2rpm")
	}
}

func chkSourceTarballs(json *simplejson.Json, notFound map[string]struct{}) {
	// check if every package specified in the .json file exists in sourcedir
	for k := range json.MustMap() {
		a := strings.Split(k, ":")
		f := a[0] + "-" + a[1] + ".tgz"
		if _, err := os.Stat(filepath.Join(SourceDir, f)); os.IsNotExist(err) {
			if _, ok := notFound[f]; !ok {
				notFound[f] = struct{}{}
			}
		}
		chkSourceTarballs(json.Get(k).Get("child"), notFound)
	}
}

func install(json *simplejson.Json, directory string) {
	if _, err := os.Stat(SiteLib); os.IsNotExist(err) {
		os.MkdirAll(SiteLib, os.ModePerm)
	}
	for k := range json.MustMap() {
		s := strings.Split(k, ":")
		f := s[0] + "-" + s[1] + ".tgz"
		t := filepath.Join(directory, s[0]+"@"+s[1])
		//fmt.Printf("Creating directory %s\n", t)
		os.MkdirAll(t, os.ModePerm)
		source := filepath.Join(SourceDir, f)
		//fmt.Printf("Unpacking %s to %s\n", source, t)
		err := unpack(source, t)
		if err != nil {
			fmt.Println(err)
		}
		postinstall(t)
		install(json.Get(k), filepath.Join(t, "node_modules"))
	}
}

func postinstall(d string) {
	files, _ := dir.Ls(d)
	for _, f := range files {
		if strings.HasSuffix(f, ".gyp") {
			if _, err := os.Stat("/usr/bin/npm"); os.IsNotExist(err) {
				fmt.Printf("Your package contains node module %s which needs npm to build, but no BuildRequires: npm nodejs-devel in specfile.\n", filepath.Base(d))
				continue
			}
			cmd := exec.Command("/usr/bin/npm", "build", d)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
		// remove build temporary file
		removed := false
		re := regexp.MustCompile(`\.(c|h|cc|cpp|o|gyp|gypi)$|Makefile$|build\/Release\/obj\.target`)
		if re.MatchString(f) {
			fmt.Printf("Removing temporary build file %s.\n", f)
			err := os.Remove(f)
			if err != nil {
				fmt.Printf("Failed to remove %s: %v\n", f, err)
			}
			removed = true
		}
		// fix binary permission
		if !removed && (strings.Contains(f, "/bin/") || strings.HasSuffix(f, ".node")) {
			fmt.Printf("Fixing permission %s.\n", f)
			err := os.Chmod(f, 0755)
			if err != nil {
				fmt.Printf("Set permission 0755 on %s failed: %v.\n", f, err)
			}
		}
		// remove empty directory
		if i, err := os.Stat(f); i.IsDir() && err == nil {
			f1, err := os.Open(f)
			if err != nil {
				fmt.Printf("Failed to open %s: %v\n", f, err)
			}
			defer f1.Close()
			_, err = f1.Readdirnames(1)
			if err == io.EOF {
				fmt.Printf("Removing empty directory %s.\n", f)
				err = os.Remove(f)
				if err != nil {
					fmt.Printf("Failed to remove %s: %v\n", f, err)
				}
			}
		}
		// check bower
	}
}

func unpack(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	tarReader := tar.NewReader(archive)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		name := strings.TrimPrefix(header.Name, "package/")
		if skipName(name) {
			continue
		}

		if strings.Contains(name, "/") {
			p := filepath.Join(target, filepath.Dir(name))
			if _, err := os.Stat(p); os.IsNotExist(err) {
				err = os.MkdirAll(p, os.ModePerm)
				if err != nil {
					return err
				}
			}
		}

		path := filepath.Join(target, name)

		info := header.FileInfo()

		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

func skipName(n string) bool {
	re := regexp.MustCompile(`^\..*$|.*~$|\.(bat|cmd|orig|bak|sln|njsproj|exe)$|example(s)?(\.js)?$|benchmark(s)?(\.js)?$|sample(s)?(\.js)?$|test(s)?(\.js)?$|_test\.|coverage|windows|appveyor\.yml|browser$`)
	if re.MatchString(n) {
		//fmt.Printf("%s is Linux unneeded, temporary, project management, or test/sample file, skipped.\n", n)
		return false
	}
	return true
}

package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	Prefix    = "/home/abuild/rpmbuild"
	SourceDir = filepath.Join(Prefix, "SOURCES")
)

func destDir() string {
	d := filepath.Join(Prefix, "BUILDROOT")
	f, _ := dir.Ls(d, "dir")
	return f[1]
}

func siteLib() string {
	return filepath.Join(destDir(), "/usr/lib/node_modules")
}

func main() {
	var prep, inst, filelist bool
	flag.BoolVar(&prep, "prep", false, "do pre-build checks")
	flag.BoolVar(&inst, "install", false, "install modules from json")
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
		install(json, siteLib())
	}

	if filelist {
		list()
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
	lib := siteLib()
	if _, err := os.Stat(lib); os.IsNotExist(err) {
		os.MkdirAll(lib, os.ModePerm)
	}
	for k := range json.MustMap() {
		s := strings.Split(k, ":")
		f := s[0] + "-" + s[1] + ".tgz"
		t := filepath.Join(directory, s[0]+"@"+s[1])
		fmt.Printf("Creating directory %s\n", t)
		os.MkdirAll(t, os.ModePerm)
		source := filepath.Join(SourceDir, f)
		fmt.Printf("Unpacking %s to %s\n", source, t)
		err := unpack(source, t)
		if err != nil {
			fmt.Println(err)
		}
		install(json.Get(k), filepath.Join(t, "node_modules"))
	}
	postinstall(lib)
}

func postinstall(d string) {
	files, _ := dir.Ls(d)
	for _, f := range files {
		if strings.HasSuffix(f, ".gyp") {
			if _, err := os.Stat("/usr/bin/npm"); os.IsNotExist(err) {
				fmt.Printf("Your package contains node module %s which needs npm to build, but no BuildRequires: npm nodejs-devel gcc-c++ in specfile.\n", filepath.Base(d))
				continue
			}
			cmd := exec.Command("/usr/bin/npm", "build", filepath.Dir(f))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				os.Exit(1)
			}
		}
		// remove build temporary file
		re := regexp.MustCompile(`\.(c|h|cc|cpp|o|gyp|gypi|target\.mk|deps)$|Makefile$|build\/Release\/obj\.target`)
		if re.MatchString(f) {
			fmt.Printf("Removing temporary build file %s.\n", f)
			err := os.RemoveAll(f)
			if err != nil {
				fmt.Printf("Failed to remove %s: %v\n", f, err)
			}
			continue
		}

		f1, err := os.Open(f)
		if err != nil {
			fmt.Printf("Failed to open %s: %v\n", f, err)
		}
		f2, _ := f1.Stat()

		// remove empty directory
		if f2.IsDir() {
			_, err := f1.Readdirnames(1)
			if err == io.EOF {
				fmt.Printf("Removing empty directory %s.\n", f)
				err = os.Remove(f)
				if err != nil {
					fmt.Printf("Failed to remove %s: %v.\n", f, err)
				}
			}
			f1.Close()
			continue
		}

		// fix binary permission
		if strings.Contains(f, "/bin") || strings.HasSuffix(f, ".node") {
			// no need to fix file already binary
			f1.Close()
			continue
		}

		buffer := make([]byte, 512)
		_, err = f1.Read(buffer)
		if err != nil {
			// skip zero-byte files
			if err != io.EOF {
				fmt.Printf("Failed to read the first 512 bytes of %s: %v.\n", f, err)
			}
			f1.Close()
			continue
		}

		if http.DetectContentType(buffer) != "application/octet-stream" && strings.Contains(f2.Mode().String(), "x") {
			fmt.Printf("Fixing permission %s.\n", f)
			err := os.Chmod(f, 0644)
			if err != nil {
				fmt.Printf("Set permission 0755 on %s failed: %v.\n", f, err)
			}
		}
		f1.Close()
		// check bower
	}
}

func list() {
	lib := siteLib()
	dest := destDir()
	d, _ := dir.Ls(lib, "dir")
	f, _ := dir.Ls(lib)
	var s string
	for _, v := range d {
		s += "%dir " + strings.TrimPrefix(v, dest) + "\n"
	}
	for _, v := range f {
		s += strings.TrimPrefix(v, dest) + "\n"
	}
	ioutil.WriteFile(filepath.Join(SourceDir, "files.txt"), []byte(s), 0644)
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
	re := regexp.MustCompile(`(\/|^)\..*$|~$|\.(bat|cmd|orig|bak|sln|njsproj|exe)$|example(s)?(\.js)?|benchmark(s)?(\.js)?|sample(s)?(\.js)?|test(s)?(\.)?(js)?|coverage|windows|appveyor\.yml|browser(\/)?|node_modules`)
	if re.MatchString(n) {
		fmt.Printf("%s is Linux unneeded, temporary, project management, or test/sample file, skipped.\n", n)
		return true
	}
	return false
}

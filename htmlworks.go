package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/playree/goingtpl"
)

// DirectoriesConfig various directory settings.
type DirectoriesConfig struct {
	Contents  string `toml:"contents"`
	Exclusion string `toml:"exclusion"`
	Resources string `toml:"resources"`
	Generate  string `toml:"generate"`
}

// ServerConfig various server settings.
type ServerConfig struct {
	Port int `toml:"port"`
}

// Config settings.
type Config struct {
	Directories DirectoriesConfig `toml:"directories"`
	Server      ServerConfig      `toml:"server"`
}

const (
	configFilePath = "./htmlworks.toml"
	paramStart     = "<!--params"
	paramEnd       = "-->"
)

var (
	conf Config
)

func main() {
	flag.Parse()
	mode := flag.Arg(0)
	log.Println("args:", mode)

	// Load config file.
	if _, err := toml.DecodeFile(configFilePath, &conf); err != nil {
		log.Fatalln("Error:", err)
	}

	switch mode {
	case "init":
		procInit()
		break
	case "serve":
		procServer()
		break
	case "gen":
		procGenerate()
		break
	default:
		break
	}
}

func procInit() {

}

func procServer() {
	log.Println("Start Server ========")
	log.Println("Port:", conf.Server.Port)
	log.Println("Contents directory:", conf.Directories.Contents)
	log.Println("Resources directory:", conf.Directories.Resources)

	goingtpl.SetBaseDir(conf.Directories.Contents)

	http.HandleFunc("/", handleServer)
	http.Handle(
		"/"+conf.Directories.Resources+"/",
		http.StripPrefix(
			"/"+conf.Directories.Resources+"/",
			http.FileServer(http.Dir(conf.Directories.Resources+"/"))))
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", conf.Server.Port), nil))
}

func handleServer(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UnixNano()
	log.Println("----")
	log.Println("RequestURI:", r.RequestURI)
	filename := convFilename(r.RequestURI)

	err := executeWriter(w, filename)
	if err != nil {
		log.Println("Not found file:", filename)
		http.NotFound(w, r)
		return
	}

	log.Printf("ExecTime=%d MicroSec\n",
		(time.Now().UnixNano()-start)/int64(time.Microsecond))
}

func executeWriter(w io.Writer, filename string) error {
	buf, err := ioutil.ReadFile(filepath.Join(conf.Directories.Contents, filename))
	if err != nil {
		return err
	}
	log.Println("Load file:", filename)

	// Extraction params.
	params, contents, err := extParam(string(buf))
	if err != nil {
		log.Println(err)
	}

	funcMap := template.FuncMap{
		"repeat": func(s string, i int) string {
			return strings.Repeat(s, i)
		}}
	tpl := template.Must(goingtpl.ParseFuncs(filename, contents, funcMap))

	err = tpl.Execute(w, params)
	if err != nil {
		panic(err)
	}
	return nil
}

func extParam(contents string) (map[string]interface{}, string, error) {
	if start := strings.Index(contents, paramStart); start >= 0 {
		start += len(paramStart)
		if end := strings.Index(contents[start:], paramEnd); end >= 0 {
			end += start
			var params map[string]interface{}
			err := json.Unmarshal([]byte(contents[start:end]), &params)
			log.Println("params", params)

			return params, contents[end+len(paramEnd):], err
		}
	}
	return map[string]interface{}{}, contents, nil
}

func procGenerate() {
	start := time.Now().UnixNano()
	log.Println("Start Generate ========")
	log.Println("Contents directory:", conf.Directories.Contents)
	log.Println("Exclusion directory:", conf.Directories.Exclusion)
	log.Println("Resources directory:", conf.Directories.Resources)

	goingtpl.SetBaseDir(conf.Directories.Contents)

	// Generate Contents
	genContents()

	// Copy Resources
	copyResources()

	log.Printf("ExecTime=%d MicroSec\n",
		(time.Now().UnixNano()-start)/int64(time.Microsecond))
}

func genContents() {
	targetFileList := getTargetContents()
	if len(targetFileList) > 0 {
		log.Println("Target file:", len(targetFileList))
		for _, file := range targetFileList {
			// Generate HTML
			log.Println("----")
			var buffSrc bytes.Buffer
			executeWriter(&buffSrc, file)

			targetPath := filepath.Join(conf.Directories.Generate, file)
			targetDir := filepath.Dir(targetPath)

			// Exist Directory & Create Directory
			if f, err := os.Stat(targetDir); os.IsNotExist(err) || !f.IsDir() {
				err := os.MkdirAll(targetDir, 0777)
				if err != nil {
					log.Fatalln("Error:", err)
				}
			}

			// Exist Destination
			buffDest, err := ioutil.ReadFile(targetPath)
			if err != nil {
				// Create File
				log.Println("Create:", targetPath)
				err := ioutil.WriteFile(targetPath, buffSrc.Bytes(), 0664)
				if err != nil {
					log.Fatalln("Error:", err)
				}
			} else {
				// Compare Src & Dest
				if bytes.Compare(buffSrc.Bytes(), buffDest) != 0 {
					// Update File
					log.Println("Update:", targetPath)
					err := ioutil.WriteFile(targetPath, buffSrc.Bytes(), 0664)
					if err != nil {
						log.Fatalln("Error:", err)
					}
				} else {
					log.Println("Pass:", targetPath)
				}
			}
		}
	} else {
		log.Println("Target file not found.")
	}
}

func copyResources() {
	targetFileList := getTargetResources()
	if len(targetFileList) > 0 {
		log.Println("Target file:", len(targetFileList))
		for _, file := range targetFileList {
			log.Println("----")
			log.Println("Found resource:", file)
			pathFrom := filepath.Join(conf.Directories.Resources, file)
			pathTo := filepath.Join(conf.Directories.Generate, pathFrom)
			targetDir := filepath.Dir(pathTo)

			// Exist From
			buffFrom, err := ioutil.ReadFile(pathFrom)
			if err != nil {
				log.Fatalln("Error:", err)
			}

			// Exist Directory & Create Directory
			if f, err := os.Stat(targetDir); os.IsNotExist(err) || !f.IsDir() {
				err := os.MkdirAll(targetDir, 0777)
				if err != nil {
					log.Fatalln("Error:", err)
				}
			}

			// Exist To
			buffTo, err := ioutil.ReadFile(pathTo)
			if err != nil {
				// Copy file
				log.Println("Copy:", pathTo)
				err := ioutil.WriteFile(pathTo, buffFrom, 0664)
				if err != nil {
					log.Fatalln("Error:", err)
				}
			} else {
				// Compare From & To
				if bytes.Compare(buffFrom, buffTo) != 0 {
					// Copy file
					log.Println("Copy:", pathTo)
					err := ioutil.WriteFile(pathTo, buffFrom, 0664)
					if err != nil {
						log.Fatalln("Error:", err)
					}
				} else {
					log.Println("Pass:", pathTo)
				}
			}
		}
	}
}

func convFilename(uri string) string {
	filename := uri[1:]
	if strings.HasSuffix(uri, "/") {
		filename += "index.html"
	}
	return filename
}

func getTargetContents() []string {
	return _getTargetContents(conf.Directories.Contents)
}

func _getTargetContents(targetDir string) []string {
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		panic(err)
	}

	var fileList []string
	for _, file := range files {
		if file.Name()[0] == '.' {
			continue
		} else if file.IsDir() {
			if file.Name() != conf.Directories.Exclusion {
				fileList = append(fileList, _getTargetContents(filepath.Join(targetDir, file.Name()))...)
			}
		} else {
			fileList = append(fileList, (filepath.Join(targetDir, file.Name()))[len(conf.Directories.Contents)+1:])
		}
	}
	return fileList
}

func getTargetResources() []string {
	return _getTargetResources(conf.Directories.Resources)
}

func _getTargetResources(targetDir string) []string {
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		panic(err)
	}

	var fileList []string
	for _, file := range files {
		if file.Name()[0] == '.' {
			continue
		} else if file.IsDir() {
			fileList = append(fileList, _getTargetContents(filepath.Join(targetDir, file.Name()))...)
		} else {
			fileList = append(fileList, (filepath.Join(targetDir, file.Name()))[len(conf.Directories.Resources)+1:])
		}
	}
	return fileList
}

func genFile(targetPath string) error {
	return nil
}

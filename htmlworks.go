package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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

// GenerateConfig various generate settings.
type GenerateConfig struct {
	Delete bool `toml:"delete"`
}

// Config settings.
type Config struct {
	Directories DirectoriesConfig `toml:"directories"`
	Server      ServerConfig      `toml:"server"`
	Generate    GenerateConfig    `toml:"generate"`
}

const (
	version        = "1.0.0"
	configFilePath = "./htmlworks.toml"
	paramStart     = "<!--params"
	paramEnd       = "-->"
	initToml       = `
### HTML Works Settings
[directories]
contents = "contents" #
exclusion = "_parts"
resources = "resources"
generate = "public"

[server]
# Development server port.
port = 8088

[generate]
# Delete files that are not generated.
delete = true
`
)

var (
	conf Config
	serv *http.Server
)

func main() {
	flag.Parse()
	mode := flag.Arg(0)
	log.Println("HTML Works", "ver", version)
	log.Println("args:", mode)

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

func initGoingtpl() {
	goingtpl.SetBaseDir(conf.Directories.Contents)
	goingtpl.AddFixedFunc(
		"now",
		func(format string) string {
			if format == "" {
				format = "2006/01/02 15:04:05"
			}
			return time.Now().Format(format)
		})
}

func procInit() {
	log.Println("Start Init ========")
	if f, err := os.Stat(configFilePath); os.IsNotExist(err) || f.IsDir() {
		// Create htmlworks.toml
		log.Println("Create htmlworks.toml")
		fatalCheck(ioutil.WriteFile(configFilePath, []byte(initToml), 0664))
		if f, err := os.Stat("contents\\_parts"); os.IsNotExist(err) || !f.IsDir() {
			fatalCheck(os.MkdirAll("contents\\_parts", 0777))
		}
		if f, err := os.Stat("resources"); os.IsNotExist(err) || !f.IsDir() {
			fatalCheck(os.MkdirAll("resources", 0777))
		}
	} else {
		log.Println("htmlworks.toml already exists")
	}
	log.Println("End Init ========")
}

func procServer() {
	// Load config file.
	log.Println("Load Setting >", configFilePath)
	_, err := toml.DecodeFile(configFilePath, &conf)
	fatalCheck(err)

	log.Println("Start Server ========")
	log.Println("Port:", conf.Server.Port)
	log.Println("Contents directory:", conf.Directories.Contents)
	log.Println("Resources directory:", conf.Directories.Resources)

	initGoingtpl()

	http.HandleFunc("/", handleServer)
	http.Handle(
		"/"+conf.Directories.Resources+"/",
		http.StripPrefix(
			"/"+conf.Directories.Resources+"/",
			http.FileServer(http.Dir(conf.Directories.Resources+"/"))))

	serv = &http.Server{Addr: fmt.Sprintf(":%d", conf.Server.Port)}
	go func() {
		fatalCheck(serv.ListenAndServe())
	}()

	log.Printf("Starting development server at http://localhost:%d/\n", conf.Server.Port)
	log.Println("Quit the server with CONTROL-C.")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	log.Printf("SIGNAL %d received, then shutting down...\n", <-quit)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serv.Shutdown(ctx)
}

func handleServer(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UnixNano()
	log.Println("- RequestURI:", r.RequestURI)
	filename := convFilename(r.RequestURI)

	if err := executeWriter(w, filename); err != nil {
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
	log.Println("> Load file:", filename)

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
	// Load config file.
	log.Println("Load Setting >", configFilePath)
	_, err := toml.DecodeFile(configFilePath, &conf)
	fatalCheck(err)

	start := time.Now().UnixNano()
	log.Println("Start Generate ========")
	log.Println("Contents directory:", conf.Directories.Contents)
	log.Println("Exclusion directory:", conf.Directories.Exclusion)
	log.Println("Resources directory:", conf.Directories.Resources)

	initGoingtpl()

	processedMap := map[string]bool{}

	// Get list before generation
	log.Println("# Check before generate")
	if f, err := os.Stat(conf.Directories.Generate); os.IsExist(err) && f.IsDir() {
		for _, file := range getTargetGenerate() {
			processedMap[file] = true
		}
	}

	// Generate contents
	log.Println("# Generate contents")
	for _, file := range genContents() {
		if _, ok := processedMap[file]; ok {
			delete(processedMap, file)
		}
	}

	// Copy resources
	log.Println("# Copy resources")
	for _, file := range copyResources() {
		file = filepath.Join(conf.Directories.Resources, file)
		if _, ok := processedMap[file]; ok {
			delete(processedMap, file)
		}
	}

	// Delete processed
	if conf.Generate.Delete {
		log.Println("# Delete not generated")
		for key := range processedMap {
			file := filepath.Join(conf.Directories.Generate, key)
			fatalCheck(os.Remove(file))
			log.Println("Delete:", file)
		}
	}

	log.Println("End Generate ========")
	log.Printf("ExecTime=%d MicroSec\n",
		(time.Now().UnixNano()-start)/int64(time.Microsecond))
}

func genContents() []string {
	targetFileList := getTargetContents()
	if len(targetFileList) > 0 {
		log.Println("Target file:", len(targetFileList))
		for _, file := range targetFileList {
			// Generate HTML
			var buffSrc bytes.Buffer
			executeWriter(&buffSrc, file)

			targetPath := filepath.Join(conf.Directories.Generate, file)
			targetDir := filepath.Dir(targetPath)

			// Exist Directory & Create Directory
			if f, err := os.Stat(targetDir); os.IsNotExist(err) || !f.IsDir() {
				fatalCheck(os.MkdirAll(targetDir, 0777))
			}

			// Exist Destination
			buffDest, err := ioutil.ReadFile(targetPath)
			if err != nil {
				// Create File
				log.Println("Create:", targetPath)
				fatalCheck(ioutil.WriteFile(targetPath, buffSrc.Bytes(), 0664))
			} else {
				// Compare Src & Dest
				if bytes.Compare(buffSrc.Bytes(), buffDest) != 0 {
					// Update File
					log.Println("Update:", targetPath)
					fatalCheck(ioutil.WriteFile(targetPath, buffSrc.Bytes(), 0664))
				} else {
					log.Println("Pass:", targetPath)
				}
			}
		}
	} else {
		log.Println("Target file not found.")
	}
	return targetFileList
}

func copyResources() []string {
	targetFileList := getTargetResources()
	if len(targetFileList) > 0 {
		log.Println("Target file:", len(targetFileList))
		for _, file := range targetFileList {
			log.Println("> Found resource:", file)
			pathFrom := filepath.Join(conf.Directories.Resources, file)
			pathTo := filepath.Join(conf.Directories.Generate, pathFrom)
			targetDir := filepath.Dir(pathTo)

			// Exist From
			buffFrom, err := ioutil.ReadFile(pathFrom)
			fatalCheck(err)

			// Exist Directory & Create Directory
			if f, err := os.Stat(targetDir); os.IsNotExist(err) || !f.IsDir() {
				fatalCheck(os.MkdirAll(targetDir, 0777))
			}

			// Exist To
			buffTo, err := ioutil.ReadFile(pathTo)
			if err != nil {
				// Copy file
				log.Println("Copy:", pathTo)
				fatalCheck(ioutil.WriteFile(pathTo, buffFrom, 0664))
			} else {
				// Compare From & To
				if bytes.Compare(buffFrom, buffTo) != 0 {
					// Copy file
					log.Println("Copy:", pathTo)
					fatalCheck(ioutil.WriteFile(pathTo, buffFrom, 0664))
				} else {
					log.Println("Pass:", pathTo)
				}
			}
		}
	}
	return targetFileList
}

func convFilename(uri string) string {
	filename := uri[1:]
	if strings.HasSuffix(uri, "/") {
		filename += "index.html"
	}
	return filename
}

func getTargetContents() []string {
	return _getTargetList(
		conf.Directories.Contents,
		conf.Directories.Contents,
		conf.Directories.Exclusion)
}

func getTargetResources() []string {
	return _getTargetList(
		conf.Directories.Resources,
		conf.Directories.Resources,
		"")
}

func getTargetGenerate() []string {
	return _getTargetList(
		conf.Directories.Generate,
		conf.Directories.Generate,
		"")
}

func _getTargetList(targetDir string, baseDir string, excDir string) []string {
	files, err := ioutil.ReadDir(targetDir)
	fatalCheck(err)

	var fileList []string
	for _, file := range files {
		if file.Name()[0] == '.' {
			continue
		} else if file.IsDir() {
			if file.Name() != excDir {
				fileList = append(fileList, _getTargetList(filepath.Join(targetDir, file.Name()), baseDir, excDir)...)
			}
		} else {
			fileList = append(fileList, (filepath.Join(targetDir, file.Name()))[len(baseDir)+1:])
		}
	}
	return fileList
}

func fatalCheck(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

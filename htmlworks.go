package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/playree/goingtpl"
)

// DirectoriesConfig various directory settings.
type DirectoriesConfig struct {
	Contents  string `toml:"contents"`
	Resources string `toml:"resources"`
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
		log.Fatal(err)
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

	goingtpl.SetBaseDir(conf.Directories.Contents)

	http.HandleFunc("/", handleServer)
	http.Handle(
		"/"+conf.Directories.Resources+"/",
		http.StripPrefix(
			"/"+conf.Directories.Resources+"/",
			http.FileServer(http.Dir(conf.Directories.Resources+"/"))))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Server.Port), nil))
}

func handleServer(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UnixNano()
	log.Println("RequestURI:", r.RequestURI)
	filename := convFilename(r.RequestURI)

	buf, err := ioutil.ReadFile(goingtpl.GetBaseDir() + filename)
	if err != nil {
		log.Println("Not found file:", filename)
		http.NotFound(w, r)
		return
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

	log.Printf("ExecTime=%d MicroSec\n",
		(time.Now().UnixNano()-start)/int64(time.Microsecond))
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
	log.Println("Start Generate ========")
}

func convFilename(uri string) string {
	filename := uri[1:]
	if strings.HasSuffix(uri, "/") {
		filename += "index.html"
	}
	return filename
}

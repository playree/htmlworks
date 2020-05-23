package main

import (
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
		http.StripPrefix("/"+conf.Directories.Resources+"/", http.FileServer(http.Dir(conf.Directories.Resources+"/"))))
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

	funcMap := template.FuncMap{
		"repeat": func(s string, i int) string {
			return strings.Repeat(s, i)
		}}
	tpl := template.Must(goingtpl.ParseFuncs(filename, string(buf), funcMap))

	m := map[string]string{
		"Date": time.Now().Format("2006-01-02"),
		"Time": time.Now().Format("15:04:05"),
	}
	err = tpl.Execute(w, m)
	if err != nil {
		panic(err)
	}

	log.Printf("ExecTime=%d MicroSec\n",
		(time.Now().UnixNano()-start)/int64(time.Microsecond))
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

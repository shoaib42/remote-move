package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Void struct{}

type Constants struct {
	DnDir            string   `yaml:"dnDir"`
	RootDir          string   `yaml:"rootDir"`
	ExcRDir          []string `yaml:"excludeInRootDir"`
	ExcTDir          []string `yaml:"excludeInTxmnDir"`
	AllowedCIDRs     []string `yaml:"allowedCIDRs"`
	ServerBindAddr   string   `yaml:"serverBindAddr"`
	ServerBindPort   string   `yaml:"serverBindPort"`
	ChownUsrGrp      string   `yaml:"chownUsrGrp"`
	excludeInRootDir map[string]Void
	excludeInTxmnDir map[string]Void
	uid              int
	gid              int
}

type MoveRequest struct {
	Location    string   `json:"location"`
	Collections []string `json:"collections"`
}

var void Void
var constants Constants

func loadConfiguration(filepath string) error {
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &constants)
	if err != nil {
		return err
	}

	constants.excludeInRootDir = map[string]Void{}
	constants.excludeInTxmnDir = map[string]Void{}

	for _, item := range constants.ExcRDir {
		constants.excludeInRootDir[item] = void
	}

	for _, item := range constants.ExcTDir {
		constants.excludeInTxmnDir[item] = void
	}
	uid_gid := strings.Split(constants.ChownUsrGrp, ":")
	if 2 != len(uid_gid) {
		log.Fatal("failed to parse chownUsrGrp, should be of format uid:gid (127:127)")
	}
	if constants.uid, err = strconv.Atoi(uid_gid[0]); nil != err {
		log.Fatal("failed to parse usr_id from chownUsrGrp, provided : " + uid_gid[0] + " should numeric int value")
	}
	if constants.gid, err = strconv.Atoi(uid_gid[1]); nil != err {
		log.Fatal("failed to parse grp_id from chownUsrGrp, provided : " + uid_gid[1] + " should numeric int value")
	}
	return nil
}

func doMvChown(what string, where string) error {
	entity := constants.RootDir + "/" + where + "/" + what
	if err := os.Rename(constants.DnDir+"/"+what, entity); nil != err {
		return err
	}
	return filepath.Walk(entity, func(name string, info os.FileInfo, err error) error {
		if nil == err {
			err = os.Chown(name, constants.uid, constants.gid)
		}
		return err
	})
}

func getList(root string, dirOnly bool, exclude map[string]Void) ([]string, error) {
	list := make([]string, 0)

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !dirOnly || (dirOnly && e.IsDir()) {
			if _, ok := exclude[e.Name()]; !ok {
				list = append(list, e.Name())
			}
		}
	}

	return list, nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	htmlFile := "index.html"
	htmlContent, err := os.ReadFile(htmlFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.New("").Parse(string(htmlContent)))

	collection, err := getList(constants.DnDir, false, constants.excludeInTxmnDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	location, err := getList(constants.RootDir, true, constants.excludeInRootDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Collection []string
		Location   []string
	}{
		Collection: collection,
		Location:   location,
	}

	w.Header().Set("Content-Type", "text/html")

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var moveRequest MoveRequest
	err := json.NewDecoder(r.Body).Decode(&moveRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := getList(constants.DnDir, false, constants.excludeInTxmnDir)
	collectionMap := map[string]Void{}
	for _, c := range collection {
		collectionMap[c] = void
	}

	location, err := getList(constants.RootDir, true, constants.excludeInRootDir)
	locationMap := map[string]Void{}
	for _, c := range location {
		locationMap[c] = void
	}

	//Make sure that we are only allowing the destination as listed from getList
	if _, ok := locationMap[moveRequest.Location]; ok {
		for _, item := range moveRequest.Collections {
			//Check to see if this value is from the list we get from getList
			if _, ok := collectionMap[item]; ok {
				if err := doMvChown(item, moveRequest.Location); nil != err {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Get the updated collection list
	collection, err = getList(constants.DnDir, false, constants.excludeInTxmnDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Collection []string `json:"collection"`
		Location   []string `json:"location"`
	}{
		Collection: collection,
		Location:   location,
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func isIPAllowed(ip string) bool {
	requestIP := net.ParseIP(ip)

	for _, subnet := range constants.AllowedCIDRs {
		_, subnetIPNet, err := net.ParseCIDR(subnet)
		if nil == err && subnetIPNet.Contains(requestIP) {
			return true
		}
	}

	return false
}

func ipRestrictionMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)

		if err != nil || !isIPAllowed(remoteIP) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	current_user, err := user.Current()
	if "root" != current_user.Username {
		log.Fatal("Must be run as root")
	}
	err = loadConfiguration("configuration.yaml")
	if err != nil {
		log.Fatal("Failed to load constants:", err)
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/move", handleMove)

	restrictedMux := http.NewServeMux()
	restrictedMux.HandleFunc("/", handleIndex)
	restrictedMux.HandleFunc("/move", handleMove)

	// Create a new HTTP server with the restricted middleware
	server := &http.Server{
		Addr:    constants.ServerBindAddr + ":" + constants.ServerBindPort,
		Handler: ipRestrictionMiddleware(restrictedMux),
	}

	log.Printf("Server listening on %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

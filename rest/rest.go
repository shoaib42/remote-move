package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/shoaib42/remote-move/io"
)

var indexContent []byte

type RemoteMoveREST interface {
	Serve()
	ipRestrictionMiddleware(next http.Handler) http.Handler
	handleData(w http.ResponseWriter, r *http.Request)
	handleMove(w http.ResponseWriter, r *http.Request)
}

type Handle struct {
	allowedCIDRs   []string
	serverBindAddr string
	serverBindPort string
	filedir        io.IOHelpers
}

type MoveOpertationResponse struct {
	Operation string `json:"operation"`
	Src       string `json:"source"`
	Dest      string `json:"destination"`
	Message   string `json:"message"`
}

type DataResponse struct {
	OpResponse           []MoveOpertationResponse `json:"opResponse"`
	ListingErrors        bool                     `json:"listingErrors"`
	SrcDirAndItsContents map[string][]string      `json:"srcDirAndItsContents"`
	Destination          []string                 `json:"destination"`
}

type MoveRequest struct {
	Src   string   `json:"src"`
	Items []string `json:"items"`
	Dest  string   `json:"dest"`
}

func validateIPCIDR(allowedCIDRs []string) ([]string, error) {

	okCIDRs := make([]string, 0)

	for _, ipOrCIDR := range allowedCIDRs {
		ip := net.ParseIP(ipOrCIDR)
		if ip != nil {
			cidr := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			okCIDRs = append(okCIDRs, cidr.String())
		} else {
			_, _, err := net.ParseCIDR(ipOrCIDR)
			if err == nil {
				okCIDRs = append(okCIDRs, ipOrCIDR)
			}
		}
	}
	if 0 == len(okCIDRs) {
		return nil, errors.New("No valid CIDRs or ip provided")
	}
	return okCIDRs, nil
}

func validateServerBind(bindAddr string, bindPort string) error {

	ip := net.ParseIP(bindAddr)
	if nil == ip {
		return errors.New("Invalid server bind address")
	}

	port, err := strconv.Atoi(bindPort)
	if nil != err {
		return errors.New("Invalid server bind port: " + err.Error())
	}
	if port < 1 || port > 65535 {
		return errors.New("Invalid server bind port. Port number must be between 1 and 65535")
	}

	return nil
}

func New(indexFilePath, bindArr, port string, allowedCIDRs []string, ioHelpers io.IOHelpers) (RemoteMoveREST, error) {
	file, err := os.Open(indexFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	indexContent, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err = validateServerBind(bindArr, port); nil != err {
		return nil, err
	}

	okCIDRs, err := validateIPCIDR(allowedCIDRs)

	if nil != err {
		return nil, err
	}

	return &Handle{
		allowedCIDRs:   okCIDRs,
		serverBindAddr: bindArr,
		serverBindPort: port,
		filedir:        ioHelpers,
	}, nil
}

func (h *Handle) ipRestrictionMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
		requestIP := net.ParseIP(remoteIP)
		isIPAllowed := false
		for _, subnet := range h.allowedCIDRs {
			_, subnetIPNet, err := net.ParseCIDR(subnet)
			if nil == err && subnetIPNet.Contains(requestIP) {
				isIPAllowed = true
				break
			}
		}

		if err != nil || !isIPAllowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handle) Serve() {
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	restrictedMux := http.NewServeMux()
	restrictedMux.HandleFunc("/", handleIndex)
	restrictedMux.HandleFunc("/move", h.handleMove)
	restrictedMux.HandleFunc("/data", h.handleData)
	restrictedMux.Handle("/static/", staticHandler)

	server := &http.Server{
		Addr:    h.serverBindAddr + ":" + h.serverBindPort,
		Handler: h.ipRestrictionMiddleware(restrictedMux),
	}
	server.ListenAndServe()
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader(indexContent))
}

func (h *Handle) responseData(w http.ResponseWriter, mor []MoveOpertationResponse) {
	mup, err := h.filedir.GetSrcMapItems()
	listingErrors := false
	if nil != err {
		mup = make(map[string][]string)
		listingErrors = true
	}

	ddir, err := h.filedir.GetDestDirList()

	if nil != err {
		ddir = make([]string, 0)
		listingErrors = true
	}
	data := DataResponse{
		OpResponse:           mor,
		ListingErrors:        listingErrors,
		SrcDirAndItsContents: mup,
		Destination:          ddir,
	}

	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, "Error responding data", http.StatusInternalServerError)
	}

}

func (h *Handle) handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Allow", "GET")
	w.Header().Set("Content-Type", "application/json")
	h.responseData(w, nil)
}

func (h *Handle) handleMove(w http.ResponseWriter, r *http.Request) {
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

	mor := make([]MoveOpertationResponse, 0)
	for _, i := range moveRequest.Items {
		if err = h.filedir.DoMvChown(moveRequest.Src, i, moveRequest.Dest); nil != err {
			mor = append(mor, MoveOpertationResponse{
				Src:       moveRequest.Src + "/" + i,
				Dest:      moveRequest.Dest,
				Operation: "move",
				Message:   err.Error(),
			})
		}
	}
	w.Header().Set("Allow", "POST")
	w.Header().Set("Content-Type", "application/json")

	h.responseData(w, mor)
}

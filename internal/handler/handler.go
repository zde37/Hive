package handler

import "net/http"

type Handler interface {
	Mux() *http.ServeMux
	health(w http.ResponseWriter, r *http.Request) error
	getNodeInfo(w http.ResponseWriter, r *http.Request) error
	pingNode(w http.ResponseWriter, r *http.Request) error
	addFile(w http.ResponseWriter, r *http.Request) error
	downloadFile(w http.ResponseWriter, r *http.Request) error
	getPeers(w http.ResponseWriter, r *http.Request) error
	listPins(w http.ResponseWriter, r *http.Request) error
	pinObject(w http.ResponseWriter, r *http.Request) error
	unPinObject(w http.ResponseWriter, r *http.Request) error
	displayFileContents(w http.ResponseWriter, r *http.Request) error
	downloadFolder(w http.ResponseWriter, r *http.Request) error
}

package handler

import "net/http"

// Handler is an interface that defines the methods for handling HTTP requests.
type Handler interface {
	Mux() *http.ServeMux
	Health(w http.ResponseWriter, r *http.Request) error
	GetNodeInfo(w http.ResponseWriter, r *http.Request) error
	PingNode(w http.ResponseWriter, r *http.Request) error
	AddFile(w http.ResponseWriter, r *http.Request) error
	DownloadFile(w http.ResponseWriter, r *http.Request) error
	ListNodes(w http.ResponseWriter, r *http.Request) error
	ListPins(w http.ResponseWriter, r *http.Request) error
	PinObject(w http.ResponseWriter, r *http.Request) error
	DeleteFile(w http.ResponseWriter, r *http.Request) error
	DisplayFileContents(w http.ResponseWriter, r *http.Request) error
	DownloadFolder(w http.ResponseWriter, r *http.Request) error
}

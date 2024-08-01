package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/zde37/Hive/internal/ipfs"
)

const maxUploadSize = 100 * 1024 * 1024 // 100MB in bytes

// handlerImpl implements the Handler interface and manages HTTP request handling.
type handlerImpl struct {
	ipfs     ipfs.Client
	server   *http.ServeMux
	validate *validator.Validate
}

// NewHandlerImpl creates and initializes a new Handler instance.
func NewHandlerImpl(ipfs ipfs.Client) Handler {
	mux := http.NewServeMux()
	handlerImpl := &handlerImpl{
		ipfs:     ipfs,
		server:   mux,
		validate: validator.New(),
	}

	handlerImpl.registerRoutes()
	return handlerImpl
}

// Mux returns the http.ServeMux associated with the handler.
func (h *handlerImpl) Mux() *http.ServeMux {
	return h.server
}

// registerRoutes sets up the routing for the handler.
func (h *handlerImpl) registerRoutes() {
	h.server.Handle("GET /hello-world", errorMiddleware(h.health))
	h.server.Handle("GET /info/{peerid}", errorMiddleware(h.getNodeInfo))
	// h.server.Handle("GET /ping/{peerid}", errorMiddleware(h.pingNode))
	h.server.Handle("GET /peers", errorMiddleware(h.getPeers))
	h.server.Handle("GET /file", errorMiddleware(h.downloadFile))
	// h.server.Handle("GET /cat/{cid}", errorMiddleware(h.displayFileContents))
	// h.server.Handle("GET /folder", errorMiddleware(h.downloadFolder))
	h.server.Handle("GET /pins", errorMiddleware(h.listPins))
	h.server.Handle("DELETE /file/{cid}", errorMiddleware(h.deleteFile))
	// h.server.Handle("POST /pin", errorMiddleware(h.pinObject))
	h.server.Handle("POST /file", errorMiddleware(h.addFile))
	// h.server.Handle("POST /folder", errorMiddleware(h.addFolder))
	h.serveStaticFiles()
	corsServer := corsMiddleware(h.server)

	v1 := http.NewServeMux()
	v1.Handle("/v1/", http.StripPrefix("/v1", corsServer))
	h.server = v1
}

func (h *handlerImpl) serveStaticFiles() {
	pages := []string{"home", "files", "nodes", "status"}
	for _, page := range pages {
		pageName := page
		h.server.HandleFunc(fmt.Sprintf("/%s", pageName), func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, fmt.Sprintf("./frontend/%s.html", pageName))
		})
	}
}

// healthHandler responds to health check requests.
func (h *handlerImpl) health(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Hello world")
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}
	return nil
}

func (h *handlerImpl) getNodeInfo(w http.ResponseWriter, r *http.Request) error {
	peerID := r.PathValue("peerid")
	if peerID == "" {
		return NewErrorStatus(fmt.Errorf("peerid is required"), http.StatusBadRequest, 0)
	}

	nodeInfo, err := h.ipfs.NodeInfo(r.Context(), peerID)
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(nodeInfo)
}

func (h *handlerImpl) pingNode(w http.ResponseWriter, r *http.Request) error {
	peerID := r.PathValue("peerid")
	if peerID == "" {
		return NewErrorStatus(fmt.Errorf("peerid is required"), http.StatusBadRequest, 0)
	}

	pingInfo, err := h.ipfs.Ping(r.Context(), peerID)
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
		Time    string `json:"time"`
	}{
		Success: pingInfo[len(pingInfo)-1].Success,
		Text:    pingInfo[0].Text,
		Time:    strings.ReplaceAll(pingInfo[len(pingInfo)-1].Text, "Average latency: ", ""),
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) addFile(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 mb
		return NewErrorStatus(err, http.StatusBadRequest, 0)
	}

	fileName := r.FormValue("name")
	if fileName == "" {
		return NewErrorStatus(fmt.Errorf("name is required"), http.StatusBadRequest, 0)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return NewErrorStatus(err, http.StatusBadRequest, 0)
	}
	defer file.Close()

	// Check if the file size exceeds the limit
	if header.Size > maxUploadSize {
		return NewErrorStatus(fmt.Errorf("file size exceeds the maximum limit of 100MB"), http.StatusBadRequest, 0)
	}

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy the uploaded file to the temporary file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	// TODO: avoid duplicate entries
	filePath, rootCid, err := h.ipfs.Add(r.Context(), fileName, tempFile.Name())
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		FilePath string `json:"file_path"`
		RootCid  string `json:"root_cid"`
	}{
		FilePath: filePath,
		RootCid:  rootCid,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(resp)
}

// TODO: work on this
func (h *handlerImpl) addFolder(w http.ResponseWriter, r *http.Request) error {
	reader, err := r.MultipartReader()
	if err != nil {
		return NewErrorStatus(err, http.StatusBadRequest, 0)
	}

	fileName := r.FormValue("name")
	if fileName == "" {
		return NewErrorStatus(fmt.Errorf("name is required"), http.StatusBadRequest, 0)
	}

	tempDir, err := os.MkdirTemp("", "upload-")
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}
	defer os.RemoveAll(tempDir)

	var totalSize int64
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return NewErrorStatus(err, http.StatusBadRequest, 0)
		}

		if part.FileName() == "" {
			continue // Skip non-file parts
		}

		dst, err := os.Create(filepath.Join(tempDir, part.FileName()))
		if err != nil {
			return NewErrorStatus(err, http.StatusInternalServerError, 1)
		}
		defer dst.Close()

		size, err := io.Copy(dst, part)
		if err != nil {
			return NewErrorStatus(err, http.StatusInternalServerError, 1)
		}
		totalSize += size
	}

	log.Println(totalSize)
	return nil
	if totalSize > maxUploadSize {
		return NewErrorStatus(fmt.Errorf("total upload size exceeds the maximum limit"), http.StatusBadRequest, 0)
	}

	filePath, rootCid, err := h.ipfs.Add(r.Context(), fileName, tempDir)
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		FilePath string `json:"file_path"`
		RootCid  string `json:"root_cid"`
		Size     int64  `json:"size"`
	}{
		FilePath: filePath,
		RootCid:  rootCid,
		Size:     totalSize,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) getPeers(w http.ResponseWriter, r *http.Request) error {
	peers, err := h.ipfs.GetConnectedPeers(r.Context())
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Peers []ipfs.Peer `json:"peers"`
		Total int         `json:"total"`
	}{
		Peers: peers,
		Total: len(peers),
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) listPins(w http.ResponseWriter, r *http.Request) error {
	pins, err := h.ipfs.ListPins(r.Context())
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Pins any `json:"pins"`
		// Total int        `json:"total"`
	}{
		Pins: pins,
		// Total: len(pins),
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) displayFileContents(w http.ResponseWriter, r *http.Request) error {
	cid := r.PathValue("cid")
	if cid == "" {
		return NewErrorStatus(fmt.Errorf("cid is required"), http.StatusBadRequest, 0)
	}

	path := fmt.Sprintf("/ipfs/%s", cid)
	content, err := h.ipfs.DisplayFileContent(r.Context(), path)
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Content string `json:"content"`
	}{
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) pinObject(w http.ResponseWriter, r *http.Request) error {
	name := r.FormValue("name")
	cid := r.FormValue("cid")
	if name == "" || cid == "" {
		return NewErrorStatus(fmt.Errorf("name and cid is required"), http.StatusBadRequest, 0)
	}

	path := fmt.Sprintf("/ipfs/%s", cid)
	if err := h.ipfs.PinObject(r.Context(), name, path); err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) deleteFile(w http.ResponseWriter, r *http.Request) error {
	cid := r.PathValue("cid")
	if cid == "" {
		return NewErrorStatus(fmt.Errorf("cid is required"), http.StatusBadRequest, 0)
	}

	cid = fmt.Sprintf("/ipfs/%s", cid)
	if err := h.ipfs.DeleteFile(r.Context(), cid); err != nil {
		if strings.HasPrefix(err.Error(), "..") {
			return NewErrorStatus(err, http.StatusBadRequest, 0)
		}
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Status string `json:"status"`
	}{
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func (h *handlerImpl) downloadFile(w http.ResponseWriter, r *http.Request) error {
	cid := r.URL.Query().Get("cid")
	if cid == "" {
		return NewErrorStatus(fmt.Errorf("cid is required"), http.StatusBadRequest, 0)
	}

	fileData, err := h.ipfs.DownloadFile(r.Context(), cid)
	if err != nil {
		return NewErrorStatus(fmt.Errorf("cid is required"), http.StatusInternalServerError, 1)
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+cid)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(fileData)
	return nil
}

func (h *handlerImpl) downloadFolder(w http.ResponseWriter, r *http.Request) error {
	return nil
}

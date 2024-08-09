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

	"github.com/zde37/Hive/internal/ipfs"
)

const maxUploadSize = 100 * 1024 * 1024 // 100MB in bytes

// handlerImpl implements the Handler interface and manages HTTP request handling.
type handlerImpl struct {
	ipfs   ipfs.Client
	server *http.ServeMux
}

// NewHandlerImpl creates and initializes a new Handler instance.
func NewHandlerImpl(ipfs ipfs.Client) Handler {
	mux := http.NewServeMux()
	handlerImpl := &handlerImpl{
		ipfs:   ipfs,
		server: mux,
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
	h.server.Handle("GET /hello-world", errorMiddleware(h.Health))
	h.server.Handle("GET /info/{peerid}", errorMiddleware(h.GetNodeInfo))
	h.server.Handle("GET /peers", errorMiddleware(h.ListNodes))
	h.server.Handle("GET /file", errorMiddleware(h.DownloadFile))
	h.server.Handle("GET /pins", errorMiddleware(h.ListPins))
	h.server.Handle("DELETE /file/{cid}", errorMiddleware(h.DeleteFile))
	h.server.Handle("POST /file", errorMiddleware(h.AddFile))

	// h.server.Handle("GET /ping/{peerid}", errorMiddleware(h.PingNode))
	// h.server.Handle("GET /cat/{cid}", errorMiddleware(h.DisplayFileContents))
	// h.server.Handle("GET /folder", errorMiddleware(h.DownloadFolder))

	// h.server.Handle("POST /pin", errorMiddleware(h.PinObject))
	// h.server.Handle("POST /folder", errorMiddleware(h.AddFolder))
	h.serveStaticFiles()
	corsServer := corsMiddleware(h.server)

	v1 := http.NewServeMux()
	v1.Handle("/v1/", http.StripPrefix("/v1", corsServer))
	h.server = v1
}

// serveStaticFiles sets up routes to serve static HTML files for the application's pages.
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
func (h *handlerImpl) Health(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Hello world")
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}
	return nil
}

// getNodeInfo retrieves information about the IPFS node identified by the provided peer ID.
func (h *handlerImpl) GetNodeInfo(w http.ResponseWriter, r *http.Request) error {
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

// pingNode handles a request to ping an IPFS node identified by the provided peer ID.
func (h *handlerImpl) PingNode(w http.ResponseWriter, r *http.Request) error {
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

// addFile handles the upload of a file to the IPFS network.
func (h *handlerImpl) AddFile(w http.ResponseWriter, r *http.Request) error {
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

// addFolder handles the upload of a folder to the IPFS network. // TODO: work on this
func (h *handlerImpl) AddFolder(w http.ResponseWriter, r *http.Request) error {
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

// listNodes is an HTTP handler that returns a list of connected IPFS nodes.
func (h *handlerImpl) ListNodes(w http.ResponseWriter, r *http.Request) error {
	nodes, err := h.ipfs.ListConnectedNodes(r.Context())
	if err != nil {
		return NewErrorStatus(err, http.StatusInternalServerError, 1)
	}

	resp := struct {
		Nodes []ipfs.Node `json:"nodes"`
		Total int         `json:"total"`
	}{
		Nodes: nodes,
		Total: len(nodes),
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

// listPins is an HTTP handler that returns a list of pinned IPFS objects.
func (h *handlerImpl) ListPins(w http.ResponseWriter, r *http.Request) error {
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

// displayFileContents is an HTTP handler that retrieves the content of a file
// identified by the provided CID (Content Identifier) and returns it as a JSON response.
func (h *handlerImpl) DisplayFileContents(w http.ResponseWriter, r *http.Request) error {
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

// pinObject handles a request to pin an IPFS object to the node.
func (h *handlerImpl) PinObject(w http.ResponseWriter, r *http.Request) error {
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

// deleteFile is an HTTP handler that deletes an IPFS file identified by the provided CID (Content Identifier).
func (h *handlerImpl) DeleteFile(w http.ResponseWriter, r *http.Request) error {
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

// downloadFile handles a request to download a file from the IPFS node.
func (h *handlerImpl) DownloadFile(w http.ResponseWriter, r *http.Request) error {
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

// downloadFolder is an HTTP handler that downloads an IPFS folder identified by the provided CID (Content Identifier).
func (h *handlerImpl) DownloadFolder(w http.ResponseWriter, r *http.Request) error {
	return nil
}

package api

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/chunk"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/metrics"
	"cloudWeave/internal/storage"
)

//go:embed dashboard.html
var DashboardHTML []byte

const DefaultChunkSize = 1024 * 1024 // 1 MB default chunk size

// PeerManager abstracts cluster node membership for peer broadcasts and dynamic joins/leaves.
type PeerManager interface {
	GetActiveNodes() []string
	GetAllNodes() []string
	AddNode(addr string)
	RemoveNode(addr string)
}

// ChunkStorageEngine interface abstracts storing/fetching individual chunks locally or across quorum.
type ChunkStorageEngine interface {
	PutChunk(chunkID string, data []byte) ([]string, error) // returns node locations where chunk was stored
	GetChunk(chunkID string, locations []string) ([]byte, error)
}

// LocalStorageAdapter adapts DiskStore to ChunkStorageEngine interface.
type LocalStorageAdapter struct {
	PutFunc func(chunkID string, data []byte) error
	GetFunc func(chunkID string) ([]byte, error)
	NodeID  string
}

func (l *LocalStorageAdapter) PutChunk(chunkID string, data []byte) ([]string, error) {
	if err := l.PutFunc(chunkID, data); err != nil {
		return nil, err
	}
	return []string{l.NodeID}, nil
}

func (l *LocalStorageAdapter) GetChunk(chunkID string, locations []string) ([]byte, error) {
	return l.GetFunc(chunkID)
}

type APIHandler struct {
	metaStore      *metadata.Store
	engine         ChunkStorageEngine
	chunkSize      int
	auth           *auth.Authenticator
	peerMgr        PeerManager
	localAddr      string
	httpClient     *http.Client
	wal            *metadata.WAL
	clusterSecret  string
	inFlight       *storage.InFlightRegistry
	diskStore      *storage.DiskStore
}

func NewAPIHandler(metaStore *metadata.Store, engine ChunkStorageEngine, chunkSize int) *APIHandler {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &APIHandler{
		metaStore:  metaStore,
		engine:     engine,
		chunkSize:  chunkSize,
		auth:       auth.NewDefaultAuthenticator(),
		httpClient: &http.Client{Timeout: 3 * time.Second},
		inFlight:   storage.NewInFlightRegistry(),
	}
}

func (a *APIHandler) SetInFlightRegistry(reg *storage.InFlightRegistry) {
	a.inFlight = reg
}

func (a *APIHandler) GetInFlightRegistry() *storage.InFlightRegistry {
	return a.inFlight
}

func (a *APIHandler) SetDiskStore(store *storage.DiskStore) {
	a.diskStore = store
}

func (a *APIHandler) SetHTTPClient(client *http.Client) {
	if client != nil {
		a.httpClient = client
	}
}

func (a *APIHandler) SetWAL(w *metadata.WAL) {
	a.wal = w
}

func (a *APIHandler) GetWAL() *metadata.WAL {
	return a.wal
}

func (a *APIHandler) GetMetaStore() *metadata.Store {
	return a.metaStore
}

func (a *APIHandler) GetEngine() ChunkStorageEngine {
	return a.engine
}

func (a *APIHandler) GetChunkSize() int {
	return a.chunkSize
}

func (a *APIHandler) SetAuthenticator(authenticator *auth.Authenticator) {
	if authenticator != nil {
		a.auth = authenticator
	}
}

func (a *APIHandler) SetPeerManager(peerMgr PeerManager, localAddr string) {
	a.peerMgr = peerMgr
	a.localAddr = localAddr
}

func (a *APIHandler) GetPeerManager() PeerManager {
	return a.peerMgr
}

func (a *APIHandler) GetLocalAddr() string {
	return a.localAddr
}

func (a *APIHandler) SetClusterSecret(secret string) {
	a.clusterSecret = secret
}

// validateInput checks namespace and fileID for path traversal characters.
func validateInput(ns, fileID string) bool {
	for _, s := range []string{ns, fileID} {
		if strings.Contains(s, "..") || strings.Contains(s, "\x00") || strings.Contains(s, "\\") {
			return false
		}
	}
	return true
}

func (a *APIHandler) HandleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Namespace")

	log.Printf("[HandleFiles] Incoming request: Method=%s, Path=%s, Auth=%s", r.Method, r.URL.Path, r.Header.Get("Authorization"))

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if a.auth != nil {
		key := auth.ExtractKey(r)
		cred, ok := a.auth.ValidateKey(key)
		if !ok {
			http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
			return
		}

		namespace, fileID := auth.ExtractNamespaceAndFileID(r)
		if fileID == "" {
			http.Error(w, "missing file id", http.StatusBadRequest)
			return
		}

		if !cred.CanAccessNamespace(namespace) {
			http.Error(w, "forbidden: insufficient permissions for namespace", http.StatusForbidden)
			return
		}

		if !validateInput(namespace, fileID) {
			http.Error(w, "invalid namespace or file ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut, http.MethodPost:
			a.handlePutFile(w, r, namespace, fileID)
		case http.MethodGet:
			a.handleGetFile(w, r, namespace, fileID)
		case http.MethodDelete:
			a.handleDeleteFile(w, r, namespace, fileID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	_, fileID := auth.ExtractNamespaceAndFileID(r)
	if fileID == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		a.handlePutFile(w, r, "default", fileID)
	case http.MethodGet:
		a.handleGetFile(w, r, "default", fileID)
	case http.MethodDelete:
		a.handleDeleteFile(w, r, "default", fileID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *APIHandler) handlePutFile(w http.ResponseWriter, r *http.Request, namespace, fileID string) {
	if r.Body == nil {
		http.Error(w, "missing request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var inFlightRegistered []string
	var regMu sync.Mutex
	defer func() {
		regMu.Lock()
		if a.inFlight != nil && len(inFlightRegistered) > 0 {
			a.inFlight.Unregister(inFlightRegistered...)
		}
		regMu.Unlock()
	}()

	var chunkIDs []string
	chunkLocations := make(map[string][]string)
	var mu sync.Mutex

	const numWorkers = 8
	jobs := make(chan chunk.Chunk, numWorkers)
	errChan := make(chan error, 1)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				locs, err := a.engine.PutChunk(c.ID, c.Data)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("failed to store chunk %s: %w", c.ID, err):
					default:
					}
					return
				}
				mu.Lock()
				chunkLocations[c.ID] = locs
				mu.Unlock()
			}
		}()
	}

	totalBytes, chunkIDs, err := chunk.SplitStream(r.Body, a.chunkSize, func(c chunk.Chunk) error {
		select {
		case err := <-errChan:
			return err
		default:
		}
		if a.inFlight != nil {
			a.inFlight.Register(c.ID)
			regMu.Lock()
			inFlightRegistered = append(inFlightRegistered, c.ID)
			regMu.Unlock()
		}
		jobs <- c
		return nil
	})

	close(jobs)
	wg.Wait()

	if err == nil {
		select {
		case err = <-errChan:
		default:
		}
	}

	if err != nil {
		log.Printf("streaming put file %s failed: %v", fileID, err)
		http.Error(w, "upload failed", http.StatusBadRequest)
		return
	}

	// Extract Content-Type and custom X-Meta-* / X-Object-Meta-* metadata
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	customMeta := make(map[string]string)
	for headerKey, values := range r.Header {
		lowerKey := strings.ToLower(headerKey)
		if strings.HasPrefix(lowerKey, "x-meta-") {
			metaKey := strings.TrimPrefix(lowerKey, "x-meta-")
			if len(values) > 0 {
				customMeta[metaKey] = values[0]
			}
		} else if strings.HasPrefix(lowerKey, "x-object-meta-") {
			metaKey := strings.TrimPrefix(lowerKey, "x-object-meta-")
			if len(values) > 0 {
				customMeta[metaKey] = values[0]
			}
		}
	}

	manifest := metadata.Manifest{
		Namespace:      namespace,
		FileID:         fileID,
		Size:           totalBytes,
		ChunkIDs:       chunkIDs,
		ChunkLocations: chunkLocations,
		ContentType:    contentType,
		Metadata:       customMeta,
	}

	if err := a.metaStore.RecordPlacement(manifest); err != nil {
		log.Printf("failed to record manifest: %v", err)
		http.Error(w, "failed to record metadata", http.StatusInternalServerError)
		return
	}

	if recordedManifest, found := a.metaStore.LookupScoped(namespace, fileID); found {
		a.BroadcastManifest(recordedManifest)
	} else {
		a.BroadcastManifest(manifest)
	}

	metrics.IncFileUploads()

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File %s uploaded successfully (%d bytes, %d chunks)\n", fileID, totalBytes, len(chunkIDs))
}

func (a *APIHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request, namespace, fileID string) {
	manifestToDel, found := a.metaStore.LookupScoped(namespace, fileID)
	if !found {
		manifestToDel, found = a.metaStore.Lookup(fileID)
	}
	if !found {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	deleted := a.metaStore.DeleteScoped(namespace, fileID)
	if !deleted {
		deleted = a.metaStore.Delete(fileID)
	}
	if !deleted {
		http.Error(w, "failed to delete metadata", http.StatusInternalServerError)
		return
	}

	if a.diskStore != nil {
		for _, cid := range manifestToDel.ChunkIDs {
			a.diskStore.InvalidateChunk(cid)
		}
	}

	a.BroadcastDelete(namespace, fileID)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File %s deleted successfully\n", fileID)
}

func (a *APIHandler) handleGetFile(w http.ResponseWriter, r *http.Request, namespace, fileID string) {
	if r.URL.Query().Get("versions") == "true" {
		versions, found := a.metaStore.ListVersions(namespace, fileID)
		if !found {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
		return
	}

	var manifest metadata.Manifest
	var found bool

	versionID := r.URL.Query().Get("version_id")
	if versionID == "" {
		versionID = r.URL.Query().Get("versionId")
	}

	if versionID != "" {
		manifest, found = a.metaStore.LookupVersion(namespace, fileID, versionID)
	} else {
		manifest, found = a.metaStore.LookupScoped(namespace, fileID)
		if !found {
			manifest, found = a.metaStore.Lookup(fileID)
		}
	}

	if !found {
		http.Error(w, "file or version not found", http.StatusNotFound)
		return
	}

	metrics.IncFileDownloads()

	// Populate metadata headers
	if manifest.ContentType != "" {
		w.Header().Set("Content-Type", manifest.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	for k, v := range manifest.Metadata {
		w.Header().Set("X-Meta-"+k, v)
	}

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		a.handleRangeGetFile(w, r, manifest, rangeHeader)
		return
	}

	// Full file streaming GET response
	w.Header().Set("Content-Length", strconv.FormatInt(manifest.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)

	type getResult struct {
		data []byte
		err  error
	}

	numChunks := len(manifest.ChunkIDs)
	if numChunks == 0 {
		return
	}

	if numChunks == 1 {
		chunkID := manifest.ChunkIDs[0]
		locs := manifest.ChunkLocations[chunkID]
		data, err := a.engine.GetChunk(chunkID, locs)
		if err != nil {
			log.Printf("failed to retrieve chunk %s: %v", chunkID, err)
			return
		}
		w.Write(data)
		return
	}

	results := make([]chan getResult, numChunks)
	for i := 0; i < numChunks; i++ {
		results[i] = make(chan getResult, 1)
	}

	const numGetWorkers = 8
	jobs := make(chan int, numChunks)
	for i := 0; i < numChunks; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < numGetWorkers && w < numChunks; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				chunkID := manifest.ChunkIDs[idx]
				locs := manifest.ChunkLocations[chunkID]
				data, err := a.engine.GetChunk(chunkID, locs)
				results[idx] <- getResult{data: data, err: err}
			}
		}()
	}

	go func() {
		wg.Wait()
	}()

	for idx := 0; idx < numChunks; idx++ {
		res := <-results[idx]
		if res.err != nil {
			log.Printf("failed to retrieve chunk %s (index %d): %v", manifest.ChunkIDs[idx], idx, res.err)
			return
		}
		if _, writeErr := w.Write(res.data); writeErr != nil {
			log.Printf("client disconnected during streaming read of file %s: %v", fileID, writeErr)
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (a *APIHandler) handleRangeGetFile(w http.ResponseWriter, r *http.Request, manifest metadata.Manifest, rangeHeader string) {
	start, end, err := parseRangeHeader(rangeHeader, manifest.Size)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", manifest.Size))
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1
	startChunkIdx := int(start / int64(a.chunkSize))
	endChunkIdx := int(end / int64(a.chunkSize))

	if manifest.ContentType != "" {
		w.Header().Set("Content-Type", manifest.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	for k, v := range manifest.Metadata {
		w.Header().Set("X-Meta-"+k, v)
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, manifest.Size))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	flusher, _ := w.(http.Flusher)

	for chunkIdx := startChunkIdx; chunkIdx <= endChunkIdx && chunkIdx < len(manifest.ChunkIDs); chunkIdx++ {
		chunkID := manifest.ChunkIDs[chunkIdx]
		locs := manifest.ChunkLocations[chunkID]

		data, err := a.engine.GetChunk(chunkID, locs)
		if err != nil {
			log.Printf("failed to retrieve chunk %s for range request: %v", chunkID, err)
			return
		}

		chunkStart := int64(chunkIdx) * int64(a.chunkSize)
		chunkEnd := chunkStart + int64(len(data)) - 1

		sliceStart := int64(0)
		if start > chunkStart {
			sliceStart = start - chunkStart
		}

		sliceEnd := int64(len(data))
		if end < chunkEnd {
			sliceEnd = end - chunkStart + 1
		}

		if sliceStart < 0 || sliceStart > sliceEnd || sliceEnd > int64(len(data)) {
			continue
		}

		if _, writeErr := w.Write(data[sliceStart:sliceEnd]); writeErr != nil {
			log.Printf("client disconnected during range read of file %s: %v", manifest.FileID, writeErr)
			return
		}

		if flusher != nil {
			flusher.Flush()
		}
	}
}

// parseRangeHeader parses HTTP Range header "bytes=start-end" against total file size
func parseRangeHeader(rangeHeader string, totalSize int64) (int64, int64, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range unit")
	}

	spec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(spec, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	var start, end int64
	var err error

	if parts[0] == "" && parts[1] != "" {
		// Range: bytes=-num (last num bytes)
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffixLen <= 0 {
			return 0, 0, fmt.Errorf("invalid range suffix")
		}
		start = totalSize - suffixLen
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else if parts[0] != "" && parts[1] == "" {
		// Range: bytes=start-
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= totalSize {
			return 0, 0, fmt.Errorf("invalid range start")
		}
		end = totalSize - 1
	} else if parts[0] != "" && parts[1] != "" {
		// Range: bytes=start-end
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= totalSize {
			return 0, 0, fmt.Errorf("invalid range start")
		}
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid range end")
		}
		if end >= totalSize {
			end = totalSize - 1
		}
	} else {
		return 0, 0, fmt.Errorf("empty range spec")
	}

	return start, end, nil
}

// BroadcastManifest sends manifest record updates to all active peers in the cluster.
func (a *APIHandler) BroadcastManifest(m metadata.Manifest) {
	if a.peerMgr == nil {
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, peer := range a.peerMgr.GetActiveNodes() {
		if peer == a.localAddr || peer == "" {
			continue
		}
		wg.Add(1)
		go func(targetPeer string) {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, targetPeer+"/internal/manifest", bytes.NewReader(data))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if a.clusterSecret != "" {
				req.Header.Set("X-Cluster-Secret", a.clusterSecret)
			}
			resp, err := a.httpClient.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
	wg.Wait()
}

// BroadcastDelete sends manifest deletion updates to all active peers in the cluster.
func (a *APIHandler) BroadcastDelete(ns, fileID string) {
	if a.peerMgr == nil {
		return
	}
	targetURL := "/internal/manifest?namespace=" + url.QueryEscape(ns) + "&file_id=" + url.QueryEscape(fileID)
	for _, peer := range a.peerMgr.GetActiveNodes() {
		if peer == a.localAddr || peer == "" {
			continue
		}
		go func(targetPeer string) {
			req, err := http.NewRequest(http.MethodDelete, targetPeer+targetURL, nil)
			if err != nil {
				return
			}
			if a.clusterSecret != "" {
				req.Header.Set("X-Cluster-Secret", a.clusterSecret)
			}
			resp, err := a.httpClient.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

// BroadcastJoin informs all peers that a new node has joined the cluster.
func (a *APIHandler) BroadcastJoin(nodeAddr string) {
	if a.peerMgr == nil {
		return
	}
	for _, peer := range a.peerMgr.GetActiveNodes() {
		if peer == a.localAddr || peer == nodeAddr || peer == "" {
			continue
		}
		go func(targetPeer string) {
			req, err := http.NewRequest(http.MethodPost, targetPeer+"/internal/join?node_addr="+url.QueryEscape(nodeAddr), nil)
			if err != nil {
				return
			}
			if a.clusterSecret != "" {
				req.Header.Set("X-Cluster-Secret", a.clusterSecret)
			}
			resp, err := a.httpClient.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

// BroadcastLeave informs all peers that a node has left the cluster.
func (a *APIHandler) BroadcastLeave(nodeAddr string) {
	if a.peerMgr == nil {
		return
	}
	for _, peer := range a.peerMgr.GetActiveNodes() {
		if peer == a.localAddr || peer == "" {
			continue
		}
		go func(targetPeer string) {
			req, err := http.NewRequest(http.MethodPost, targetPeer+"/internal/leave?node_addr="+url.QueryEscape(nodeAddr), nil)
			if err != nil {
				return
			}
			if a.clusterSecret != "" {
				req.Header.Set("X-Cluster-Secret", a.clusterSecret)
			}
			resp, err := a.httpClient.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

// HandleInternalManifest processes node-to-node metadata replication requests.
func (a *APIHandler) HandleInternalManifest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.metaStore == nil {
			http.Error(w, "metadata store not initialized", http.StatusInternalServerError)
			return
		}
		all := a.metaStore.GetAllManifests()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(all)
	case http.MethodPost:
		var m metadata.Manifest
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = a.metaStore.RecordPlacement(m)
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		ns := r.URL.Query().Get("namespace")
		fileID := r.URL.Query().Get("file_id")
		a.metaStore.DeleteScoped(ns, fileID)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleInternalJoin receives cluster node join notifications from peers.
func (a *APIHandler) HandleInternalJoin(w http.ResponseWriter, r *http.Request) {
	nodeAddr := r.URL.Query().Get("node_addr")
	if nodeAddr != "" && a.peerMgr != nil {
		a.peerMgr.AddNode(nodeAddr)
	}
	w.WriteHeader(http.StatusOK)
}

// HandleInternalLeave receives cluster node leave notifications from peers.
func (a *APIHandler) HandleInternalLeave(w http.ResponseWriter, r *http.Request) {
	nodeAddr := r.URL.Query().Get("node_addr")
	if nodeAddr != "" && a.peerMgr != nil {
		a.peerMgr.RemoveNode(nodeAddr)
	}
	w.WriteHeader(http.StatusOK)
}

// HandleDashboard serves the web dashboard single-page interface.
func (a *APIHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	w.Write(DashboardHTML)
}

// HandleClusterStatus returns real-time cluster health and Prometheus metrics in JSON format.
func (a *APIHandler) HandleClusterStatus(w http.ResponseWriter, r *http.Request) {
	type NodeInfo struct {
		Addr    string `json:"addr"`
		IsLocal bool   `json:"is_local"`
		IsAlive bool   `json:"is_alive"`
	}

	var activeNodes []string
	var allConfigured []string

	if a.peerMgr != nil {
		activeNodes = a.peerMgr.GetActiveNodes()
		allConfigured = a.peerMgr.GetAllNodes()
	}
	if len(allConfigured) == 0 && a.localAddr != "" {
		allConfigured = []string{a.localAddr}
		activeNodes = []string{a.localAddr}
	}

	activeSet := make(map[string]bool)
	for _, addr := range activeNodes {
		activeSet[addr] = true
	}

	var allNodes []NodeInfo
	seen := make(map[string]bool)
	for _, addr := range allConfigured {
		if addr != "" && !seen[addr] {
			seen[addr] = true
			allNodes = append(allNodes, NodeInfo{
				Addr:    addr,
				IsLocal: addr == a.localAddr,
				IsAlive: activeSet[addr],
			})
		}
	}

	type StoredObject struct {
		FileID     string `json:"file_id"`
		Namespace  string `json:"namespace"`
		Size       int64  `json:"size"`
		ChunkCount int    `json:"chunk_count"`
	}

	var storedObjects []StoredObject
	var totalBytes int64

	if a.metaStore != nil {
		allManifests := a.metaStore.GetAllManifests()
		for _, m := range allManifests {
			totalBytes += m.Size
			storedObjects = append(storedObjects, StoredObject{
				FileID:     m.FileID,
				Namespace:  m.Namespace,
				Size:       m.Size,
				ChunkCount: len(m.ChunkIDs),
			})
		}
	}

	status := map[string]interface{}{
		"active_nodes":          activeNodes,
		"all_nodes":             allNodes,
		"file_uploads_total":    atomic.LoadUint64(&metrics.DefaultMetrics.FileUploadsTotal),
		"file_downloads_total":  atomic.LoadUint64(&metrics.DefaultMetrics.FileDownloadsTotal),
		"repaired_chunks_total": atomic.LoadUint64(&metrics.DefaultMetrics.RepairedChunksTotal),
		"total_files_stored":   len(storedObjects),
		"total_bytes_stored":   totalBytes,
		"stored_objects":       storedObjects,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleAdminKill simulates node failure for live cluster self-healing demonstration.
func (a *APIHandler) HandleAdminKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetAddr := r.URL.Query().Get("node_addr")
	if targetAddr == "" {
		targetAddr = r.FormValue("node_addr")
	}

	if targetAddr != "" && a.peerMgr != nil {
		a.peerMgr.RemoveNode(targetAddr)
		a.BroadcastLeave(targetAddr)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node %s simulated failure trigger sent\n", targetAddr)
}

// HandleInternalRevokeKey receives cluster-wide key revocation broadcasts from peers.
func (a *APIHandler) HandleInternalRevokeKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	keyHash := r.URL.Query().Get("key_hash")
	if keyHash == "" {
		http.Error(w, "missing key_hash parameter", http.StatusBadRequest)
		return
	}
	if a.auth != nil {
		a.auth.RevokeCredentialByHash(keyHash)
	}
	w.WriteHeader(http.StatusOK)
}

// BroadcastKeyRevocation sends key revocation to all active peers.
func (a *APIHandler) BroadcastKeyRevocation(keyHash string) {
	if a.peerMgr == nil {
		return
	}
	for _, peer := range a.peerMgr.GetActiveNodes() {
		if peer == a.localAddr || peer == "" {
			continue
		}
		go func(targetPeer string) {
			req, err := http.NewRequest(http.MethodPost, targetPeer+"/internal/revoke-key?key_hash="+url.QueryEscape(keyHash), nil)
			if err != nil {
				return
			}
			if a.clusterSecret != "" {
				req.Header.Set("X-Cluster-Secret", a.clusterSecret)
			}
			resp, err := a.httpClient.Do(req)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}


package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/metrics"
)

type GCRunner interface {
	CollectGarbage() (int, error)
}

func jsonReader(v interface{}) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func NewRouter(apiHandler *APIHandler, transportHandler http.Handler, gcRunner GCRunner, authOpts ...*auth.Authenticator) http.Handler {
	mux := http.NewServeMux()

	var authenticator *auth.Authenticator
	if len(authOpts) > 0 && authOpts[0] != nil {
		authenticator = authOpts[0]
	} else {
		authenticator = auth.NewDefaultAuthenticator()
	}

	if apiHandler != nil {
		apiHandler.SetAuthenticator(authenticator)
		mux.HandleFunc("/files/", apiHandler.HandleFiles)
		mux.HandleFunc("/internal/manifest", apiHandler.HandleInternalManifest)
		mux.HandleFunc("/internal/join", apiHandler.HandleInternalJoin)
		mux.HandleFunc("/internal/leave", apiHandler.HandleInternalLeave)
		mux.HandleFunc("/dashboard", apiHandler.HandleDashboard)
		mux.HandleFunc("/dashboard/", apiHandler.HandleDashboard)
		mux.HandleFunc("/cluster/status", apiHandler.HandleClusterStatus)
		mux.HandleFunc("/admin/kill", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				cred, ok := authenticator.ValidateKey(key)
				if !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
				if !cred.IsAdmin && !cred.CanAccessNamespace("*") {
					http.Error(w, "forbidden: admin privileges required", http.StatusForbidden)
					return
				}
			}
			apiHandler.HandleAdminKill(w, r)
		})

		mux.HandleFunc("/admin/join", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				cred, ok := authenticator.ValidateKey(key)
				if !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
				if !cred.IsAdmin && !cred.CanAccessNamespace("*") {
					http.Error(w, "forbidden: admin privileges required", http.StatusForbidden)
					return
				}
			}

			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			nodeAddr := r.URL.Query().Get("node_addr")
			if nodeAddr == "" {
				nodeAddr = r.FormValue("node_addr")
			}
			if nodeAddr == "" {
				http.Error(w, "missing node_addr parameter", http.StatusBadRequest)
				return
			}

			if pm := apiHandler.GetPeerManager(); pm != nil {
				pm.AddNode(nodeAddr)
				apiHandler.BroadcastJoin(nodeAddr)

				if apiHandler.metaStore != nil {
					allManifests := apiHandler.metaStore.GetAllManifests()
					for _, m := range allManifests {
						if req, err := http.NewRequest(http.MethodPost, nodeAddr+"/internal/manifest", jsonReader(m)); err == nil {
							req.Header.Set("Content-Type", "application/json")
							if resp, err := apiHandler.httpClient.Do(req); err == nil && resp != nil {
								resp.Body.Close()
							}
						}
					}
				}

				for _, peer := range pm.GetActiveNodes() {
					if req, err := http.NewRequest(http.MethodPost, nodeAddr+"/internal/join?node_addr="+peer, nil); err == nil {
						if resp, err := apiHandler.httpClient.Do(req); err == nil && resp != nil {
							resp.Body.Close()
						}
					}
				}
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Node %s joined cluster successfully\n", nodeAddr)
		})

		mux.HandleFunc("/admin/leave", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				cred, ok := authenticator.ValidateKey(key)
				if !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
				if !cred.IsAdmin && !cred.CanAccessNamespace("*") {
					http.Error(w, "forbidden: admin privileges required", http.StatusForbidden)
					return
				}
			}

			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			nodeAddr := r.URL.Query().Get("node_addr")
			if nodeAddr == "" {
				nodeAddr = r.FormValue("node_addr")
			}
			if nodeAddr == "" {
				http.Error(w, "missing node_addr parameter", http.StatusBadRequest)
				return
			}

			if pm := apiHandler.GetPeerManager(); pm != nil {
				pm.RemoveNode(nodeAddr)
				apiHandler.BroadcastLeave(nodeAddr)
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Node %s left cluster gracefully\n", nodeAddr)
		})

		// GET /cluster/nodes (requires valid auth key)
		mux.HandleFunc("/cluster/nodes", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				if _, ok := authenticator.ValidateKey(key); !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
			}

			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var nodes []string
			if pm := apiHandler.GetPeerManager(); pm != nil {
				nodes = pm.GetActiveNodes()
			}
			if len(nodes) == 0 && apiHandler.GetLocalAddr() != "" {
				nodes = []string{apiHandler.GetLocalAddr()}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(nodes)
		})

		// /admin/keys (strictly admin-gated: GET, POST, DELETE)
		mux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				cred, ok := authenticator.ValidateKey(key)
				if !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
				if !cred.IsAdmin && !cred.CanAccessNamespace("*") {
					http.Error(w, "forbidden: admin privileges required", http.StatusForbidden)
					return
				}
			}

			switch r.Method {
			case http.MethodPost:
				var req struct {
					Namespaces []string `json:"namespaces"`
					IsAdmin    bool     `json:"is_admin"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)

				rawKey, err := auth.GenerateRandomKey()
				if err != nil {
					http.Error(w, "failed to generate key", http.StatusInternalServerError)
					return
				}

				keyHash := auth.HashKey(rawKey)
				cred := auth.Credential{
					KeyHash:    keyHash,
					Namespaces: req.Namespaces,
					IsAdmin:    req.IsAdmin,
				}

				if authenticator != nil {
					authenticator.AddCredentialByHash(cred)
				}

				if wal := apiHandler.GetWAL(); wal != nil {
					_ = wal.WriteRecord(metadata.WALRecord{
						Op:         metadata.OpRecordKey,
						Credential: cred,
					})
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"key":        rawKey, // Returned ONCE to client
					"key_hash":   keyHash,
					"namespaces": cred.Namespaces,
					"is_admin":   cred.IsAdmin,
				})

			case http.MethodDelete:
				keyParam := r.URL.Query().Get("key")
				hashParam := r.URL.Query().Get("key_hash")
				if keyParam == "" && hashParam == "" {
					http.Error(w, "missing key or key_hash parameter", http.StatusBadRequest)
					return
				}

				targetHash := hashParam
				if targetHash == "" {
					targetHash = auth.HashKey(keyParam)
				}

				if authenticator != nil {
					authenticator.RevokeCredentialByHash(targetHash)
				}

				if wal := apiHandler.GetWAL(); wal != nil {
					_ = wal.WriteRecord(metadata.WALRecord{
						Op: metadata.OpDeleteKey,
						Credential: auth.Credential{
							KeyHash: targetHash,
						},
					})
				}

				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Key hash %s revoked successfully\n", targetHash)

			case http.MethodGet:
				var creds []auth.Credential
				if authenticator != nil {
					creds = authenticator.GetAllCredentials()
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(creds)

			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	if transportHandler != nil {
		mux.Handle("/chunks/", transportHandler)
	}

	if gcRunner != nil {
		mux.HandleFunc("/admin/gc", func(w http.ResponseWriter, r *http.Request) {
			if authenticator != nil {
				key := auth.ExtractKey(r)
				cred, ok := authenticator.ValidateKey(key)
				if !ok {
					http.Error(w, "unauthorized: missing or invalid API key", http.StatusUnauthorized)
					return
				}
				if !cred.IsAdmin && !cred.CanAccessNamespace("*") {
					http.Error(w, "forbidden: admin privileges required", http.StatusForbidden)
					return
				}
			}

			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			targetNs := r.URL.Query().Get("namespace")
			var swept int
			var err error
			if nsGC, ok := gcRunner.(interface{ CollectGarbageForNamespace(string) (int, error) }); ok {
				swept, err = nsGC.CollectGarbageForNamespace(targetNs)
			} else {
				swept, err = gcRunner.CollectGarbage()
			}

			if err != nil {
				http.Error(w, fmt.Sprintf("GC execution failed: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "GC sweep complete: %d orphan chunks removed\n", swept)
		})
	}

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	mux.HandleFunc("/metrics", metrics.Handler())

	return mux
}


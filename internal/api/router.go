package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloudWeave/internal/auth"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/metrics"
	"cloudWeave/internal/s3"
)

type GCRunner interface {
	CollectGarbage() (int, error)
}

func jsonReader(v interface{}) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// verifyClusterSecret checks the X-Cluster-Secret header against the configured cluster secret.
// Returns true if the secret matches or if no cluster secret is configured (backward compatibility).
func verifyClusterSecret(r *http.Request, clusterSecret string) bool {
	if clusterSecret == "" {
		return true // No cluster secret configured — allow (backward compat)
	}
	provided := r.Header.Get("X-Cluster-Secret")
	return subtle.ConstantTimeCompare([]byte(provided), []byte(clusterSecret)) == 1
}

// wrapClusterAuth wraps an http.Handler with cluster secret verification.
func wrapClusterAuth(handler http.Handler, clusterSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !verifyClusterSecret(r, clusterSecret) {
			http.Error(w, "forbidden: invalid or missing cluster secret", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// wrapClusterAuthFunc wraps an http.HandlerFunc with cluster secret verification.
func wrapClusterAuthFunc(handler http.HandlerFunc, clusterSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !verifyClusterSecret(r, clusterSecret) {
			http.Error(w, "forbidden: invalid or missing cluster secret", http.StatusForbidden)
			return
		}
		handler(w, r)
	}
}

func NewRouter(apiHandler *APIHandler, transportHandler http.Handler, gcRunner GCRunner, authOpts ...*auth.Authenticator) http.Handler {
	return NewRouterWithClusterSecret(apiHandler, transportHandler, gcRunner, "", authOpts...)
}

func NewRouterWithClusterSecret(apiHandler *APIHandler, transportHandler http.Handler, gcRunner GCRunner, clusterSecret string, authOpts ...*auth.Authenticator) http.Handler {
	mux := http.NewServeMux()

	var authenticator *auth.Authenticator
	if len(authOpts) > 0 && authOpts[0] != nil {
		authenticator = authOpts[0]
	} else {
		authenticator = auth.NewDefaultAuthenticator()
	}

	if apiHandler != nil {
		apiHandler.SetAuthenticator(authenticator)
		apiHandler.SetClusterSecret(clusterSecret)
		mux.HandleFunc("/files/", apiHandler.HandleFiles)
		mux.HandleFunc("/internal/manifest", wrapClusterAuthFunc(apiHandler.HandleInternalManifest, clusterSecret))
		mux.HandleFunc("/internal/join", wrapClusterAuthFunc(apiHandler.HandleInternalJoin, clusterSecret))
		mux.HandleFunc("/internal/leave", wrapClusterAuthFunc(apiHandler.HandleInternalLeave, clusterSecret))
		mux.HandleFunc("/internal/revoke-key", wrapClusterAuthFunc(apiHandler.HandleInternalRevokeKey, clusterSecret))
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
				if !cred.IsAdmin {
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
				if !cred.IsAdmin {
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
							if clusterSecret != "" {
								req.Header.Set("X-Cluster-Secret", clusterSecret)
							}
							if resp, err := apiHandler.httpClient.Do(req); err == nil && resp != nil {
								resp.Body.Close()
							}
						}
					}
				}

				for _, peer := range pm.GetActiveNodes() {
					if req, err := http.NewRequest(http.MethodPost, nodeAddr+"/internal/join?node_addr="+peer, nil); err == nil {
						if clusterSecret != "" {
							req.Header.Set("X-Cluster-Secret", clusterSecret)
						}
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
				if !cred.IsAdmin {
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
				if !cred.IsAdmin {
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

				// Broadcast revocation to all cluster peers (#2)
				apiHandler.BroadcastKeyRevocation(targetHash)

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
		mux.Handle("/chunks/", wrapClusterAuth(transportHandler, clusterSecret))
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
				if !cred.IsAdmin {
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
				http.Error(w, "GC execution failed", http.StatusInternalServerError)
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

	var s3Handler http.Handler
	if apiHandler != nil && apiHandler.GetMetaStore() != nil && apiHandler.GetEngine() != nil {
		s3H := s3.NewS3Handler(apiHandler.GetMetaStore(), apiHandler.GetEngine(), apiHandler.GetChunkSize(), authenticator)
		if apiHandler.GetInFlightRegistry() != nil {
			s3H.SetInFlightRegistry(apiHandler.GetInFlightRegistry())
		}
		if gcRunner != nil {
			if gcWithMP, ok := gcRunner.(interface{ SetMultipartProvider(p gc.MultipartChunkProvider) }); ok {
				gcWithMP.SetMultipartProvider(s3H.GetMultipartStore())
			}
		}
		s3Handler = s3H
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Namespace")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := r.URL.Path
		if path == "/" || (!strings.HasPrefix(path, "/files/") &&
			!strings.HasPrefix(path, "/internal/") &&
			!strings.HasPrefix(path, "/dashboard") &&
			!strings.HasPrefix(path, "/cluster/") &&
			!strings.HasPrefix(path, "/admin/") &&
			!strings.HasPrefix(path, "/chunks/") &&
			path != "/health" &&
			path != "/metrics") {
			if s3Handler != nil {
				s3Handler.ServeHTTP(w, r)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}


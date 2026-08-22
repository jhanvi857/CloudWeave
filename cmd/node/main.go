package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloudWeave/internal/api"
	"cloudWeave/internal/auth"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/replication"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func getEnvOrDefault(envKey, fallback string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return fallback
}

func getEnvIntOrDefault(envKey string, fallback int) int {
	if val := os.Getenv(envKey); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func loadDotEnv(filenames ...string) {
	if len(filenames) == 0 {
		filenames = []string{".env", "../.env", "../../.env"}
	}
	for _, filename := range filenames {
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if key != "" && os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

func main() {
	loadDotEnv()
	port := flag.String("port", getEnvOrDefault("CLOUDWEAVE_PORT", "9000"), "port to listen on")
	dataDir := flag.String("data", getEnvOrDefault("CLOUDWEAVE_DATA", "./data"), "directory to store chunks in")
	peersFlag := flag.String("peers", getEnvOrDefault("CLOUDWEAVE_PEERS", ""), "comma-separated list of peer HTTP node addresses")
	walPath := flag.String("wal", getEnvOrDefault("CLOUDWEAVE_WAL", ""), "path to WAL file")
	nFlag := flag.Int("n", getEnvIntOrDefault("CLOUDWEAVE_N", 3), "replication factor N")
	wFlag := flag.Int("w", getEnvIntOrDefault("CLOUDWEAVE_W", 2), "write quorum W")
	rFlag := flag.Int("r", getEnvIntOrDefault("CLOUDWEAVE_R", 2), "read quorum R")
	apiKeysFlag := flag.String("api-keys", getEnvOrDefault("CLOUDWEAVE_API_KEYS", ""), "comma-separated list of key=ns1,ns2 or key=admin")
	storageMode := flag.String("storage-mode", getEnvOrDefault("CLOUDWEAVE_STORAGE_MODE", "replication"), "storage engine strategy")
	kFlag := flag.Int("k", getEnvIntOrDefault("CLOUDWEAVE_K", 4), "data shards K for erasure coding mode")
	mFlag := flag.Int("m", getEnvIntOrDefault("CLOUDWEAVE_M", 2), "parity shards M for erasure coding mode")
	tlsCert := flag.String("tls-cert", getEnvOrDefault("CLOUDWEAVE_TLS_CERT", ""), "path to TLS certificate file")
	tlsKey := flag.String("tls-key", getEnvOrDefault("CLOUDWEAVE_TLS_KEY", ""), "path to TLS private key file")
	tlsCA := flag.String("tls-ca", getEnvOrDefault("CLOUDWEAVE_TLS_CA", ""), "path to CA bundle file for node-to-node TLS verification")
	tlsClientAuth := flag.String("tls-client-auth", getEnvOrDefault("CLOUDWEAVE_TLS_CLIENT_AUTH", "verify-if-given"), "TLS client auth mode: 'require' (strict mTLS for all), 'verify-if-given' (mTLS for peers, API keys for clients), 'none'")
	tlsSkipVerify := flag.Bool("tls-insecure-skip-verify", os.Getenv("CLOUDWEAVE_TLS_SKIP_VERIFY") == "true", "skip TLS certificate verification for development")
	clusterSecretFlag := flag.String("cluster-secret", getEnvOrDefault("CLOUDWEAVE_CLUSTER_SECRET", ""), "shared secret for inter-node authentication")
	flag.Parse()

	if *dataDir == "" {
		log.Fatalf("data directory path cannot be empty")
	}

	// 0. TLS Setup
	var tlsConfig *tls.Config
	if *tlsCert != "" && *tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("failed to load TLS certificate/key pair: %v", err)
		}
		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: *tlsSkipVerify,
		}
		if *tlsCA != "" {
			caCert, err := os.ReadFile(*tlsCA)
			if err != nil {
				log.Fatalf("failed to read TLS CA bundle: %v", err)
			}
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caPool
			tlsConfig.ClientCAs = caPool

			switch strings.ToLower(*tlsClientAuth) {
			case "require", "strict":
				tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
				log.Printf("[TLS] Enforced strict mutual TLS (mTLS) certificate verification for all endpoints")
			case "verify-if-given", "optional":
				tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
				log.Printf("[TLS] Enabled mTLS for peer transport with API Key authentication for external clients")
			default:
				tlsConfig.ClientAuth = tls.NoClientCert
				log.Printf("[TLS] Server TLS enabled (client certificates disabled)")
			}
		}
		log.Printf("[TLS] Enabled end-to-end TLS encryption for client API and node-to-node mesh")
		if *tlsSkipVerify {
			log.Printf("[SECURITY WARNING] TLS certificate verification is DISABLED — do not use in production")
		}
	}

	var nodeHttpClient *http.Client
	if tlsConfig != nil {
		tlsTr := transport.DefaultTransport()
		tlsTr.TLSClientConfig = tlsConfig
		nodeHttpClient = &http.Client{Timeout: 10 * time.Second, Transport: tlsTr}
	} else {
		nodeHttpClient = transport.DefaultPooledClient()
	}

	scheme := "http"
	if tlsConfig != nil {
		scheme = "https"
	}
	localAddr := fmt.Sprintf("%s://localhost:%s", scheme, *port)
	log.Printf("[Main] Starting CloudWeave node on %s (Engine: %s, K=%d, M=%d)", localAddr, *storageMode, *kFlag, *mFlag)

	// 1. Storage
	diskStore, err := storage.NewDiskStore(*dataDir)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	// 2. Metadata Store & WAL
	if *walPath == "" {
		*walPath = filepath.Join(*dataDir, "metadata.wal")
	}

	metaStore := metadata.NewStore()

	// 0. Auth Setup
	var authenticator *auth.Authenticator
	if envKeys := os.Getenv("CLOUDWEAVE_API_KEYS"); envKeys != "" && *apiKeysFlag == "" {
		*apiKeysFlag = envKeys
	}

	if *apiKeysFlag != "" {
		var creds []auth.Credential
		keySpecs := strings.Split(*apiKeysFlag, ",")
		for _, spec := range keySpecs {
			spec = strings.TrimSpace(spec)
			if spec == "" {
				continue
			}
			parts := strings.SplitN(spec, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				rawNs := strings.TrimSpace(parts[1])
				isAdmin := rawNs == "admin" || rawNs == "*"
				var nsList []string
				if !isAdmin {
					nsList = strings.Split(rawNs, ";")
				} else {
					nsList = []string{"*"}
				}
				creds = append(creds, auth.Credential{
					KeyHash:    auth.HashKey(key),
					Namespaces: nsList,
					IsAdmin:    isAdmin,
				})
			}
		}
		authenticator = auth.NewAuthenticator(creds)
	} else {
		authenticator = auth.NewAuthenticator(nil)
	}

	if err := metadata.ReplayWAL(*walPath, metaStore, authenticator); err != nil {
		log.Printf("[WAL] Warning replaying WAL log: %v", err)
	} else {
		log.Printf("[WAL] Replayed metadata log from %s", *walPath)
	}

	wal, err := metadata.OpenWAL(*walPath)
	if err != nil {
		log.Fatalf("failed to open WAL: %v", err)
	}
	defer wal.Close()
	metaStore.SetWAL(wal)

	// If no API credentials exist (first boot without CLOUDWEAVE_API_KEYS), generate random admin key
	if len(authenticator.GetAllCredentials()) == 0 {
		rawAdminKey, err := auth.GenerateRandomKey()
		if err != nil {
			log.Fatalf("failed to generate initial admin API key: %v", err)
		}
		adminCred := auth.Credential{
			KeyHash:    auth.HashKey(rawAdminKey),
			Namespaces: []string{"*"},
			IsAdmin:    true,
		}
		authenticator.AddCredentialByHash(adminCred)
		_ = wal.WriteRecord(metadata.WALRecord{
			Op:         metadata.OpRecordKey,
			Credential: adminCred,
		})
		log.Printf("================================================================================")
		log.Printf("[SECURITY] First boot detected — Generated initial admin API Key:")
		log.Printf("  Key: %s", rawAdminKey)
		log.Printf("  Save this key securely! It is stored as a SHA-256 hash and cannot be recovered.")
		log.Printf("================================================================================")
	}

	// 3. Ring & Cluster Membership
	hashRing := ring.New()

	repairMgr := replication.NewRepairManager(metaStore, hashRing, *nFlag, localAddr, diskStore)
	repairMgr.SetHTTPClient(nodeHttpClient)
	repairMgr.SetClusterSecret(*clusterSecretFlag)
	workerPool := replication.NewRepairWorkerPool(repairMgr, 2)
	defer workerPool.Stop()

	membership := cluster.NewMembership(hashRing, func(deadNodeAddr string) {
		log.Printf("[Cluster] Event: Node %s died, submitting repair job...", deadNodeAddr)
		workerPool.SubmitDeadNodeJob(deadNodeAddr)
	})

	// Add local node and peers
	membership.AddNode(localAddr)

	if *peersFlag != "" {
		peers := strings.Split(*peersFlag, ",")
		for _, p := range peers {
			p = strings.TrimSpace(p)
			if p != "" && p != localAddr {
				membership.AddNode(p)
				log.Printf("[Cluster] Added peer node %s", p)
			}
		}
	}

	heartbeat := cluster.NewHeartbeatChecker(membership, 2*time.Second, 8*time.Second)
	heartbeat.SetHTTPClient(nodeHttpClient)
	heartbeat.SetClusterSecret(*clusterSecretFlag)
	heartbeat.Start()
	defer heartbeat.Stop()

	// Anti-Entropy Periodic Reconciler (Reconciles missing manifests across peers)
	reconciler := cluster.NewAntiEntropyReconciler(metaStore, membership, localAddr, 30*time.Second)
	reconciler.SetHTTPClient(nodeHttpClient)
	reconciler.SetClusterSecret(*clusterSecretFlag)
	reconciler.Start()
	defer reconciler.Stop()

	// 4. Quorum Coordinator & Garbage Collector
	coord := coordinator.NewCoordinator(hashRing, metaStore, localAddr, diskStore, *nFlag, *wFlag, *rFlag)
	coord.SetHTTPClient(nodeHttpClient)
	coord.SetClusterSecret(*clusterSecretFlag)
	gcEngine := gc.NewGarbageCollector(metaStore, diskStore)
	gcEngine.SetInFlightRegistry(diskStore.GetInFlightRegistry())

	// 5. API Handler & Transport Server
	apiHandler := api.NewAPIHandler(metaStore, coord, api.DefaultChunkSize)
	apiHandler.SetPeerManager(membership, localAddr)
	apiHandler.SetWAL(wal)
	apiHandler.SetHTTPClient(nodeHttpClient)
	apiHandler.SetClusterSecret(*clusterSecretFlag)
	apiHandler.SetDiskStore(diskStore)
	apiHandler.SetInFlightRegistry(diskStore.GetInFlightRegistry())

	transportServer := transport.NewServer(diskStore)
	router := api.NewRouterWithClusterSecret(apiHandler, transportServer.Handler(), gcEngine, *clusterSecretFlag, authenticator)


	serverAddr := ":" + *port
	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	if tlsConfig != nil {
		srv.TLSConfig = tlsConfig
		log.Printf("[Main] Node listening on %s (HTTPS/TLS), storing data in %s", serverAddr, *dataDir)
		if err := srv.ListenAndServeTLS(*tlsCert, *tlsKey); err != nil {
			log.Fatalf("server exit: %v", err)
			os.Exit(1)
		}
	} else {
		log.Printf("[Main] Node listening on %s (HTTP), storing data in %s", serverAddr, *dataDir)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server exit: %v", err)
			os.Exit(1)
		}
	}
}


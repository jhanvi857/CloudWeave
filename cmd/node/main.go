package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloudWeave/internal/api"
	"cloudWeave/internal/cluster"
	"cloudWeave/internal/consensus"
	"cloudWeave/internal/coordinator"
	"cloudWeave/internal/gc"
	"cloudWeave/internal/metadata"
	"cloudWeave/internal/replication"
	"cloudWeave/internal/ring"
	"cloudWeave/internal/storage"
	"cloudWeave/internal/transport"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store chunks in")
	peersFlag := flag.String("peers", "", "comma-separated list of peer HTTP node addresses (e.g. http://localhost:8081,http://localhost:8082)")
	walPath := flag.String("wal", "", "path to WAL file (defaults to <dataDir>/metadata.wal)")
	nFlag := flag.Int("n", 3, "replication factor N")
	wFlag := flag.Int("w", 2, "write quorum W")
	rFlag := flag.Int("r", 2, "read quorum R")
	storageMode := flag.String("storage-mode", "replication", "storage engine strategy: 'replication' or 'erasure'")
	kFlag := flag.Int("k", 4, "data shards K for erasure coding mode")
	mFlag := flag.Int("m", 2, "parity shards M for erasure coding mode")
	flag.Parse()

	if *dataDir == "" {
		log.Fatalf("data directory path cannot be empty")
	}

	localAddr := fmt.Sprintf("http://localhost:%s", *port)
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
	if err := metadata.ReplayWAL(*walPath, metaStore); err != nil {
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

	// Raft Consensus Engine (Replicated Log State Machine)
	var peerAddrs []string
	if *peersFlag != "" {
		peerAddrs = strings.Split(*peersFlag, ",")
	}
	raftEngine := consensus.NewRaftNode(localAddr, peerAddrs, metaStore)
	raftEngine.Start()
	defer raftEngine.Stop()
	raftEngine.ForceLeader()

	// 3. Ring & Cluster Membership
	hashRing := ring.New()

	repairMgr := replication.NewRepairManager(metaStore, hashRing, *nFlag, localAddr, diskStore)
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

	heartbeat := cluster.NewHeartbeatChecker(membership, 2*time.Second, 5*time.Second)
	heartbeat.Start()
	defer heartbeat.Stop()

	// 4. Quorum Coordinator & Garbage Collector
	coord := coordinator.NewCoordinator(hashRing, metaStore, localAddr, diskStore, *nFlag, *wFlag, *rFlag)
	gcEngine := gc.NewGarbageCollector(metaStore, diskStore)

	// 5. API Handler & Transport Server
	apiHandler := api.NewAPIHandler(metaStore, coord, api.DefaultChunkSize)
	transportServer := transport.NewServer(diskStore)

	router := api.NewRouter(apiHandler, transportServer.Handler(), gcEngine)

	serverAddr := ":" + *port
	log.Printf("[Main] Node listening on %s, storing data in %s", serverAddr, *dataDir)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatalf("server exit: %v", err)
		os.Exit(1)
	}
}


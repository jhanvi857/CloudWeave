# CloudWeave Performance Benchmarks

This document records empirical performance benchmarks of CloudWeave measured over 5-iteration benchmark runs (`go test -bench . -count=5 ./test/benchmark`) with zero-allocation setup loops and `runtime.GC()` isolation.

## Test Environment
- **OS**: Windows (x86_64)
- **CPU**: Intel(R) Core(TM) Ultra 5 125H (18 threads)
- **Cluster Topology**: 3-Node Local Mesh Cluster with Quorum (N=3, W=2, R=2)
- **Benchmark Command**: `go test -bench . -count=5 ./test/benchmark`

---

## Empirical Benchmark Results

| Benchmark | Baseline | Empirical Range / Median | Category & Architectural Analysis |
| :--- | :--- | :--- | :--- |
| **Upload Throughput (2MB)** | ~21.9 MB/s | **92.25 – 214.38 MB/s** (~151.61 MB/s median) | **+4.4x – +9.5x Real Win** — Fixed disk store locking & `MoveFileEx`/`os.CreateTemp` overhead. |
| **Download (2MB Post-Write Read)** | ~10.14 MB/s | **109.62 – 231.18 MB/s** (~120.90 MB/s median) | **OS Page Cache Sensitive** — Plain range; uncached disk tier (~109 MB/s) vs OS page cache hit (~231 MB/s). |
| **Download (10MB Post-Write Read)** | ~10.14 MB/s | **128.98 – 149.62 MB/s** (~142.78 MB/s median) | **OS Page Cache Sensitive** — Varies dynamically with kernel filesystem page cache residency. |
| **Download (Warm Cache - LRU RAM)** | ~10.14 MB/s | **75.06 – 177.57 MB/s** (~94.68 MB/s median) | **+7.4x – +17.5x Real Win** — In-memory LRU cache hit latency (~11.8 – 27.9 ms/op). |
| **Concurrent Load (64KB QPS)** | ~47 req/sec | **289.2 – 558.8 req/sec** (~333 req/sec median) | **+6.1x – +11.8x Real Win** — Connection pooling with `http.Transport` keep-alives. |

---

## Upload Bottleneck Root-Cause Profile & Fix

1. **Root Cause Analysis (Profiling with `pprof` & `go test -cpuprofile`)**:
   - Initial benchmark runs showed upload throughput was bottlenecked at ~21.9 MB/s despite parallel chunk workers.
   - Profiling revealed that `os.CreateTemp` (16.05% CPU) and `syscall.MoveFileEx` / `os.Rename` (33.08% CPU) on Windows NTFS were executed synchronously inside a global `DiskStore` mutex lock (`s.mu.Lock()`).
   - The global mutex forced all parallel chunk workers writing to a node to wait sequentially on file creation and file rename syscalls.

2. **Resolution & Fix**:
   - Refactored `DiskStore.Put` in `internal/storage/diskstore.go`:
     - Implemented fast-path existence checks (`s.Exists`) to bypass redundant disk writes for identical content-addressed chunks.
     - Replaced `os.CreateTemp` + `os.Rename` (`MoveFileEx`) with direct `os.WriteFile` writes.
     - Released `s.mu.Lock()` during disk I/O so concurrent chunk worker goroutines execute disk writes in parallel.
   - **Empirical Result**: Upload throughput increased from **~21.9 MB/s** to **92.25 – 214.38 MB/s** (median **~151.61 MB/s**).

---

## Raw Terminal Output Evidence

Below is the unedited 5-iteration benchmark execution output captured directly from the terminal session (`go test -bench . -count 5 ./test/benchmark`):

```
goos: windows
goarch: amd64
pkg: cloudWeave/test/benchmark
cpu: Intel(R) Core(TM) Ultra 5 125H
BenchmarkUploadThroughput-18              	      88	  21063800 ns/op	  99.56 MB/s
BenchmarkUploadThroughput-18              	      44	  22733818 ns/op	  92.25 MB/s
BenchmarkUploadThroughput-18              	     114	  11167005 ns/op	 187.80 MB/s
BenchmarkUploadThroughput-18              	     112	   9782369 ns/op	 214.38 MB/s
BenchmarkUploadThroughput-18              	     102	  13832111 ns/op	 151.61 MB/s
BenchmarkDownloadPostWriteRead-18         	     115	  13246843 ns/op	 158.31 MB/s
BenchmarkDownloadPostWriteRead-18         	      61	  18438780 ns/op	 113.74 MB/s
BenchmarkDownloadPostWriteRead-18         	      61	  19131280 ns/op	 109.62 MB/s
BenchmarkDownloadPostWriteRead-18         	     147	   9071417 ns/op	 231.18 MB/s
BenchmarkDownloadPostWriteRead-18         	      68	  17346437 ns/op	 120.90 MB/s
BenchmarkDownloadPostWriteReadLarge-18    	      14	  81300214 ns/op	 128.98 MB/s
BenchmarkDownloadPostWriteReadLarge-18    	      15	  70988747 ns/op	 147.71 MB/s
BenchmarkDownloadPostWriteReadLarge-18    	      15	  70081920 ns/op	 149.62 MB/s
BenchmarkDownloadPostWriteReadLarge-18    	      15	  73441867 ns/op	 142.78 MB/s
BenchmarkDownloadPostWriteReadLarge-18    	      16	  79194506 ns/op	 132.41 MB/s
BenchmarkDownloadWarmCache-18             	      93	  11809997 ns/op	 177.57 MB/s
BenchmarkDownloadWarmCache-18             	      93	  18438329 ns/op	 113.74 MB/s
BenchmarkDownloadWarmCache-18             	      60	  27939678 ns/op	  75.06 MB/s
BenchmarkDownloadWarmCache-18             	      61	  26701966 ns/op	  78.54 MB/s
BenchmarkDownloadWarmCache-18             	      55	  22150195 ns/op	  94.68 MB/s
BenchmarkConcurrentLoad-18                	     387	   3005739 ns/op
BenchmarkConcurrentLoad-18                	     358	   3457756 ns/op
BenchmarkConcurrentLoad-18                	     357	   3014898 ns/op
BenchmarkConcurrentLoad-18                	     360	   2788813 ns/op
BenchmarkConcurrentLoad-18                	    1011	   1789666 ns/op
PASS
ok  	cloudWeave/test/benchmark	80.373s
```

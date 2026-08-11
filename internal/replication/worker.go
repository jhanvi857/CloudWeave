package replication

import (
	"log"
	"sync"
)

type RepairWorkerPool struct {
	repairMgr *RepairManager
	jobQueue  chan string
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

func NewRepairWorkerPool(repairMgr *RepairManager, workers int) *RepairWorkerPool {
	if workers <= 0 {
		workers = 2
	}

	pool := &RepairWorkerPool{
		repairMgr: repairMgr,
		jobQueue:  make(chan string, 100),
		stopChan:  make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.workerLoop(i)
	}

	return pool
}

func (p *RepairWorkerPool) SubmitDeadNodeJob(deadNodeAddr string) {
	select {
	case p.jobQueue <- deadNodeAddr:
	default:
		log.Printf("[RepairWorkerPool] Job queue full, skipping duplicate dead node job for %s", deadNodeAddr)
	}
}

func (p *RepairWorkerPool) workerLoop(id int) {
	defer p.wg.Done()

	for {
		select {
		case deadAddr := <-p.jobQueue:
			log.Printf("[RepairWorker %d] Starting repair process for dead node %s", id, deadAddr)
			count, err := p.repairMgr.RepairDeadNode(deadAddr)
			if err != nil {
				log.Printf("[RepairWorker %d] Repair for %s failed: %v", id, deadAddr, err)
			} else {
				log.Printf("[RepairWorker %d] Repair for %s completed (%d chunks re-replicated)", id, deadAddr, count)
			}
		case <-p.stopChan:
			return
		}
	}
}

func (p *RepairWorkerPool) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

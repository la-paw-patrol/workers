package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type WorkerService struct {
	cancelToken context.CancelFunc
	ctx         context.Context
	wg          *sync.WaitGroup
	log         *log.Logger
	sigChan     chan os.Signal
	service     *Service
}

type Service interface {
	doWork()
}

func NewWorkerService(service Service) *WorkerService {
	defaultLogger := log.New(log.Default().Writer(), "[INFO] ", log.Flags())
	return NewWorkerServiceWithLogger(defaultLogger, service)
}

func NewWorkerServiceWithLogger(logger *log.Logger, service Service) *WorkerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerService{
		cancelToken: cancel,
		ctx:         ctx,
		wg:          &sync.WaitGroup{},
		log:         logger,
		sigChan:     make(chan os.Signal),
		service:     &service,
	}
}

func (ws *WorkerService) Start() {
	signal.Notify(ws.sigChan, syscall.SIGINT, syscall.SIGTERM)

	ws.wg.Add(1)

	go func() {
		defer ws.wg.Done()
		ws.run(*ws.service)
	}()

	ws.log.Print("Worker service started.")

	ws.wg.Wait()
}

func (ws *WorkerService) run(service Service) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case sig := <-ws.sigChan:
			ws.log.Printf("Received signal: %s", sig)
			ws.Stop()
			return
		case <-ticker.C:
			service.doWork()
		}
	}
}

func (ws *WorkerService) Stop() {
	ws.log.Printf("Worker server terminating gracefully...")
	ws.cancelToken()
}

type ServiceImpl struct{}

func (s *ServiceImpl) doWork() {
	log.Printf("WORKING...")
}

func (ws *WorkerService) doWork() {
	ws.log.Printf("WORKING!")
}

func main() {
	service := Service(&ServiceImpl{})

	ws := NewWorkerService(service)

	ws.Start()
}

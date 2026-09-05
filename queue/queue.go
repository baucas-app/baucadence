package queue

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RequestResult holds the result of a queued request.
type RequestResult struct {
	Body []byte
	Err  error
}

// RequestFunc is a function that performs an HTTP request using the provided client,
// and sends its result to the given result channel.
type RequestFunc func(client *http.Client, done chan<- RequestResult)

type RequestQueue struct {
	client  *http.Client
	limiter *rate.Limiter
	queue   chan func(*http.Client) // now this is a wrapped closure
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewRequestQueue creates a new rate-limited request queue.
// `rps` = requests per second, `burst` = burst capacity
func NewRequestQueue(rps int, burst int) *RequestQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &RequestQueue{
		client:  &http.Client{Timeout: 10 * time.Second},
		limiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(rps)), burst),
		queue:   make(chan func(*http.Client), 100), // accepts wrapped closures
		ctx:     ctx,
		cancel:  cancel,
	}
	q.start()
	return q
}

// Enqueue adds a new request to the queue and returns a result channel.
func (q *RequestQueue) Enqueue(job RequestFunc) <-chan RequestResult {
	resultChan := make(chan RequestResult, 1)
	q.queue <- func(client *http.Client) {
		job(client, resultChan)
	}
	return resultChan
}

// start begins the worker loop.
func (q *RequestQueue) start() {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-q.ctx.Done():
				return
			case job := <-q.queue:
				if err := q.limiter.Wait(q.ctx); err != nil {
					log.Println("[queue] limiter wait failed:", err)
					continue
				}
				go job(q.client)
			}
		}
	}()
}

// Shutdown stops the queue and waits for the worker to finish.
func (q *RequestQueue) Shutdown() {
	q.cancel()
	q.wg.Wait()
	close(q.queue)
}

package workerpool

import (
	"context"
	"time"
)

type Result struct {
	Err   error
	Value any
}

type Job struct {
	ctx     context.Context
	timeout time.Duration
	fn      func(ctx context.Context) (any, error)
	resCh   chan *Result
}

type Pool struct {
	jobs  chan *Job
	limit int
}

func NewPool(limit, queueSize int) *Pool {
	if queueSize == 0 {
		queueSize = limit * 5
	}

	return &Pool{
		jobs:  make(chan *Job, queueSize),
		limit: limit,
	}
}

func (p *Pool) Start() {
	for range p.limit {
		go p.worker()
	}
}

func (p *Pool) Submit(
	ctx context.Context,
	timeout time.Duration,
	fn func(ctx context.Context) (any, error),
	expectResult bool,
) chan *Result {
	var resCh chan *Result
	if expectResult {
		resCh = make(chan *Result, 1)
	}

	job := &Job{
		ctx:     ctx,
		timeout: timeout,
		fn:      fn,
		resCh:   resCh,
	}

	p.jobs <- job
	return resCh
}

func (p *Pool) worker() {
	for job := range p.jobs {
		func() {
			ctx, cancel := context.WithTimeout(job.ctx, job.timeout)
			defer cancel()

			res, err := job.fn(ctx)

			if job.resCh != nil {
				r := &Result{
					Value: res,
					Err:   err,
				}

				job.resCh <- r
			}
		}()
	}
}

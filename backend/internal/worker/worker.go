package worker

import (
    "context"
    "sync"
)

// Job represents a unit of work.
type Job struct {
    ID string
    Execute func(context.Context) error
}

// Pool executes jobs with bounded concurrency.
type Pool struct {
    ch chan Job
    wg sync.WaitGroup
}

func New(size int) *Pool {
    return &Pool{ch: make(chan Job), wg: sync.WaitGroup{}}
}

func (p *Pool) Run(ctx context.Context, size int) {
    for i := 0; i < size; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for {
                select {
                case <-ctx.Done():
                    return
                case job := <-p.ch:
                    _ = job.Execute(ctx)
                }
            }
        }()
    }
}

func (p *Pool) Submit(j Job) { p.ch <- j }
func (p *Pool) Wait()        { p.wg.Wait() }

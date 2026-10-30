package stockfish

import "context"

type Pool struct {
	path      string
	processes chan *Process
}

func NewPool(path string, size int) *Pool {
	if size < 1 {
		size = 1
	}

	pool := &Pool{
		path:      path,
		processes: make(chan *Process, size),
	}

	for i := 0; i < size; i++ {
		pool.processes <- nil
	}

	return pool
}

func (p *Pool) GetProcess(ctx context.Context) (*Process, error) {
	select {
	case process := <-p.processes:
		if process != nil {
			return process, nil
		}

		newProcess, err := StartProcess(ctx, p.path)
		if err != nil {
			p.processes <- nil
			return nil, err
		}
		return newProcess, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) ReturnProcess(process *Process) {
	p.processes <- process
}

func (p *Pool) ReplaceBrokenProcess(process *Process) {
	if process != nil {
		process.Close()
	}
	p.processes <- nil
}

func (p *Pool) Close() {
	for {
		select {
		case process := <-p.processes:
			if process != nil {
				process.Close()
			}
		default:
			return
		}
	}
}

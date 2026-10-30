package stockfish

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Process struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

func StartProcess(ctx context.Context, path string) (*Process, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.Command(path) //nolint:gosec // path comes from server config, not from users

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stockfish stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stockfish stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start stockfish: %w", err)
	}

	process := &Process{
		cmd:     cmd,
		stdin:   stdin,
		scanner: bufio.NewScanner(stdout),
	}

	if err := process.Send("uci"); err != nil {
		process.Close()
		return nil, err
	}
	if _, err := process.WaitForWithContext(startupCtx, "uciok"); err != nil {
		process.Close()
		return nil, err
	}

	return process, nil
}

func (p *Process) Send(command string) error {
	if _, err := io.WriteString(p.stdin, command+"\n"); err != nil {
		return fmt.Errorf("send %q to stockfish: %w", command, err)
	}
	return nil
}

func (p *Process) WaitFor(prefix string, ctx ...context.Context) (string, error) {
	if len(ctx) > 0 && ctx[0] != nil {
		return p.WaitForWithContext(ctx[0], prefix)
	}
	return p.waitForLine(prefix)
}

func (p *Process) WaitForWithContext(ctx context.Context, prefix string) (string, error) {
	type waitResult struct {
		line string
		err  error
	}

	resultCh := make(chan waitResult, 1)
	go func() {
		line, err := p.waitForLine(prefix)
		resultCh <- waitResult{line: line, err: err}
	}()

	select {
	case res := <-resultCh:
		return res.line, res.err
	case <-ctx.Done():
		p.Kill()
		select {
		case <-resultCh:
		case <-time.After(500 * time.Millisecond):
		}
		return "", ctx.Err()
	}
}

func (p *Process) waitForLine(prefix string) (string, error) {
	for p.scanner.Scan() {
		line := p.scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return line, nil
		}
	}

	if err := p.scanner.Err(); err != nil {
		return "", fmt.Errorf("read stockfish output: %w", err)
	}

	return "", fmt.Errorf("stockfish closed before %q", prefix)
}

func (p *Process) ReadLine() (string, error) {
	if p.scanner.Scan() {
		return p.scanner.Text(), nil
	}

	if err := p.scanner.Err(); err != nil {
		return "", fmt.Errorf("read stockfish output: %w", err)
	}

	return "", errors.New("stockfish closed")
}

func (p *Process) Close() {
	if p == nil {
		return
	}
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
	}
}

func (p *Process) Kill() {
	if p == nil {
		return
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

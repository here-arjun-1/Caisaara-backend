package stockfish

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Process struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

func StartProcess(path string) (*Process, error) {
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
	if _, err := process.WaitFor("uciok"); err != nil {
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

func (p *Process) WaitFor(prefix string) (string, error) {
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
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.cmd.Wait()
}

func (p *Process) Kill() {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

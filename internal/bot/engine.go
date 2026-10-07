package bot

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Engine struct {
	Path string
}

func NewEngine(path string) *Engine {
	return &Engine{
		Path: path,
	}
}

func (e *Engine) BestMove(
	ctx context.Context,
	fen string,
	settings EngineSettings,
) (string, error) {
	cmd := exec.CommandContext(ctx, e.Path) //nolint:gosec // path comes from server config, not from users

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("open stockfish stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("open stockfish stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start stockfish: %w", err)
	}

	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)

	if err := send(stdin, "uci"); err != nil {
		return "", err
	}
	if _, err := waitFor(scanner, "uciok"); err != nil {
		return "", err
	}

	var commands []string
	if settings.LimitStrength {
		commands = append(commands,
			"setoption name UCI_LimitStrength value true",
			fmt.Sprintf("setoption name UCI_Elo value %d", settings.Elo),
		)
	}
	commands = append(commands,
		fmt.Sprintf("setoption name Skill Level value %d", settings.SkillLevel),
		"isready",
	)

	for _, command := range commands {
		if err := send(stdin, command); err != nil {
			return "", err
		}
	}
	if _, err := waitFor(scanner, "readyok"); err != nil {
		return "", err
	}

	goCommand := fmt.Sprintf("go movetime %d", settings.MoveTimeMs)
	if settings.Depth > 0 {
		goCommand = fmt.Sprintf("go depth %d movetime %d", settings.Depth, settings.MoveTimeMs)
	}

	if err := send(stdin, "position fen "+fen); err != nil {
		return "", err
	}
	if err := send(stdin, goCommand); err != nil {
		return "", err
	}

	line, err := waitFor(scanner, "bestmove")
	if err != nil {
		return "", err
	}

	fields := strings.Fields(line)
	if len(fields) < 2 || fields[1] == "(none)" {
		return "", ErrNoBestMove
	}

	return fields[1], nil
}

func send(w io.Writer, command string) error {
	if _, err := io.WriteString(w, command+"\n"); err != nil {
		return fmt.Errorf("send %q to stockfish: %w", command, err)
	}
	return nil
}

func waitFor(scanner *bufio.Scanner, prefix string) (string, error) {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return line, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read stockfish output: %w", err)
	}

	return "", fmt.Errorf("stockfish closed before %q", prefix)
}

package analysis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrEngineNotFound = errors.New("stockfish engine binary not found or failed to start")
	ErrNoBestMove     = errors.New("no valid best move returned by engine")
	ErrTimeout        = errors.New("engine search timed out")
)

type AnalysisEngine struct {
	Path string
	mu   sync.Mutex
}

type PositionOptions struct {
	FEN   string   `json:"fen,omitempty"`
	Moves []string `json:"moves,omitempty"`
}

func (p PositionOptions) BuildCommand() string {
	if p.FEN != "" && len(p.Moves) > 0 {
		return fmt.Sprintf("position fen %s moves %s", p.FEN, strings.Join(p.Moves, " "))
	}
	if p.FEN != "" {
		return fmt.Sprintf("position fen %s", p.FEN)
	}
	if len(p.Moves) > 0 {
		return fmt.Sprintf("position startpos moves %s", strings.Join(p.Moves, " "))
	}
	return "position startpos"
}

type AnalysisOptions struct {
	Depth         int           `json:"depth,omitempty"`
	MoveTimeMs    int64         `json:"move_time_ms,omitempty"`
	MultiPV       int           `json:"multipv,omitempty"`
	Threads       int           `json:"threads,omitempty"`
	HashMB        int           `json:"hash_mb,omitempty"`
	SkillLevel    *int          `json:"skill_level,omitempty"`
	Elo           *int          `json:"elo,omitempty"`
	LimitStrength bool          `json:"limit_strength,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty"`
}

type AnalysisResult struct {
	BestMove             string   `json:"best_move"`
	PonderMove           string   `json:"ponder_move,omitempty"`
	EvaluationCentipawns *int     `json:"evaluation_centipawns,omitempty"`
	MateIn               *int     `json:"mate_in,omitempty"`
	Depth                int      `json:"depth"`
	Nodes                int64    `json:"nodes"`
	PV                   []string `json:"pv,omitempty"`
}

func NewAnalysisEngine(path string) *AnalysisEngine {
	if path == "" {
		path = "stockfish"
	}
	return &AnalysisEngine{
		Path: path,
	}
}

func (e *AnalysisEngine) Analyze(ctx context.Context, pos PositionOptions, opts AnalysisOptions) (*AnalysisResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	} else if opts.MoveTimeMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.MoveTimeMs+2000)*time.Millisecond)
	} else {
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, e.Path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEngineNotFound, err)
	}

	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)

	if err := sendCommand(stdin, "uci"); err != nil {
		return nil, err
	}
	if _, err := waitForPrefix(scanner, "uciok"); err != nil {
		return nil, fmt.Errorf("uci handshake failed: %w", err)
	}

	var options []string
	if opts.Threads > 0 {
		options = append(options, fmt.Sprintf("setoption name Threads value %d", opts.Threads))
	}
	if opts.HashMB > 0 {
		options = append(options, fmt.Sprintf("setoption name Hash value %d", opts.HashMB))
	}
	if opts.MultiPV > 0 {
		options = append(options, fmt.Sprintf("setoption name MultiPV value %d", opts.MultiPV))
	}
	if opts.LimitStrength && opts.Elo != nil {
		options = append(options,
			"setoption name UCI_LimitStrength value true",
			fmt.Sprintf("setoption name UCI_Elo value %d", *opts.Elo),
		)
	}
	if opts.SkillLevel != nil {
		options = append(options, fmt.Sprintf("setoption name Skill Level value %d", *opts.SkillLevel))
	}

	options = append(options, "isready")

	for _, opt := range options {
		if err := sendCommand(stdin, opt); err != nil {
			return nil, err
		}
	}
	if _, err := waitForPrefix(scanner, "readyok"); err != nil {
		return nil, fmt.Errorf("readyok failed: %w", err)
	}

	if err := sendCommand(stdin, pos.BuildCommand()); err != nil {
		return nil, fmt.Errorf("send position command: %w", err)
	}

	goCmd := "go"
	if opts.Depth > 0 && opts.MoveTimeMs > 0 {
		goCmd = fmt.Sprintf("go depth %d movetime %d", opts.Depth, opts.MoveTimeMs)
	} else if opts.Depth > 0 {
		goCmd = fmt.Sprintf("go depth %d", opts.Depth)
	} else if opts.MoveTimeMs > 0 {
		goCmd = fmt.Sprintf("go movetime %d", opts.MoveTimeMs)
	} else {
		goCmd = "go depth 15"
	}

	if err := sendCommand(stdin, goCmd); err != nil {
		return nil, fmt.Errorf("send go command: %w", err)
	}

	result := &AnalysisResult{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "info ") {
			parseInfoLine(line, result)
		} else if strings.HasPrefix(line, "bestmove ") {
			parseBestMoveLine(line, result)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan output error: %w", err)
	}

	if ctx.Err() == context.DeadlineExceeded {
		return nil, ErrTimeout
	}

	if result.BestMove == "" || result.BestMove == "(none)" {
		return nil, ErrNoBestMove
	}

	_ = sendCommand(stdin, "quit")

	return result, nil
}

func parseInfoLine(line string, res *AnalysisResult) {
	fields := strings.Fields(line)
	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "depth":
			if i+1 < len(fields) {
				if d, err := strconv.Atoi(fields[i+1]); err == nil {
					res.Depth = d
				}
			}
		case "nodes":
			if i+1 < len(fields) {
				if n, err := strconv.ParseInt(fields[i+1], 10, 64); err == nil {
					res.Nodes = n
				}
			}
		case "score":
			if i+2 < len(fields) {
				scoreType := fields[i+1]
				scoreVal, err := strconv.Atoi(fields[i+2])
				if err == nil {
					switch scoreType {
					case "cp":
						res.EvaluationCentipawns = &scoreVal
						res.MateIn = nil
					case "mate":
						res.MateIn = &scoreVal
						res.EvaluationCentipawns = nil
					}
				}
			}
		case "pv":
			if i+1 < len(fields) {
				res.PV = fields[i+1:]
			}
			return
		}
	}
}

func parseBestMoveLine(line string, res *AnalysisResult) {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		res.BestMove = fields[1]
	}
	if len(fields) >= 4 && fields[2] == "ponder" {
		res.PonderMove = fields[3]
	}
}

func sendCommand(w io.Writer, cmd string) error {
	if _, err := io.WriteString(w, cmd+"\n"); err != nil {
		return fmt.Errorf("send command %q: %w", cmd, err)
	}
	return nil
}

func waitForPrefix(scanner *bufio.Scanner, prefix string) (string, error) {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("EOF reached before %q", prefix)
}

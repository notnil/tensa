package tegra

import (
	"bufio"
	"context"
	"log/slog"
	"os/exec"
	"syscall"

	"github.com/notnil/tensa/pkg/metrics"
)

// StatsSub implements pubsubx.Sub for metrics.Metric[Stats].
// It runs the 'tegrastats' command and sends parsed stats to a channel.
type StatsSub struct {
	logger *slog.Logger
	parser Parser
}

// NewStatsSub creates a new StatsSub.
func NewStatsSub(logger *slog.Logger, parser Parser) *StatsSub {
	return &StatsSub{
		logger: logger.With(slog.String("system", "tegra")),
		parser: parser,
	}
}

// Subscribe runs the tegrastats command, parses its output line by line,
// and sends the parsed TegraStats metrics to the provided channel.
// It stops when the context is cancelled or if an error occurs.
// It implements the pubsubx.Sub[metrics.Metric[Stats]] interface.
func (s *StatsSub) Subscribe(ctx context.Context, ch chan<- metrics.Metric[Stats]) error {
	cmd := exec.CommandContext(ctx, "tegrastats")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.logger.Error("Failed to get stdout pipe for tegrastats", slog.Any("error", err))
		return err
	}

	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start tegrastats", slog.Any("error", err))
		return err
	}
	s.logger.Info("tegrastats process started")

	scanner := bufio.NewScanner(stdout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				s.logger.Error("Scanner error while reading tegrastats output", slog.Any("error", err))
				return err
			}
			return nil // EOF
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		stats, err := s.parser.Parse(line)
		if err != nil {
			s.logger.Warn("Failed to parse tegrastats line", slog.String("line", line), slog.Any("error", err))
			continue
		}

		metric := metrics.Metric[Stats]{
			Timestamp: stats.Timestamp,
			Value:     stats,
		}

		ch <- metric
	}
}

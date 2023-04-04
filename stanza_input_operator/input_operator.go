package kubearmor_receiver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"go.uber.org/zap"
	"io"
	"os/exec"
	"sync"
	"time"
)

const operatorType = "kubearmor_input"

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new input config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new input config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		InputConfig: helper.NewInputConfig(operatorID, operatorType),
		Endpoint:    ":32767",
		LogFilter:   "all",
	}
}

// Config is the configuration of a kubearmor input operator
type Config struct {
	helper.InputConfig `mapstructure:",squash"`

	Endpoint  string `mapstructure:"endpoint,omitempty"`
	LogFilter string `mapstructure:"logfilter,omitempty"`
}

// Build will build a journald input operator from the supplied configuration
func (c Config) Build(logger *zap.SugaredLogger) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(logger)
	if err != nil {
		return nil, err
	}

	var args []string

	// Set endpoint option
	args = append(args, fmt.Sprintf("--gRPC=%s", c.Endpoint))

	// Set Log filter option
	args = append(args, fmt.Sprintf("--logFilter=%s", c.LogFilter))

	// Set to json format
	args = append(args, "--json")

	return &Input{
		InputOperator: inputOperator,
		newCmd: func(ctx context.Context) cmd {
			return exec.CommandContext(ctx, "logClient", args...) // #nosec - ...
			// logClient is a an executable that is required for this operator
			//    to function
		},
		json: jsoniter.ConfigFastest,
	}, nil
}

// Input is an operator that process logs using journald
type Input struct {
	helper.InputOperator

	newCmd func(ctx context.Context) cmd

	json   jsoniter.API
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type cmd interface {
	StdoutPipe() (io.ReadCloser, error)
	Start() error
}

// Start will start generating log entries.
func (operator *Input) Start(_ operator.Persister) error {
	ctx, cancel := context.WithCancel(context.Background())
	operator.cancel = cancel

	logClient := operator.newCmd(ctx)
	stdout, err := logClient.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get logClient stdout: %w", err)
	}
	err = logClient.Start()
	if err != nil {
		return fmt.Errorf("start logClient: %w", err)
	}

	// Start the reader goroutine
	operator.wg.Add(1)
	go func() {
		defer operator.wg.Done()

		stdoutBuf := bufio.NewReader(stdout)

		for {
			line, err := stdoutBuf.ReadBytes('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					operator.Errorw("Received error reading from logClient stdout", zap.Error(err))
				}
				return
			}

			entry, err := operator.parseLogEntry(line)
			if err != nil {
				operator.Warnw("Failed to parse journal entry", zap.Error(err))
				continue
			}
			operator.Write(ctx, entry)
		}
	}()

	return nil
}

func (operator *Input) parseLogEntry(line []byte) (*entry.Entry, error) {
	if !operator.json.Valid(line) {
		return nil, errors.New("skipping line: invalid json")
	}
	var body map[string]interface{}
	err := operator.json.Unmarshal(line, &body)

	if err != nil {
		return nil, err
	}

	timestamp, ok := body["Timestamp"]

	if !ok {
		return nil, errors.New("log body missing timestamp field")
	}

	timestampFloat, ok := timestamp.(float64)
	if !ok {
		return nil, errors.New("log body field for timestamp is not of string type")
	}

	delete(body, "Timestamp")

	entry, err := operator.NewEntry(body)
	if err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}

	entry.Timestamp = time.Unix(0, int64(timestampFloat*1000)) // in microseconds

	return entry, nil
}

// Stop will stop generating logs.
func (operator *Input) Stop() error {
	operator.cancel()
	operator.wg.Wait()
	return nil
}

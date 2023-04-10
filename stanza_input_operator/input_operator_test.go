package kubearmor_receiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
)

func TestInputKubearmor(t *testing.T) {
	cfg := NewConfig()
	cfg.OutputIDs = []string{"output"}

	op, err := cfg.Build(testutil.Logger(t))
	require.NoError(t, err)

	mockOutput := testutil.NewMockOperator("output")
	received := make(chan *entry.Entry)
	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		received <- args.Get(1).(*entry.Entry)
	}).Return(nil)

	err = op.SetOutputs([]operator.Operator{mockOutput})
	require.NoError(t, err)

	err = op.Start(testutil.NewMockPersister("test"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, op.Stop())
	}()

	expected := map[string]interface{}{
		"Data":        "syscall=SYS_OPENAT fd=-100 flags=O_RDONLY|O_CLOEXEC",
		"HostName":    "host-name",
		"HostPID":     float64(26846),
		"Operation":   "File",
		"PID":         float64(26846),
		"PPID":        float64(14270),
		"Resource":    "/home/user/go/pkg/mod/cache/download/sumdb/sum.golang.org/tile/8/1/244",
		"Result":      "Passed",
		"Source":      "/usr/local/go/bin/go",
		"UID":         float64(1000),
		"Type":        "HostLog",
		"UpdatedTime": "2023-03-31T15:48:15.817142Z",
	}

	// Json unmarshaller converts large numbers to float64 and that is what is being simulated here
	expectedTimestamp := time.Unix(0, int64(float64(1680277695)*1000))
	select {
	case e := <-received:
		require.Equal(t, expected, e.Body)
		require.Equal(t, expectedTimestamp, e.Timestamp)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be read")
	}
}

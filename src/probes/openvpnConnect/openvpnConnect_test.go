package openvpnConnect

import (
	"boogieman/src/model"
	"context"
	"fmt"
	"github.com/go-cmd/cmd"
	"log"
	"os"
	"testing"
	"time"
)

var (
	openvpnServerTestConfigPath      = "test/openvpn-server.ovpn"
	openvpnClientTestConfigPath      = "test/openvpn-client.ovpn"
	openvpnClientTestWrongConfigPath = "test/openvpn-client-wrong-addr.ovpn"
)

func Test_Runner(t *testing.T) {
	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 2000, Expect: true}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{ConfigData: "empty\nremote 1.1.1.1 1000", LogDump: false},
			false,
		},
		{
			Config{ConfigData: "empty"},
			false,
		},
		{
			Config{ConfigData: testReadConfig(openvpnClientTestWrongConfigPath)},
			false,
		},
		{
			Config{ConfigData: testReadConfig(openvpnClientTestConfigPath)},
			true,
		},
		{
			Config{ConfigFile: openvpnClientTestConfigPath},
			true,
		},
	}
	serverProcess := testStartOpenvpnServer(ctx)

	if serverProcess == nil {
		t.Fatalf("Can't start testing openvpn server")
	}

	for i, c := range cases {
		p := New(options, c.config)
		ctx := model.ContextWithLogger(ctx, model.NewChainLogger(model.DefaultLogger, fmt.Sprintf("test %v", i+1)))
		if p.Start(ctx) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		} else {
			p.Finish(ctx)
		}
	}

	if serverProcess != nil {
		err := serverProcess.Stop()
		if err != nil {
			t.Errorf("something got wrong with stopping openvpn server process")
		}
	}
}

func testStartOpenvpnServer(ctx context.Context) (runner *cmd.Cmd) {
	runner, err := openvpnStart(ctx, openvpnServerTestConfigPath, 1*time.Second, false)
	if err != nil {
		log.Printf("can't start openvpn server: %v", err)
		runner = nil
	}
	return
}

func testReadConfig(filePath string) string {
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("can't read file %v: %v", filePath, err)
		return ""
	}
	return string(b)
}

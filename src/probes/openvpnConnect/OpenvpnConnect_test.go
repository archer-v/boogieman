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

func TestPOpenvpnConnect_Runner(t *testing.T) {
	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 2000}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{ConfigFile: "empty\nremote 1.1.1.1 1000"},
			false,
		},
		{
			Config{ConfigFile: "empty"},
			false,
		},
		{
			Config{ConfigFile: testReadConfig(openvpnClientTestWrongConfigPath)},
			false,
		},
		{
			Config{ConfigFile: testReadConfig(openvpnClientTestConfigPath)},
			true,
		},
	}
	serverProcess := testStartOpenvpnServer(ctx)

	if serverProcess == nil {
		t.Fatalf("Can't start testing openvpn server")
	}

	for i, c := range cases {
		p := New(options, c.config)
		if p.Start(context.WithValue(ctx, "id", fmt.Sprintf("test %v", i+1))) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		} else {
			p.Finish()
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

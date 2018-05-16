package execd

import (
	"fmt"
	"os"
	"testing"
	"time"

	ps "github.com/keybase/go-ps"
	"github.com/moby/moby/pkg/reexec"
)

func init() {
	reexec.Register("execd-test-run", func() {
		var i int
		for {
			time.Sleep(1 * time.Second)
			fmt.Println("===> execd-test: ", i)
			i++
		}
	})

	reexec.Register("execd-test-stop", func() {
		var i int
		for {
			time.Sleep(1 * time.Second)
			fmt.Println("===> execd-test: ", i)
			i++
		}
	})

}

func TestRun(t *testing.T) {

	if reexec.Init() {
		os.Exit(0)
	}

	cmd := DaemonFrom(reexec.Command("execd-test-run"))
	err := cmd.RunForever()
	if err != nil {
		t.Error(err)
	}

	timer := time.NewTimer(3 * time.Second)
	killer := time.NewTimer(500 * time.Millisecond)

LOOP:
	for {
		select {
		case <-timer.C:
			p, err := ps.FindProcess(cmd.Command().Process.Pid)
			if p == nil && err == nil {
				t.Error("no process")
			}
			break LOOP
		case <-killer.C:
			cmd.Command().Process.Kill()
		}
	}

	cmd.Stop()
	time.Sleep(time.Second)

}

func TestStop(t *testing.T) {
	if reexec.Init() {
		os.Exit(0)
	}

	cmd := DaemonFrom(reexec.Command("execd-test-stop"))
	err := cmd.RunForever()
	if err != nil {
		t.Error(err)
	}
	cmd.SetGracePeriod(500 * time.Millisecond)

	cmd.Stop()
	<-time.After(1 * time.Second)

	if cmd.IsRunning() {
		t.Error("still running")
	}
}

func TestCrasLoopBackoff(t *testing.T) {
	cmd := &D{
		Path: "/not-found-path",
		Args: []string{"execd-test-crash"},
	}

	cmd.keepalive()
	cmd.reportError()
	<-time.After(5 * time.Second)

	if cmd.IsRunning() {
		t.Error("still running")
	}
	cmd.Stop()
}

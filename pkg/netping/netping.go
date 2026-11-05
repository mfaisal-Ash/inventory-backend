package netping

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

var ErrInvalidTarget = errors.New("alamat IP aset tidak valid")

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Result struct {
	Online bool

	RTT time.Duration
}

func Check(target string, timeout time.Duration) (Result, error) {
	ip := net.ParseIP(target)
	if ip == nil || ip.IsLoopback() || ip.IsMulticast() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
		return Result{}, ErrInvalidTarget
	}
	// Do not allow this endpoint to probe cloud metadata or private networks.
	if ip.IsPrivate() {
		return Result{}, ErrInvalidTarget
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Second)
	defer cancel()

	if timeout > 10*time.Second {
		timeout = 10 * time.Second
	}
	waitSecs := strconv.Itoa(maxInt(1, int(timeout.Seconds())))
	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"-n", "1", "-w", strconv.Itoa(int(timeout / time.Millisecond)), target}
	} else {
		args = []string{"-c", "1", "-W", waitSecs, target}
	}
	cmd := exec.CommandContext(ctx, "ping", args...)
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	if ctx.Err() != nil {
		return Result{}, fmt.Errorf("ping ke %s timeout: %w", target, ctx.Err())
	}
	if err != nil {

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Result{Online: false}, nil
		}
		return Result{}, fmt.Errorf("gagal menjalankan ping: %w", err)
	}
	return Result{Online: true, RTT: elapsed}, nil
}

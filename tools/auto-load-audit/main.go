// auto-load-audit is an explicitly opted-in, loopback-only synthetic-user audit.
// It never reads installed accounts/configs, changes production enrollment,
// or offers a way to direct load at a remote address.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

var sourceCommit = "unrecorded"
var errIntegrity = errors.New("synthetic payload integrity mismatch")

type resources struct {
	HarnessRSSKB uint64 `json:"harnessRssKb"`
	XrayRSSKB    uint64 `json:"xrayRssKb"`
	HarnessFDs   int    `json:"harnessFds"`
	Goroutines   int    `json:"goroutines"`
	AutoInUse    int    `json:"autoInUse"`
}

type phaseResult struct {
	Name                string               `json:"name"`
	Users               int                  `json:"syntheticUsers"`
	Connections         int                  `json:"connectionsPerUser"`
	Attempts            int                  `json:"connectionAttempts"`
	Concurrency         int                  `json:"attemptConcurrency"`
	HoldAll             bool                 `json:"heldUntilAllAttemptsFinished"`
	Capacity            int                  `json:"enrolledCapacity,omitempty"`
	InitialSuccess      int                  `json:"initialSuccess"`
	VerifiedSuccess     int                  `json:"verifiedSuccess"`
	UsersAll            int                  `json:"usersAllConnectionsVerified"`
	UsersPartial        int                  `json:"usersPartiallyConnected"`
	UsersNone           int                  `json:"usersWithoutVerifiedConnection"`
	Failures            map[string]int       `json:"initialErrors"`
	LostAfterOpen       map[string]int       `json:"errorsAfterEstablishment"`
	SuccessP50MS        float64              `json:"successP50Ms"`
	SuccessP95MS        float64              `json:"successP95Ms"`
	FailureP95MS        float64              `json:"failureP95Ms"`
	DurationMS          int64                `json:"durationMs"`
	Peak                resources            `json:"peak"`
	HarnessCPUTicks     uint64               `json:"harnessCpuTicks"`
	XrayCPUTicks        uint64               `json:"xrayCpuTicks"`
	StateWhileFull      string               `json:"stateWhileLoaded,omitempty"`
	GenerationBefore    uint64               `json:"generationBefore,omitempty"`
	GenerationAfter     uint64               `json:"generationAfter,omitempty"`
	FlowsWhileLoaded    *peerauto.FlowStatus `json:"flowsWhileLoaded,omitempty"`
	FlowsAfterClose     *peerauto.FlowStatus `json:"flowsAfterClose,omitempty"`
	PeerFlowsAfterClose *peerauto.FlowStatus `json:"peerFlowsAfterClose,omitempty"`
	DataWhileLoaded     *peerauto.DataStatus `json:"dataWhileLoaded,omitempty"`
	PeakQueued          int                  `json:"peakQueuedTcp"`
}

type report struct {
	Source          string         `json:"sourceCommit"`
	Go              string         `json:"go"`
	Kernel          string         `json:"kernel"`
	CPUCount        int            `json:"hostCpuCount"`
	GOMAXPROCS      int            `json:"gomaxprocs"`
	ClockTicks      string         `json:"clockTicksPerSecond"`
	XrayVersion     string         `json:"xrayVersion"`
	XraySHA256      string         `json:"xraySha256"`
	Started         time.Time      `json:"started"`
	Finished        time.Time      `json:"finished"`
	Phases          []phaseResult  `json:"phases"`
	UDP             map[string]any `json:"udpCapacity"`
	InvariantErrors []string       `json:"invariantErrors"`
}

func emit(v any) { _ = json.NewEncoder(os.Stdout).Encode(v) }

func procRSS(pid int) uint64 {
	b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) > 1 && f[0] == "VmRSS:" {
			n, _ := strconv.ParseUint(f[1], 10, 64)
			return n
		}
	}
	return 0
}

func cpuTicks(pid int) uint64 {
	b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	at := strings.LastIndexByte(string(b), ')')
	if at < 0 {
		return 0
	}
	f := strings.Fields(string(b[at+1:]))
	if len(f) < 13 {
		return 0
	}
	u, _ := strconv.ParseUint(f[11], 10, 64)
	s, _ := strconv.ParseUint(f[12], 10, 64)
	return u + s
}

func sampleResources(x *xrayFixture, a *autoFixture) resources {
	fds, _ := os.ReadDir("/proc/self/fd")
	r := resources{HarnessRSSKB: procRSS(os.Getpid()), HarnessFDs: len(fds), Goroutines: runtime.NumGoroutine()}
	if x != nil {
		r.XrayRSSKB = procRSS(x.cmd.Process.Pid)
	}
	if a != nil {
		r.AutoInUse = a.iran.Snapshot().Flows.Live
	}
	return r
}

func maxResources(a, b resources) resources {
	return resources{max(a.HarnessRSSKB, b.HarnessRSSKB), max(a.XrayRSSKB, b.XrayRSSKB), max(a.HarnessFDs, b.HarnessFDs), max(a.Goroutines, b.Goroutines), max(a.AutoInUse, b.AutoInUse)}
}

func errorKind(err error) string {
	var n net.Error
	switch {
	case errors.Is(err, errIntegrity):
		return "integrity"
	case errors.As(err, &n) && n.Timeout():
		return "timeout"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return "eof"
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE):
		return "reset"
	default:
		return "other"
	}
}

func echoCheck(c net.Conn, payload []byte) error {
	if err := c.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	if _, err := c.Write(payload); err != nil {
		return err
	}
	b := make([]byte, len(payload))
	if _, err := io.ReadFull(c, b); err != nil {
		return err
	}
	if !bytes.Equal(b, payload) {
		return errIntegrity
	}
	return nil
}

func dialUser(ctx context.Context, address, backend, password string) (net.Conn, error) {
	c, err := (&net.Dialer{Timeout: 4 * time.Second}).DialContext(ctx, "tcp4", address)
	if err != nil {
		return nil, err
	}
	if password == "" {
		return c, nil
	}
	return &ssConn{Conn: c, master: ssMaster(password), writePending: ssDestination(backend)}, nil
}

func percentile(v []float64, p int) float64 {
	if len(v) == 0 {
		return 0
	}
	slices.Sort(v)
	return v[min(len(v)-1, len(v)*p/100)]
}

func runPhase(ctx context.Context, name, address, backend string, users, perUser, concurrent, capacity int, x *xrayFixture, a *autoFixture) phaseResult {
	// Isolate each measurement from the fixture's previous, already closed
	// sockets and pooled buffers. This is test-harness housekeeping, not a
	// production GC setting or an allowance to evict live application flows.
	debug.FreeOSMemory()
	start := time.Now()
	r := phaseResult{Name: name, Users: users, Connections: perUser, Attempts: users * perUser, Concurrency: concurrent, Capacity: capacity,
		Failures: map[string]int{}, LostAfterOpen: map[string]int{}}
	beforeCPU, beforeXray := cpuTicks(os.Getpid()), cpuTicks(x.cmd.Process.Pid)
	if a != nil {
		r.GenerationBefore = a.iran.Snapshot().Generation
	}
	var mu sync.Mutex
	var goodTimes, badTimes []float64
	okByUser := make([]int, users)
	type heldConn struct {
		c       net.Conn
		user    int
		payload []byte
	}
	var held []heldConn
	var workers sync.WaitGroup
	stopSamples, samplesDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(samplesDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			v := sampleResources(x, a)
			mu.Lock()
			r.Peak = maxResources(r.Peak, v)
			if a != nil {
				r.PeakQueued = max(r.PeakQueued, a.iran.Snapshot().Flows.QueuedTCP)
			}
			mu.Unlock()
			select {
			case <-stopSamples:
				return
			case <-ticker.C:
			}
		}
	}()
	gate := make(chan struct{}, concurrent)
	barrier := make(chan struct{})
	burst := concurrent == r.Attempts && name != "auto-queued" || strings.HasSuffix(name, "-held")
	r.HoldAll = burst
	for task := range r.Attempts {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-barrier
			select {
			case gate <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-gate }()
			user := task / perUser
			password := ""
			if name != "plain-direct" {
				password = x.passwords[user]
			}
			payload := make([]byte, 1024)
			_, _ = rand.Read(payload)
			copy(payload, fmt.Sprintf("FullPack synthetic user %d connection %d", user, task%perUser))
			at := time.Now()
			c, err := dialUser(ctx, address, backend, password)
			if err == nil {
				err = echoCheck(c, payload)
			}
			ms := float64(time.Since(at).Microseconds()) / 1000
			if err != nil {
				if c != nil {
					c.Close()
				}
				mu.Lock()
				r.Failures[errorKind(err)]++
				badTimes = append(badTimes, ms)
				mu.Unlock()
				return
			}
			mu.Lock()
			r.InitialSuccess++
			goodTimes = append(goodTimes, ms)
			mu.Unlock()
			if burst {
				mu.Lock()
				held = append(held, heldConn{c, user, payload})
				mu.Unlock()
			} else {
				err = echoCheck(c, payload)
				c.Close()
				mu.Lock()
				if err == nil {
					r.VerifiedSuccess++
					okByUser[user]++
				} else {
					r.LostAfterOpen[errorKind(err)]++
				}
				mu.Unlock()
			}
		}()
	}
	close(barrier)
	workers.Wait()
	if a != nil {
		s := a.iran.Snapshot()
		r.StateWhileFull, r.FlowsWhileLoaded = s.State, &s.Flows
		r.DataWhileLoaded = s.Data
		mu.Lock()
		r.Peak.AutoInUse = max(r.Peak.AutoInUse, s.Flows.Live)
		mu.Unlock()
	}
	// All accepted streams must still exchange their challenge after overflow.
	for _, h := range held {
		workers.Add(1)
		go func() {
			defer workers.Done()
			err := echoCheck(h.c, h.payload)
			h.c.Close()
			mu.Lock()
			if err == nil {
				r.VerifiedSuccess++
				okByUser[h.user]++
			} else {
				r.LostAfterOpen[errorKind(err)]++
			}
			mu.Unlock()
		}()
	}
	workers.Wait()
	close(stopSamples)
	<-samplesDone
	for _, n := range okByUser {
		switch {
		case n == perUser:
			r.UsersAll++
		case n == 0:
			r.UsersNone++
		default:
			r.UsersPartial++
		}
	}
	if a != nil {
		// A backend may keep its permitted TCP half-close drain alive. Join
		// both sides before the next held workload, rather than overlapping
		// the previous peer's relay buffers with a new measurement.
		until := time.Now().Add(35 * time.Second)
		for (a.iran.Snapshot().Flows.Live != 0 || a.peer.Snapshot().Flows.Live != 0) && time.Now().Before(until) {
			time.Sleep(20 * time.Millisecond)
		}
		s := a.iran.Snapshot()
		r.FlowsAfterClose, r.GenerationAfter = &s.Flows, s.Generation
		peer := a.peer.Snapshot()
		r.PeerFlowsAfterClose = &peer.Flows
	}
	r.SuccessP50MS, r.SuccessP95MS, r.FailureP95MS = percentile(goodTimes, 50), percentile(goodTimes, 95), percentile(badTimes, 95)
	r.DurationMS = time.Since(start).Milliseconds()
	r.HarnessCPUTicks, r.XrayCPUTicks = cpuTicks(os.Getpid())-beforeCPU, cpuTicks(x.cmd.Process.Pid)-beforeXray
	return r
}

func udpProbe(address, backend, password string) (*net.UDPConn, error) {
	a, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return nil, err
	}
	c, err := net.DialUDP("udp4", nil, a)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(time.Now().Add(5 * time.Second))
	payload := []byte("FullPack synthetic UDP identity challenge")
	packet, err := ssUDPPacket(password, append(ssDestination(backend), payload...))
	if err == nil {
		_, err = c.Write(packet)
	}
	b := make([]byte, 4096)
	var n int
	if err == nil {
		n, err = c.Read(b)
	}
	if err == nil {
		var plain []byte
		plain, err = ssUDPPlain(password, b[:n])
		if err == nil && (len(plain) < 7 || !bytes.Equal(plain[7:], payload)) {
			err = errIntegrity
		}
	}
	if err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func runUDP(ctx context.Context, backend string, x *xrayFixture) (map[string]any, error) {
	f, err := startAuto(ctx, x.address, 64)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before := f.iran.Snapshot().Generation
	var held []*net.UDPConn
	defer func() {
		for _, c := range held {
			c.Close()
		}
	}()
	udpLimit := f.iran.Snapshot().Flows.UDPLimit
	for i := range udpLimit {
		c, err := udpProbe(f.entry, backend, x.passwords[i%len(x.passwords)])
		if err != nil {
			return nil, fmt.Errorf("synthetic UDP %d: %w", i, err)
		}
		held = append(held, c)
	}
	result := map[string]any{"udpAssociations": len(held), "atCapacity": f.iran.Snapshot().Flows, "generationBefore": before}
	// An additional authenticated UDP identity must be denied at the UDP share,
	// while new TCP must work without waiting for the 60-second UDP idle expiry.
	if c, err := udpProbe(f.entry, backend, x.passwords[0]); err == nil {
		c.Close()
		return result, errors.New("UDP exceeded the reserved share")
	}
	tryTCP := func() string {
		c, err := dialUser(ctx, f.entry, backend, x.passwords[0])
		if err == nil {
			defer c.Close()
			err = echoCheck(c, []byte("synthetic TCP after UDP occupancy"))
		}
		if err == nil {
			return "passed"
		}
		return errorKind(err)
	}
	result["tcpWhileUdpFull"] = tryTCP()
	if result["tcpWhileUdpFull"] != "passed" {
		return result, errors.New("UDP share starved TCP")
	}
	for _, c := range held {
		c.Close()
	}
	start := time.Now()
	time.Sleep(2 * time.Second)
	result["inUseTwoSecondsAfterClientsClose"] = f.iran.Snapshot().Flows.Live
	result["tcpAfterUdpClientsClose"] = tryTCP()
	if result["tcpAfterUdpClientsClose"] != "passed" {
		return result, errors.New("idle UDP blocked TCP")
	}
	until := time.Now().Add(70 * time.Second)
	for f.iran.Snapshot().Flows.Live != 0 && time.Now().Before(until) {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	result["udpCapacityReleaseMs"] = time.Since(start).Milliseconds()
	result["tcpAfterIdleExpiry"] = tryTCP()
	result["generationAfter"] = f.iran.Snapshot().Generation
	result["endState"] = f.iran.Snapshot().State
	return result, nil
}

func main() {
	run := flag.Bool("run", false, "explicitly enable loopback synthetic-user tests")
	binary := flag.String("xray", "", "explicit isolated copy of the installed Xray binary; never enrolled as an Auto helper")
	userFlag := flag.String("users", "500,800", "comma-separated synthetic identities, each in 1..1000")
	output := flag.String("output", "", "JSON report path; contains measurements, no account secrets")
	skipUDP := flag.Bool("skip-udp-hold", false, "skip the approximately one-minute UDP shared-capacity test")
	unlimited := flag.Bool("unlimited", false, "also verify held unlimited Auto at 1, 2 and 4 simultaneous connections per synthetic user")
	unlimitedOnly := flag.Bool("unlimited-only", false, "run direct/unlimited comparisons without repeating historical capped cases")
	setupConcurrency := flag.Int("setup-concurrency", 128, "simultaneous new handshakes for held unlimited comparisons; all accepted connections remain open together")
	connectionCounts := flag.String("connections-per-user", "1,2,4", "held unlimited comparisons per identity; explicit comma-separated subset of 1,2,4")
	flag.Parse()
	var perUserCounts []int
	for _, v := range strings.Split(*connectionCounts, ",") {
		n, err := strconv.Atoi(v)
		if err != nil || n != 1 && n != 2 && n != 4 || slices.Contains(perUserCounts, n) {
			fmt.Fprintln(os.Stderr, "diagnostic connections per user must be a distinct subset of 1,2,4")
			os.Exit(2)
		}
		perUserCounts = append(perUserCounts, n)
	}
	if *setupConcurrency < 1 || *setupConcurrency > 4000 {
		fmt.Fprintln(os.Stderr, "diagnostic setup concurrency must be in 1..4000")
		os.Exit(2)
	}
	if !*run || *binary == "" || runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "Requires --run --xray <isolated-binary> on Linux; all endpoints are loopback only.")
		os.Exit(2)
	}
	var counts []int
	for _, v := range strings.Split(*userFlag, ",") {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			fmt.Fprintln(os.Stderr, "synthetic identities must be in 1..1000")
			os.Exit(2)
		}
		counts = append(counts, n)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, stop := context.WithTimeout(ctx, 15*time.Minute)
	defer stop()
	r := report{Source: sourceCommit, Go: runtime.Version(), CPUCount: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Started: time.Now().UTC(), InvariantErrors: []string{}}
	version, _ := exec.CommandContext(ctx, *binary, "version").Output()
	r.XrayVersion = strings.SplitN(string(version), "\n", 2)[0]
	kernel, _ := exec.CommandContext(ctx, "uname", "-r").Output()
	r.Kernel = strings.TrimSpace(string(kernel))
	ticks, _ := exec.CommandContext(ctx, "getconf", "CLK_TCK").Output()
	r.ClockTicks = strings.TrimSpace(string(ticks))
	if data, err := os.ReadFile(*binary); err == nil {
		sum := sha256.Sum256(data)
		r.XraySHA256 = hex.EncodeToString(sum[:])
	}
	emit(map[string]any{"event": "start", "source": r.Source, "users": counts, "xray": r.XrayVersion})
	err := execute(ctx, *binary, counts, *skipUDP, *unlimited || *unlimitedOnly, *unlimitedOnly, *setupConcurrency, perUserCounts, &r)
	if err != nil {
		r.InvariantErrors = append(r.InvariantErrors, err.Error())
	}
	r.Finished = time.Now().UTC()
	if *output != "" {
		data, _ := json.MarshalIndent(r, "", "  ")
		if e := os.WriteFile(*output, data, 0600); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	emit(map[string]any{"event": "finished", "report": r})
	if len(r.InvariantErrors) > 0 {
		os.Exit(1)
	}
}

func execute(ctx context.Context, binary string, counts []int, skipUDP, unlimited, unlimitedOnly bool, setupConcurrency int, perUserCounts []int, r *report) error {
	echo, err := startEcho(ctx)
	if err != nil {
		return err
	}
	defer echo.Close()
	for _, users := range counts {
		x, err := startXray(ctx, binary, echo.address, users)
		if err != nil {
			return err
		}
		err = func() error {
			defer x.Close()
			// Validate the test client against real Xray before measuring it.
			for _, user := range []int{0, users / 2, users - 1} {
				c, err := dialUser(ctx, x.address, echo.address, x.passwords[user])
				if err != nil {
					return err
				}
				err = echoCheck(c, []byte("synthetic AEAD compatibility check"))
				c.Close()
				if err != nil {
					return fmt.Errorf("Xray fixture compatibility failed: %w", err)
				}
			}
			add := func(p phaseResult) {
				r.Phases = append(r.Phases, p)
				emit(map[string]any{"event": "phase", "result": p})
				if p.Capacity > 0 && p.Peak.AutoInUse > p.Capacity {
					r.InvariantErrors = append(r.InvariantErrors, "admission ceiling escaped")
				}
				if p.GenerationBefore != p.GenerationAfter {
					r.InvariantErrors = append(r.InvariantErrors, "load changed generation")
				}
				if p.FlowsAfterClose != nil && p.FlowsAfterClose.Live != 0 || p.PeerFlowsAfterClose != nil && p.PeerFlowsAfterClose.Live != 0 {
					r.InvariantErrors = append(r.InvariantErrors, "TCP capacity was not released")
				}
				if p.Failures["integrity"] > 0 || p.LostAfterOpen["integrity"] > 0 {
					r.InvariantErrors = append(r.InvariantErrors, "payload corruption")
				}
				if strings.HasPrefix(p.Name, "xray-direct") && p.VerifiedSuccess != p.Attempts {
					r.InvariantErrors = append(r.InvariantErrors, "direct Xray comparison did not serve every connection")
				}
			}
			add(runPhase(ctx, "plain-direct", echo.address, echo.address, users, 1, users, 0, x, nil))
			perUser := []int{1, 2}
			if unlimited {
				perUser = perUserCounts
			}
			for _, per := range perUser {
				name, window := "xray-direct", users*per
				if unlimited {
					name = "xray-direct-held"
					window = min(setupConcurrency, window)
				}
				add(runPhase(ctx, name, x.address, echo.address, users, per, window, 0, x, nil))
			}
			if unlimited {
				f, err := startAuto(ctx, x.address, 0)
				if err != nil {
					return err
				}
				for _, per := range perUser {
					p := runPhase(ctx, "auto-unlimited-held", f.entry, echo.address, users, per, min(setupConcurrency, users*per), 0, x, f)
					add(p)
					if p.VerifiedSuccess != p.Attempts || p.UsersAll != users || len(p.LostAfterOpen) != 0 || p.FlowsWhileLoaded == nil || !p.FlowsWhileLoaded.Unlimited || p.FlowsWhileLoaded.Live != p.Attempts {
						r.InvariantErrors = append(r.InvariantErrors, "unlimited held application workload did not serve every connection")
					}
				}
				f.Close()
			}
			if unlimitedOnly {
				return nil
			}
			for _, limit := range []int{64, 192} {
				f, err := startAuto(ctx, x.address, limit)
				if err != nil {
					return err
				}
				for _, per := range []int{1, 2} {
					add(runPhase(ctx, "auto-burst", f.entry, echo.address, users, per, users*per, limit, x, f))
				}
				add(runPhase(ctx, "auto-queued", f.entry, echo.address, users, 2, min(2*limit, users*2), limit, x, f))
				if limit == 64 {
					add(runPhase(ctx, "auto-paced", f.entry, echo.address, users, 2, min(32, users*2), limit, x, f))
				}
				f.Close()
			}
			if !skipUDP && users == counts[len(counts)-1] {
				emit(map[string]string{"event": "udp_shared_capacity_start"})
				r.UDP, err = runUDP(ctx, echo.address, x)
				emit(map[string]any{"event": "udp_shared_capacity_finished", "result": r.UDP})
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

package sonotui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// PlayerState holds the current playback state from SSE events.
type PlayerState struct {
	Transport string // PLAYING, PAUSED_PLAYBACK, STOPPED, TRANSITIONING
	Volume    int
	Elapsed   int
	Duration  int
	Title     string
	Artist    string
}

// SpectrumFrame holds one FFT snapshot from the /spectrum endpoint.
type SpectrumFrame struct {
	Bands    []float64 `json:"bands"`
	Playing  bool      `json:"playing"`
	TrackGen int       `json:"track_gen"`
}

var (
	baseURL string
	client  = &http.Client{Timeout: 5 * time.Second}
)

// Init reads SONOTUI_URL from the environment.
func Init() {
	baseURL = os.Getenv("SONOTUI_URL")
	if baseURL != "" {
		fmt.Printf("[sonotui] configured: %s\n", baseURL)
	}
}

// Available reports whether a sonotui URL is configured.
func Available() bool { return baseURL != "" }

// FetchStatus returns the current playback state from GET /status.
func FetchStatus() (PlayerState, error) {
	var state PlayerState
	if baseURL == "" {
		return state, fmt.Errorf("not configured")
	}
	resp, err := client.Get(baseURL + "/status")
	if err != nil {
		return state, err
	}
	defer resp.Body.Close()
	var raw struct {
		Transport string `json:"transport"`
		Volume    int    `json:"volume"`
		Elapsed   int    `json:"elapsed"`
		Duration  int    `json:"duration"`
		Track     struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
		} `json:"track"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return state, err
	}
	state.Transport = raw.Transport
	state.Volume = raw.Volume
	state.Elapsed = raw.Elapsed
	state.Duration = raw.Duration
	state.Title = raw.Track.Title
	state.Artist = raw.Track.Artist
	return state, nil
}

// --- Transport actions (all return success bool) ---

func Play() bool  { return post("/play", nil) }
func Pause() bool { return post("/pause", nil) }
func Next() bool  { return post("/next", nil) }
func Prev() bool  { return post("/prev", nil) }

func VolumeUp() bool {
	return post("/volume/relative", map[string]int{"delta": 5})
}

func VolumeDown() bool {
	return post("/volume/relative", map[string]int{"delta": -5})
}

func PlayMood(name string) bool {
	return post("/moods/"+name+"/play", nil)
}

func post(path string, body interface{}) bool {
	if baseURL == "" {
		fmt.Println("[sonotui] not configured")
		return false
	}
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	resp, err := client.Post(baseURL+path, "application/json", r)
	if err != nil {
		fmt.Printf("[sonotui] %s failed: %v\n", path, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// --- SSE subscriber ---

// Subscribe connects to the SSE stream and sends PlayerState updates on the
// returned channel. It auto-reconnects with backoff on failure.
// Close the context to stop.
func Subscribe(ctx context.Context) <-chan PlayerState {
	ch := make(chan PlayerState, 4)
	go func() {
		defer close(ch)
		backoff := time.Second
		for {
			if err := streamSSE(ctx, ch); err != nil {
				if ctx.Err() != nil {
					return
				}
				fmt.Printf("[sonotui] SSE disconnected: %v (retry in %s)\n", err, backoff)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
			} else {
				backoff = time.Second
			}
		}
	}()
	return ch
}

func streamSSE(ctx context.Context, ch chan<- PlayerState) error {
	if baseURL == "" {
		return fmt.Errorf("not configured")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/events", nil)
	sseClient := &http.Client{Timeout: 0} // no timeout for SSE
	resp, err := sseClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("[sonotui] SSE connected")
	var state PlayerState
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]

		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		typ, _ := evt["type"].(string)
		switch typ {
		case "transport":
			if s, ok := evt["state"].(string); ok {
				state.Transport = s
			}
		case "track":
			if s, ok := evt["title"].(string); ok {
				state.Title = s
			}
			if s, ok := evt["artist"].(string); ok {
				state.Artist = s
			}
		case "position":
			if v, ok := evt["elapsed"].(float64); ok {
				state.Elapsed = int(v)
			}
			if v, ok := evt["duration"].(float64); ok {
				state.Duration = int(v)
			}
		case "volume":
			if v, ok := evt["value"].(float64); ok {
				state.Volume = int(v)
			}
		default:
			continue
		}

		select {
		case ch <- state:
		default:
			// drop if consumer is slow
		}
	}
	return scanner.Err()
}

// --- Spectrum poller ---

// PollSpectrum polls the /spectrum endpoint as fast as the server responds
// (back-to-back GETs). Sends SpectrumFrames on the returned channel.
// The interval parameter is used as a minimum sleep between polls to avoid
// busy-spinning if the server responds instantly.
func PollSpectrum(ctx context.Context, bands int, interval time.Duration) <-chan SpectrumFrame {
	ch := make(chan SpectrumFrame, 4)
	go func() {
		defer close(ch)
		url := fmt.Sprintf("%s/spectrum?bands=%d", baseURL, bands)
		pollClient := &http.Client{Timeout: 150 * time.Millisecond}

		var count int
		start := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			reqStart := time.Now()
			var frame SpectrumFrame
			resp, err := pollClient.Get(url)
			if err != nil {
				frame.Bands = make([]float64, bands)
				select {
				case ch <- frame:
				default:
				}
				// Brief sleep on error to avoid hammering.
				time.Sleep(100 * time.Millisecond)
				continue
			}
			json.NewDecoder(resp.Body).Decode(&frame)
			resp.Body.Close()

			select {
			case ch <- frame:
			default:
			}

			// Log effective poll rate every 5 seconds.
			count++
			if elapsed := time.Since(start); elapsed >= 5*time.Second {
				fmt.Printf("[sonotui] spectrum poll rate: %.1f Hz (avg RTT: %dms)\n",
					float64(count)/elapsed.Seconds(),
					int(elapsed.Milliseconds())/count)
				count = 0
				start = time.Now()
			}

			// If the request was very fast, sleep minimally to avoid busy-spinning.
			// The server ticks at 60Hz (~16ms), so RTT alone usually paces us.
			if rtt := time.Since(reqStart); rtt < 2*time.Millisecond {
				time.Sleep(2 * time.Millisecond)
			}
		}
	}()
	return ch
}

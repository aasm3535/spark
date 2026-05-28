package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"yutug.lol/spark/internal/config"
)

// AudioRecorder defines the interface for recording audio.
type AudioRecorder interface {
	Start() error
	Stop(outputPath string) error
}

// NewRecorder returns the platform-specific audio recorder.
func NewRecorder() AudioRecorder {
	if runtime.GOOS == "windows" {
		return &windowsRecorder{}
	}
	return &unixRecorder{}
}

// ─── Windows Recorder (using winmm.dll MCI) ───────────────────────────────────

type windowsRecorder struct{}

var (
	winmm              = syscall.NewLazyDLL("winmm.dll")
	procMciSendStringW = winmm.NewProc("mciSendStringW")

	mciChan chan mciCmd
	mciOnce sync.Once
)

type mciCmd struct {
	cmd      string
	respChan chan error
}

func startMciWorker() {
	mciChan = make(chan mciCmd, 10)
	go func() {
		runtime.LockOSThread()
		for msg := range mciChan {
			msg.respChan <- mciSendStringDirect(msg.cmd)
		}
	}()
}

func mciSendStringDirect(cmd string) error {
	cmdPtr, err := syscall.UTF16PtrFromString(cmd)
	if err != nil {
		return err
	}
	ret, _, _ := procMciSendStringW.Call(
		uintptr(unsafe.Pointer(cmdPtr)),
		0,
		0,
		0,
	)
	if ret != 0 {
		return fmt.Errorf("MCI error: %d", ret)
	}
	return nil
}

func mciSendString(cmd string) error {
	mciOnce.Do(startMciWorker)
	resp := make(chan error, 1)
	mciChan <- mciCmd{cmd: cmd, respChan: resp}
	return <-resp
}

func (w *windowsRecorder) Start() error {
	_ = mciSendString("close recsound") // make sure it's closed
	if err := mciSendString("open new type waveaudio alias recsound"); err != nil {
		return err
	}
	return mciSendString("record recsound")
}

func (w *windowsRecorder) Stop(outputPath string) error {
	defer mciSendString("close recsound")
	_ = mciSendString("stop recsound")

	// Save requires absolute path or clean Windows path separators
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}
	return mciSendString(fmt.Sprintf(`save recsound "%s"`, absPath))
}

// ─── Unix/Mac Recorder (using ffmpeg / rec / arecord) ─────────────────────────

type unixRecorder struct {
	cmd *exec.Cmd
}

func (u *unixRecorder) Start() error {
	// Look for recorders in path
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		// Use ffmpeg for macOS/Linux
		var args []string
		if runtime.GOOS == "darwin" {
			// macOS AVFoundation input
			args = []string{"-y", "-f", "avfoundation", "-i", ":0", "temp_recording.wav"}
		} else {
			// Linux ALSA input
			args = []string{"-y", "-f", "alsa", "-i", "default", "temp_recording.wav"}
		}
		u.cmd = exec.Command(path, args...)
		return u.cmd.Start()
	}

	if path, err := exec.LookPath("rec"); err == nil {
		// SoX rec
		u.cmd = exec.Command(path, "temp_recording.wav")
		return u.cmd.Start()
	}

	if path, err := exec.LookPath("arecord"); err == nil {
		// Linux ALSA arecord
		u.cmd = exec.Command(path, "-f", "cd", "temp_recording.wav")
		return u.cmd.Start()
	}

	return fmt.Errorf("no audio recording CLI utility found (install ffmpeg, sox, or arecord)")
}

func (u *unixRecorder) Stop(outputPath string) error {
	if u.cmd == nil || u.cmd.Process == nil {
		return fmt.Errorf("recorder not running")
	}

	// Send Interrupt signal to gracefully stop writing the WAV file
	_ = u.cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func() {
		done <- u.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		_ = u.cmd.Process.Kill()
	}

	u.cmd = nil

	// Rename temp file to output path
	tempFile := "temp_recording.wav"
	if _, err := os.Stat(tempFile); err == nil {
		err = os.Rename(tempFile, outputPath)
		if err != nil {
			// Try copying if rename fails across filesystems
			src, err := os.Open(tempFile)
			if err != nil {
				return err
			}
			defer src.Close()
			dst, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer dst.Close()
			_, _ = io.Copy(dst, src)
			_ = os.Remove(tempFile)
		}
	} else {
		return fmt.Errorf("recording file was not generated: %v", err)
	}

	return nil
}

// ─── Transcribe API Calls ─────────────────────────────────────────────────────

// Transcribe sends the WAV audio file to the configured STT provider.
func Transcribe(ctx context.Context, cfg config.STTConfig, wavPath string) (string, error) {
	fileData, err := os.ReadFile(wavPath)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	provider := strings.ToLower(cfg.Provider)
	switch provider {
	case "openai":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://api.openai.com/v1/audio/transcriptions"
		}
		model := cfg.Model
		if model == "" {
			model = "whisper-1"
		}
		return transcribeOpenAI(ctx, endpoint, cfg.APIKey, model, fileData)

	case "nvidia":
		endpoint := cfg.Endpoint
		if endpoint == "" {
			endpoint = "https://integrate.api.nvidia.com/v1/audio/transcriptions"
		}
		model := cfg.Model
		if model == "" {
			model = "openai/whisper-large-v3"
		}
		return transcribeOpenAI(ctx, endpoint, cfg.APIKey, model, fileData)

	case "assemblyai":
		return transcribeAssemblyAI(ctx, cfg.APIKey, cfg.Model, fileData)

	default:
		return "", fmt.Errorf("unsupported STT provider: %s", cfg.Provider)
	}
}

// transcribeOpenAI handles both OpenAI and NVIDIA endpoints.
func transcribeOpenAI(ctx context.Context, endpoint, apiKey, model string, wavData []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add model field
	if err := writer.WriteField("model", model); err != nil {
		return "", err
	}

	// Add file field
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, bytes.NewReader(wavData)); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respData))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respData, &res); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return res.Text, nil
}

// transcribeAssemblyAI uploads audio and polls for transcript.
func transcribeAssemblyAI(ctx context.Context, apiKey, model string, wavData []byte) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	// 1. Upload file
	uploadURL := "https://api.assemblyai.com/v2/upload"
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(wavData))
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", apiKey)
	req.Header.Set("content-type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AssemblyAI upload error (%d): %s", resp.StatusCode, string(respData))
	}

	var uploadRes struct {
		UploadURL string `json:"upload_url"`
	}
	if err := json.Unmarshal(respData, &uploadRes); err != nil {
		return "", err
	}

	// 2. Request transcription
	transcriptURL := "https://api.assemblyai.com/v2/transcript"
	
	speechModels := []string{"universal-3-pro", "universal-2"}
	if model != "" && !strings.Contains(strings.ToLower(model), "whisper") {
		speechModels = []string{model, "universal-3-pro", "universal-2"}
	}

	payload := map[string]interface{}{
		"audio_url":     uploadRes.UploadURL,
		"speech_models": speechModels,
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err = http.NewRequestWithContext(ctx, "POST", transcriptURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", apiKey)
	req.Header.Set("content-type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("AssemblyAI transcription request error (%d): %s", resp.StatusCode, string(respData))
	}

	var transcriptRes struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respData, &transcriptRes); err != nil {
		return "", err
	}

	// 3. Poll for result
	pollURL := fmt.Sprintf("https://api.assemblyai.com/v2/transcript/%s", transcriptRes.ID)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err = http.NewRequestWithContext(ctx, "GET", pollURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("authorization", apiKey)

		resp, err = client.Do(req)
		if err != nil {
			return "", err
		}
		respData, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("AssemblyAI poll error (%d): %s", resp.StatusCode, string(respData))
		}

		var pollRes struct {
			Status string `json:"status"`
			Text   string `json:"text"`
			Error  string `json:"error"`
		}
		if err := json.Unmarshal(respData, &pollRes); err != nil {
			return "", err
		}

		if pollRes.Status == "completed" {
			return pollRes.Text, nil
		}
		if pollRes.Status == "error" {
			return "", fmt.Errorf("AssemblyAI transcription failed: %s", pollRes.Error)
		}

		// Wait 1 second before polling again
		time.Sleep(1 * time.Second)
	}
}

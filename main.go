package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-skynet/go-llama.cpp"
)

const (
	DefaultModelPath = "./models/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
	DefaultPort      = 8080
	DefaultTimeout   = 30 * time.Second
)

type (
	AppConfig struct {
		ModelPath              string
		Port, ContextSize      int
		Threads, GPULayers     int
		MaxTokens              int
		Temperature, TopP      float32
		RepeatPenalty          float32
	}
	ChatRequest struct {
		Message string         `json:"message"`
		History []MessageEntry `json:"history,omitempty"`
	}
	ChatResponse struct {
		Response string `json:"response"`
		Tokens   int    `json:"tokens,omitempty"`
		TimeMS   int64  `json:"time_ms,omitempty"`
	}
	PingRequest struct {
		CallbackURL string `json:"callback_url"`
	}
	MessageEntry struct {
		Role, Content string `json:"role","content"`
	}
)

type Server struct {
	cfg   AppConfig
	model *llama.LLama
	lock  sync.Mutex
	start time.Time
}

func main() {
	cfg := parseFlags()
	s := &Server{cfg: cfg, start: time.Now()}

	if err := s.loadModel(); err != nil {
		log.Fatal("Model init:", err)
	}
	defer s.model.Free()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.routes(),
		ReadTimeout:  DefaultTimeout,
		WriteTimeout: DefaultTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Server at :", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Listen error:", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func parseFlags() AppConfig {
	cfg := AppConfig{}
	flag.StringVar(&cfg.ModelPath, "model", DefaultModelPath, "")
	flag.IntVar(&cfg.Port, "port", DefaultPort, "")
	flag.IntVar(&cfg.ContextSize, "context", 2048, "")
	flag.IntVar(&cfg.Threads, "threads", 4, "")
	flag.IntVar(&cfg.GPULayers, "gpu", 0, "")
	flag.IntVar(&cfg.MaxTokens, "tokens", 512, "")
	flag.Float32Var(&cfg.Temperature, "temp", 0.7, "")
	flag.Float32Var(&cfg.TopP, "top_p", 0.5, "")
	flag.Float32Var(&cfg.RepeatPenalty, "repeat", 1.1, "")
	flag.Parse()
	return cfg
}

func (s *Server) loadModel() error {
	m, err := llama.New(
		s.cfg.ModelPath,
		llama.SetContext(s.cfg.ContextSize),
		llama.SetNThreads(s.cfg.Threads),
		llama.SetNGPULayers(s.cfg.GPULayers),
	)
	if err != nil {
		return err
	}
	s.model = m
	return nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", s.chat)
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/ping", s.ping)
	return http.TimeoutHandler(mux, DefaultTimeout, `{"error":"timeout"}`)
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, `{"error":"invalid input"}`, http.StatusBadRequest)
		return
	}

	prompt := s.buildPrompt(req.Message, req.History)
	start := time.Now()
	resp, tokens, err := s.predict(prompt)
	if err != nil {
		http.Error(w, `{"error":"model error"}`, http.StatusInternalServerError)
		return
	}

	s.respond(w, ChatResponse{Response: resp, Tokens: tokens, TimeMS: time.Since(start).Milliseconds()})
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !strings.HasPrefix(req.CallbackURL, "http") {
		http.Error(w, `{"error":"invalid callback_url"}`, http.StatusBadRequest)
		return
	}

	go func(url string) {
		time.Sleep(8 * time.Minute)
		http.Post(url, "application/json", strings.NewReader(`{"status":"pong"}`))
	}(req.CallbackURL)

	s.respond(w, map[string]string{"message": "pong scheduled"})
}

func (s *Server) predict(prompt string) (string, int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	opts := []llama.PredictOption{
		llama.SetTokens(s.cfg.MaxTokens),
		llama.SetTemperature(s.cfg.Temperature),
		llama.SetTopP(s.cfg.TopP),
		llama.SetRepeatPenalty(s.cfg.RepeatPenalty),
		llama.SetStopWords("</s>", "<|user|>"),
	}
	res, err := s.model.Predict(prompt, opts...)
	if err != nil {
		return "", 0, err
	}
	out := strings.TrimSpace(strings.TrimSuffix(res, "</s>"))
	return out, len(res), nil
}

func (s *Server) buildPrompt(msg string, history []MessageEntry) string {
	var sb strings.Builder
	sb.WriteString("<|system|>\nYou are Libra designed By Sanne Karibo, an AI expert market analyst and chatbot, answer questions from analysis.\n</s>\n")
	for _, m := range history {
		sb.WriteString(fmt.Sprintf("<|%s|>\n%s</s>\n", m.Role, m.Content))
	}
	sb.WriteString(fmt.Sprintf("<|user|>\n%s</s>\n<|assistant|>\n", msg))
	return sb.String()
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	s.respond(w, map[string]interface{}{
		"status":    "ok",
		"uptime":    time.Since(s.start).String(),
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) respond(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

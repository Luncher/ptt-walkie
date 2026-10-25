package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/plant-squad/ptt-walkie/internal/archive"
	"github.com/plant-squad/ptt-walkie/internal/store"
	"github.com/plant-squad/ptt-walkie/internal/voice"
)

func TestSummaryFallbackUsesLatestTranscribedRecordings(t *testing.T) {
	for _, llmFails := range []bool{false, true} {
		mode := "without_llm"
		if llmFails {
			mode = "llm_failure"
		}
		for _, tc := range []struct {
			name        string
			transcripts []string
			want        []string
		}{
			{"older_transcript", []string{"前面封路，走辅路", "", "", "", "", ""}, []string{"前面封路，走辅路"}},
			{"latest_five_transcripts", []string{"消息一", "消息二", "", "消息三", "消息四", "", "消息五", "消息六", "消息七", "", "", "", "", ""}, []string{"消息三", "消息四", "消息五", "消息六", "消息七"}},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				st, err := store.New("")
				if err != nil {
					t.Fatal(err)
				}
				g, err := st.CreateGroup("wang", store.CreateGroupInput{Name: "车队"})
				if err != nil {
					t.Fatal(err)
				}
				since := time.Now().Add(-10 * time.Minute)
				dir := t.TempDir()
				f, err := os.Create(filepath.Join(dir, "index.jsonl"))
				if err != nil {
					t.Fatal(err)
				}
				enc := json.NewEncoder(f)
				write := func(id, speaker, text string, at time.Time) {
					t.Helper()
					err := enc.Encode(archive.Record{Utterance: voice.Utterance{ID: id, GroupID: g.ID, SpeakerID: speaker, SpeakerName: speaker, StartedAt: at}, Transcript: text})
					if err != nil {
						t.Fatal(err)
					}
				}
				write("old", "wang", "时间范围之外", since.Add(-time.Minute))
				for i, text := range tc.transcripts {
					write(fmt.Sprint(i), "wang", text, since.Add(time.Duration(i+1)*time.Second))
				}
				write("self", "li", "自己的发言", since.Add(time.Minute))
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				arc, err := archive.Open(dir)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { arc.Close() })
				svc := &Service{Archive: arc, Store: st}
				var calls atomic.Int32
				if llmFails {
					ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
					}))
					defer ts.Close()
					svc.Client = &Client{BaseURL: ts.URL, Key: "test", Model: "test", HTTP: ts.Client()}
				}
				sum, err := svc.Summarize(context.Background(), "li", g.ID, since)
				if err != nil {
					t.Fatal(err)
				}
				var parts []string
				for _, text := range tc.want {
					parts = append(parts, "wang说「"+text+"」")
				}
				want := fmt.Sprintf("过去10分钟车队有%d条消息，最近的：%s。", len(tc.transcripts), strings.Join(parts, "；"))
				if sum.Text != want {
					t.Errorf("summary = %q, want %q", sum.Text, want)
				}
				if sum.ByLLM || sum.Count != len(tc.transcripts) || len(sum.Recordings) != len(tc.transcripts) {
					t.Errorf("unexpected summary metadata: %+v", sum)
				}
				if llmFails && calls.Load() != 1 {
					t.Errorf("LLM calls = %d, want 1", calls.Load())
				}
			})
		}
	}
}

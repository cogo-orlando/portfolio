package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Petit routeur de test : /ok répond 200, /ko répond 500, /panic plante
func testMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/ko", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) { panic("page cassée") })
	return mux
}

func TestSnapshot_PendingBeforeFirstCheck(t *testing.T) {
	m := New(testMux(), []Target{{"OK", "/ok"}}, time.Minute, 10)
	if got := m.Snapshot().Overall; got != "pending" {
		t.Errorf("avant la 1re vérification : attendu pending, obtenu %s", got)
	}
}

func TestCheckAll_AllOperational(t *testing.T) {
	m := New(testMux(), []Target{{"A", "/ok"}, {"B", "/ok"}}, time.Minute, 10)
	m.CheckAll()
	snap := m.Snapshot()
	if snap.Overall != "operational" {
		t.Errorf("attendu operational, obtenu %s", snap.Overall)
	}
	for _, r := range snap.Routes {
		if !r.OK || r.Status != 200 || r.Uptime != 100 {
			t.Errorf("%s : attendu OK/200/100%%, obtenu %v/%d/%v", r.Path, r.OK, r.Status, r.Uptime)
		}
	}
}

func TestCheckAll_Degraded(t *testing.T) {
	m := New(testMux(), []Target{{"OK", "/ok"}, {"KO", "/ko"}}, time.Minute, 10)
	m.CheckAll()
	if got := m.Snapshot().Overall; got != "degraded" {
		t.Errorf("une route en échec : attendu degraded, obtenu %s", got)
	}
}

func TestCheckAll_Down(t *testing.T) {
	m := New(testMux(), []Target{{"KO", "/ko"}, {"404", "/inexistant"}}, time.Minute, 10)
	m.CheckAll()
	if got := m.Snapshot().Overall; got != "down" {
		t.Errorf("toutes les routes en échec : attendu down, obtenu %s", got)
	}
}

// Un panic dans une page ne doit pas arrêter le moniteur
func TestCheckAll_PanicCountsAs500(t *testing.T) {
	m := New(testMux(), []Target{{"Panic", "/panic"}, {"OK", "/ok"}}, time.Minute, 10)
	m.CheckAll()
	snap := m.Snapshot()
	if snap.Routes[0].OK || snap.Routes[0].Status != 500 {
		t.Errorf("un panic doit compter comme 500, obtenu %d", snap.Routes[0].Status)
	}
	if !snap.Routes[1].OK {
		t.Error("les autres routes doivent quand même être vérifiées après un panic")
	}
}

func TestUptime_Percentage(t *testing.T) {
	ok := true
	mux := http.NewServeMux()
	mux.HandleFunc("/flaky", func(w http.ResponseWriter, r *http.Request) {
		if !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	m := New(mux, []Target{{"Flaky", "/flaky"}}, time.Minute, 4)
	for _, state := range []bool{true, true, true, false} {
		ok = state
		m.CheckAll()
	}
	if got := m.Snapshot().Routes[0].Uptime; got != 75 {
		t.Errorf("3 réussites sur 4 : attendu 75%%, obtenu %v", got)
	}
}

// L'historique tourne : seules les N dernières vérifications comptent
func TestUptime_RingBuffer(t *testing.T) {
	ok := false
	mux := http.NewServeMux()
	mux.HandleFunc("/r", func(w http.ResponseWriter, r *http.Request) {
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	m := New(mux, []Target{{"R", "/r"}}, time.Minute, 2)
	m.CheckAll() // échec
	ok = true
	m.CheckAll() // succès
	m.CheckAll() // succès → l'échec sort de la fenêtre de 2
	if got := m.Snapshot().Routes[0].Uptime; got != 100 {
		t.Errorf("fenêtre de 2 : attendu 100%%, obtenu %v", got)
	}
}

func TestHandler_JSON(t *testing.T) {
	m := New(testMux(), []Target{{"OK", "/ok"}}, time.Minute, 10)
	m.CheckAll()

	w := httptest.NewRecorder()
	m.Handler(w, httptest.NewRequest(http.MethodGet, "/api/status", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control attendu no-store, obtenu %q", cc)
	}
	var snap Snapshot
	if err := json.NewDecoder(w.Body).Decode(&snap); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if snap.Overall != "operational" || len(snap.Routes) != 1 {
		t.Errorf("contenu inattendu : %+v", snap)
	}
}

func TestHandler_RejectsPost(t *testing.T) {
	m := New(testMux(), nil, time.Minute, 10)
	w := httptest.NewRecorder()
	m.Handler(w, httptest.NewRequest(http.MethodPost, "/api/status", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST devrait donner 405, obtenu %d", w.Code)
	}
}

// Run s'arrête proprement quand le serveur s'arrête
func TestRun_StopsOnContextCancel(t *testing.T) {
	m := New(testMux(), []Target{{"OK", "/ok"}}, 10*time.Millisecond, 10)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()

	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run ne s'est pas arrêté après l'annulation du contexte")
	}
	if m.Snapshot().Overall != "operational" {
		t.Error("au moins une vérification aurait dû avoir lieu")
	}
}

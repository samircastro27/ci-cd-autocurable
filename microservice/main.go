package main

import (
    "fmt"
    "net/http"
    "math/rand"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    totalRequests = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "demo_requests_total",
        Help: "Número total de solicitudes HTTP",
    })
    totalErrors = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "demo_errors_total",
        Help: "Errores simulados para prueba de healing",
    })
    lastRequestDuration = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "demo_last_request_duration_seconds",
        Help: "Duración de la última solicitud",
    })
)

func main() {
    prometheus.MustRegister(totalRequests, totalErrors, lastRequestDuration)
    rand.Seed(time.Now().UnixNano())

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        delay := time.Duration(rand.Intn(700)) * time.Millisecond
        time.Sleep(delay)

        totalRequests.Inc()
        if rand.Float64() < 0.1 {
            totalErrors.Inc()
            http.Error(w, "Error simulado", http.StatusInternalServerError)
            return
        }
        lastRequestDuration.Set(time.Since(start).Seconds())
        fmt.Fprintln(w, "Microservicio activo - KCD Bogotá 2025")
    })

    http.Handle("/metrics", promhttp.Handler())
    fmt.Println("Escuchando en :8080")
    http.ListenAndServe(":8080", nil)
}

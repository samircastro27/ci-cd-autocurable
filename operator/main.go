package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/rest"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

type Metrics struct {
    ErrorRate float64
    Latency float64
}

func fetchMetrics() (Metrics, error) {
    resp, err := http.Get("http://demo-microservice:8080/metrics")
    if err != nil {
        return Metrics{}, err
    }
    defer resp.Body.Close()

    var m Metrics
    var errors, requests, duration float64

    var line string
    for {
        _, err := fmt.Fscanf(resp.Body, "%s %f\n", &line, &duration)
        if err != nil {
            break
        }
        if line == "demo_errors_total" {
            errors = duration
        } else if line == "demo_requests_total" {
            requests = duration
        } else if line == "demo_last_request_duration_seconds" {
            m.Latency = duration
        }
    }
    if requests > 0 {
        m.ErrorRate = errors / requests
    }
    return m, nil
}

func main() {
    config, _ := rest.InClusterConfig()
    dynClient, _ := dynamic.NewForConfig(config)
    gvr := schema.GroupVersionResource{
        Group: "demo.kcd",
        Version: "v1",
        Resource: "healingpolicies",
    }

    for {
        list, _ := dynClient.Resource(gvr).Namespace("default").List(context.TODO(), metav1.ListOptions{})
        metrics, err := fetchMetrics()
        if err != nil {
            fmt.Println("Error obteniendo métricas:", err)
            continue
        }

        for _, item := range list.Items {
            spec := item.Object["spec"].(map[string]interface{})
            action := spec["action"].(string)
            maxLat := spec["maxLatencySeconds"].(float64)
            maxErr := spec["maxErrorRate"].(float64)

            fmt.Printf("Métricas actuales: latency=%.2f, errorRate=%.2f\n", metrics.Latency, metrics.ErrorRate)

            if metrics.Latency > maxLat || metrics.ErrorRate > maxErr {
                fmt.Println("🔁 Ejecutando acción:", action)
                switch action {
                case "restart":
                    os.system("kubectl rollout restart deployment demo-microservice")
                case "scale":
                    os.system("kubectl scale deployment demo-microservice --replicas=3")
                case "alert":
                    fmt.Println("⚠️ Alerta: se excedieron los límites")
                }
            }
        }
        time.Sleep(30 * time.Second)
    }
}

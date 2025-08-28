# Proyecto: CI/CD Autocurable con Tekton, Go y Kubernetes

Este proyecto demuestra cómo implementar un pipeline CI/CD autocurable utilizando Tekton, Prometheus y un operador personalizado en Go para Kubernetes. Fue diseñado como una demo técnica para eventos como KCD Bogotá 2025.

---

## 📁 Estructura del Proyecto

```
project-demo/
├── microservice/              # Microservicio en Go con métricas Prometheus
│   ├── main.go
│   ├── go.mod
│   ├── Dockerfile
│   └── k8s/                   # Manifiestos Kubernetes
│       ├── deployment.yaml
│       └── service.yaml
├── tekton/                    # Pipeline y Triggers de Tekton
│   ├── pipeline.yaml
│   └── triggers.yaml
├── operator/                  # HealingPolicy y operador en Go
│   ├── main.go
│   ├── go.mod
│   ├── api/v1/
│   │   └── healingpolicy_types.go
│   └── config/
│       ├── crd.yaml
│       └── samples/healingpolicy.yaml
└── monitoring/                # Configuración de Prometheus y Grafana
    ├── prometheus.yaml
    └── grafana.yaml
```

---

## ✅ Requisitos

- Kubernetes (MicroK8s recomendado)
- Docker
- Go >= 1.20
- kubectl
- ko (opcional, para deploy del operador)
- Tekton Pipelines y Triggers instalados

---

## 🚀 Pasos de Implementación

### 1. Microservicio en Go

```bash
cd microservice
go mod tidy
go build -o server .
docker build -t localhost:32000/demo-microservice:latest .
docker push localhost:32000/demo-microservice:latest
kubectl apply -f k8s/
```

### 2. Pipeline y Triggers

```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply -n tekton-pipelines -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.10/git-clone.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/kaniko/0.6/kaniko.yaml -n tekton-pipelines



kubectl apply -f tekton/pipeline.yaml
kubectl apply -f tekton/triggers.yaml
```

> Configura el webhook en GitHub apuntando al EventListener.

### 3. Prometheus y Grafana

```bash
kubectl apply -f monitoring/prometheus.yaml
kubectl apply -f monitoring/grafana.yaml
```

> Accede a Grafana vía NodePort y configura dashboards personalizados.

### 4. CRD y Operador HealingPolicy

```bash
kubectl apply -f operator/config/crd.yaml
# Despliega el operador (usando ko o manifiesto)
kubectl apply -f operator/config/samples/healingpolicy.yaml
```

---

## 🧪 Validación

- Haz push a GitHub para iniciar la pipeline
- Fuerza errores para probar reintento automático
- Genera carga para simular latencia
- Verifica escalamiento y alertas desde el operador
- Visualiza todo en Grafana

---

## 📌 Créditos

Desarrollado por Samir Castro (Cuemby) para demo técnica en KCD Bogotá 2025.

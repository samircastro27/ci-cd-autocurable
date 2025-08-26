package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	// Importar paquetes de k8s, Tekton y esquema API de HealingPolicy
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	// suponiendo que healingpolicy_types.go generó paquete demo.kcd2025/v1alpha1
	demov1alpha1 "github.com/samircastro27/operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HealingPolicyReconciler reconcilia objetos HealingPolicy
type HealingPolicyReconciler struct {
	client.Client
	// normalmente incluiría Scheme, Log, etc.
}

// parseMetricValue busca una métrica por nombre en el texto de métricas y devuelve su valor (float64)
func parseMetricValue(metricsText string, metricName string) float64 {
	for _, line := range strings.Split(metricsText, "\n") {
		if strings.HasPrefix(line, metricName+" ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
					return val
				}
			}
		}
	}
	return 0.0
}

func (r *HealingPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 1. Obtener el objeto HealingPolicy actual
	var policy demov1alpha1.HealingPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		// Si fue borrado, no hay nada que hacer
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Extraer especificaciones para fácil acceso
	hp := policy.Spec
	pipelineName := hp.PipelineName
	pipelineNS := hp.PipelineNamespace
	deployName := hp.DeploymentName
	deployNS := hp.DeploymentNamespace

	// 2. Listar PipelineRuns de Tekton en el namespace objetivo, filtrando por pipelineName
	var prList tektonv1beta1.PipelineRunList
	if err := r.List(ctx, &prList, client.InNamespace(pipelineNS)); err == nil {
		for _, pr := range prList.Items {
			if pr.Spec.PipelineRef != nil && pr.Spec.PipelineRef.Name == pipelineName {
				// Hallamos un PipelineRun de nuestra pipeline objetivo
				for _, cond := range pr.Status.Conditions {
					if cond.Type == "Succeeded" && cond.Status == corev1.ConditionFalse {
						// La condición Succeeded es False => PipelineRun falló
						fmt.Printf("[HealingPolicy] PipelineRun fallida detectada: %s (razón: %s)\n", pr.Name, cond.Reason)
						// Verificar que no haya sido ya re-ejecutada (podríamos marcar en status, pero omitir en demo)
						// 3. Crear un nuevo PipelineRun para reintentar la pipeline
						newPR := tektonv1beta1.PipelineRun{
							ObjectMeta: ctrl.ObjectMeta{
								GenerateName: pipelineName + "-retry-",
								Namespace:    pipelineNS,
							},
							Spec: tektonv1beta1.PipelineRunSpec{
								PipelineRef: &tektonv1beta1.PipelineRef{Name: pipelineName},
								// Podríamos pasar mismos params que el PR original si queremos reintentar misma commit.
								// Simplificamos usando misma pipeline con parámetros por defecto o fijos.
							},
						}
						if err := r.Create(ctx, &newPR); err != nil {
							fmt.Printf("Error re-creando PipelineRun: %v\n", err)
						} else {
							fmt.Printf("[HealingPolicy] Reejecutada PipelineRun nueva: %s\n", newPR.Name)
						}
					}
				}
			}
		}
	}

	// 4. Obtener métricas actuales del microservicio desde Prometheus o directamente vía HTTP
	metricsURL := fmt.Sprintf("http://%s.%s.svc:8080/metrics", deployName, deployNS)
	resp, err := http.Get(metricsURL)
	if err != nil {
		fmt.Printf("No se pudo obtener métricas de %s: %v\n", metricsURL, err)
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		metricsText := string(body)
		// 5. Parsear métricas relevantes
		totalReq := parseMetricValue(metricsText, "demo_requests_total")
		totalErr := parseMetricValue(metricsText, "demo_errors_total")
		lastLat := parseMetricValue(metricsText, "demo_last_request_duration_seconds")

		var currentErrorRate float64 = 0.0
		if totalReq > 0 {
			currentErrorRate = totalErr / totalReq
		}

		fmt.Printf("[HealingPolicy] Métricas actuales -> requests: %.0f, errors: %.0f, lastLatency: %.3fs, errorRate: %.2f%%\n",
			totalReq, totalErr, lastLat, currentErrorRate*100)

		// 6. Verificar umbral de latencia
		if hp.LatencyThresholdSeconds > 0 && lastLat > hp.LatencyThresholdSeconds {
			// Necesario escalar
			var deploy appsv1.Deployment
			if err := r.Get(ctx, client.ObjectKey{Name: deployName, Namespace: deployNS}, &deploy); err == nil {
				// Aumentar replicas en 1 (también podríamos duplicar, o usar un max definido)
				desired := *deploy.Spec.Replicas + 1
				deploy.Spec.Replicas = &desired
				if err := r.Update(ctx, &deploy); err != nil {
					fmt.Printf("Error escalando deployment: %v\n", err)
				} else {
					fmt.Printf("[HealingPolicy] Latencia %.3fs > %.3fs umbral. Réplicas de %s escaladas a %d\n",
						lastLat, hp.LatencyThresholdSeconds, deployName, desired)
				}
			}
		}

		// 7. Verificar umbral de error rate (SLO de errores)
		if hp.ErrorRateThreshold > 0 && currentErrorRate > hp.ErrorRateThreshold {
			// Enviar alerta (aquí simulamos con un log)
			fmt.Printf("[HealingPolicy][ALERTA] Tasa de error actual %.2f%% excede umbral de %.2f%%\n",
				currentErrorRate*100, hp.ErrorRateThreshold*100)
			// (Opcional: también podríamos escalar el Deployment en caso de alto error rate)
		}
	}

	// 8. Programar próxima reconciliación periódica (por ejemplo en 30s) para seguir monitoreando continuamente
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

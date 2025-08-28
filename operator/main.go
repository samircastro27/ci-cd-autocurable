package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	demov1alpha1 "github.com/samircastro27/operator/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var scheme = runtime.NewScheme()

const maxRetries = 5 // 👈 Límite de reintentos para la demo

// HealingPolicyReconciler reconcilia objetos HealingPolicy
type HealingPolicyReconciler struct {
	client.Client
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
	// 1) Obtener HealingPolicy
	var policy demov1alpha1.HealingPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2) Si ya alcanzamos el máximo de reintentos, no crear más
	if policy.Status.RetryCount >= maxRetries {
		fmt.Printf("[HealingPolicy] Max retries (%d) alcanzado; no se crearán más PipelineRuns para %s/%s\n",
			maxRetries, policy.Namespace, policy.Name)
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// 3) Extraer especificaciones
	hp := policy.Spec
	pipelineName := hp.PipelineName
	pipelineNS := hp.PipelineNamespace
	deployName := hp.DeploymentName
	deployNS := hp.DeploymentNamespace

	// 4) Buscar PipelineRuns relacionados y actuar según su estado
	var prList tektonv1beta1.PipelineRunList
	if err := r.List(ctx, &prList, client.InNamespace(pipelineNS)); err == nil {
		for _, pr := range prList.Items {
			// Filtrar por pipeline objetivo
			if pr.Spec.PipelineRef == nil || pr.Spec.PipelineRef.Name != pipelineName {
				continue
			}

			for _, cond := range pr.Status.Conditions {
				// Si hay una exitosa, resetea el contador
				if cond.Type == "Succeeded" && cond.Status == corev1.ConditionTrue {
					if policy.Status.RetryCount != 0 {
						policy.Status.RetryCount = 0
						if err := r.Status().Update(ctx, &policy); err != nil {
							_ = r.Update(ctx, &policy) // fallback si no hay subresource status
						}
						fmt.Printf("[HealingPolicy] Éxito detectado; RetryCount=0 para %s/%s\n", policy.Namespace, policy.Name)
					}
				}

				// Si falló, intenta reintentar (respetando el límite)
				if cond.Type == "Succeeded" && cond.Status == corev1.ConditionFalse {
					if policy.Status.RetryCount >= maxRetries {
						continue
					}

					newPR := tektonv1beta1.PipelineRun{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: pipelineName + "-retry-",
							Namespace:    pipelineNS,
						},
						Spec: tektonv1beta1.PipelineRunSpec{
							PipelineRef: &tektonv1beta1.PipelineRef{Name: pipelineName},
						},
					}
					if err := r.Create(ctx, &newPR); err != nil {
						fmt.Printf("Error re-creando PipelineRun: %v\n", err)
					} else {
						policy.Status.RetryCount++
						if err := r.Status().Update(ctx, &policy); err != nil {
							_ = r.Update(ctx, &policy)
						}
						fmt.Printf("[HealingPolicy] Reintento #%d -> PR %s creado\n", policy.Status.RetryCount, newPR.Name)
					}
				}
			}
		}
	}

	// 5) Métricas del servicio (latencia/errores) — opcional en la demo
	metricsURL := fmt.Sprintf("http://%s.%s.svc:8080/metrics", deployName, deployNS)
	if resp, err := http.Get(metricsURL); err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		metricsText := string(body)

		totalReq := parseMetricValue(metricsText, "demo_requests_total")
		totalErr := parseMetricValue(metricsText, "demo_errors_total")
		lastLat := parseMetricValue(metricsText, "demo_last_request_duration_seconds")

		var currentErrorRate float64 = 0.0
		if totalReq > 0 {
			currentErrorRate = totalErr / totalReq
		}
		fmt.Printf("[HealingPolicy] Métricas -> req: %.0f, err: %.0f, latency: %.3fs, errRate: %.2f%%\n",
			totalReq, totalErr, lastLat, currentErrorRate*100)
	}

	// 6) Requeue periódico
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *HealingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.HealingPolicy{}).
		Owns(&tektonv1beta1.PipelineRun{}).
		Complete(r)
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Registrar esquemas (fail-fast)
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(tektonv1beta1.AddToScheme(scheme))
	utilruntime.Must(demov1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		panic(fmt.Sprintf("unable to start manager: %v", err))
	}

	if err := (&HealingPolicyReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		panic(fmt.Sprintf("unable to create controller: %v", err))
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(fmt.Sprintf("problem running manager: %v", err))
	}
}
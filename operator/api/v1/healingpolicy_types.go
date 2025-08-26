package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type HealingPolicySpec struct {
    PipelineName           string  `json:"pipelineName"`
    PipelineNamespace      string  `json:"pipelineNamespace"`
    DeploymentName         string  `json:"deploymentName"`
    DeploymentNamespace    string  `json:"deploymentNamespace"`
    LatencyThresholdSeconds float64 `json:"latencyThresholdSeconds"`
    ErrorRateThreshold      float64 `json:"errorRateThreshold"`
}

type HealingPolicyStatus struct {
    LastChecked metav1.Time `json:"lastChecked,omitempty"`
    LastAction  string      `json:"lastAction,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type HealingPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HealingPolicySpec   `json:"spec,omitempty"`
    Status HealingPolicyStatus `json:"status,omitempty"`
}
// +kubebuilder:object:root=true
type HealingPolicyList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HealingPolicy `json:"items"`
}

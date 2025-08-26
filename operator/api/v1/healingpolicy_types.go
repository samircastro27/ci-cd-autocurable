package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type HealingPolicySpec struct {
    MaxLatencySeconds float64 `json:"maxLatencySeconds"`
    MaxErrorRate float64 `json:"maxErrorRate"`
    Action string `json:"action"` // restart, scale, alert
}

type HealingPolicyStatus struct {
    LastChecked metav1.Time `json:"lastChecked,omitempty"`
    LastAction  string      `json:"lastAction,omitempty"`
}

type HealingPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HealingPolicySpec   `json:"spec,omitempty"`
    Status HealingPolicyStatus `json:"status,omitempty"`
}

type HealingPolicyList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HealingPolicy `json:"items"`
}

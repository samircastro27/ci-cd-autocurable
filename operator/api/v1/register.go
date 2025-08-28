// operator/api/v1/register.go
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Group y version de tu CRD
var (
	GroupVersion  = schema.GroupVersion{Group: "demo.kcd2025", Version: "v1alpha1"}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&HealingPolicy{},
		&HealingPolicyList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
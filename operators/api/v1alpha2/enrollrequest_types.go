package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnrollRequestSpec struct {
	Workspace string `json:"workspace"`
	Tenant    string `json:"tenant"`
}

type EnrollRequestStatus struct {
	Valid bool `json:"valid"`
}

// +kubebuilder:object:root=true
type EnrollRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnrollRequestSpec   `json:"spec,omitempty"`
	Status EnrollRequestStatus `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EnrollRequest{})
}

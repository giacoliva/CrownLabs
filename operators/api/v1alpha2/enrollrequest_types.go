package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnrollRequestSpec struct {
	Tenant string `json:"tenant"`
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

// +kubebuilder:object:root=true
type EnrollRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []EnrollRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnrollRequest{}, &EnrollRequestList{})
}

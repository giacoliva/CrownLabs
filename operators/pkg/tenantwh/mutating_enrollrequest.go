package tenantwh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/utils"
)

type EnrollRequestMutator struct {
	TenantWebhook
}

func MakeEnrollRequestMutator(c client.Client) *webhook.Admission {
	return &webhook.Admission{Handler: &EnrollRequestMutator{}}
}

func (erm *EnrollRequestMutator) Handle(ctx context.Context, req admission.Request) admission.Response { //nolint:gocritic // the signature of this method is imposed by controller runtime.
	log := ctrl.LoggerFrom(ctx).WithName("enrollrequest-mutator").WithValues("username", req.UserInfo.Username, "enrollrequest", req.Name, "namespace", req.Namespace)
	_ = ctrl.LoggerInto(ctx, log)

	log.V(utils.LogDebugLevel).Info("processing mutation request", "user", req.UserInfo.Username, "namespace", req.Namespace)

	enrollrequest, err := erm.DecodeEnrollRequest(req.Object)
	if err != nil {
		log.Error(err, "enrollrequest decode from request failed")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := erm.CheckOrInsertTenant(log, enrollrequest, req.UserInfo.Username); err != nil {
		log.Error(err, "tenant check failed")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := erm.CheckName(log, enrollrequest, req.UserInfo.Username); err != nil {
		log.Error(err, "name check failed")
		return admission.Errored(http.StatusBadRequest, err)
	}

	return erm.CreatePatchResponse(ctx, &req, enrollrequest)
}

func (erm *EnrollRequestMutator) DecodeEnrollRequest(obj runtime.RawExtension) (enrollrequest *clv1alpha2.EnrollRequest, err error) {
	enrollrequest = &clv1alpha2.EnrollRequest{}
	err = erm.decoder.DecodeRaw(obj, enrollrequest)
	return
}

func (erm *EnrollRequestMutator) CheckOrInsertTenant(log logr.Logger, enrollrequest *clv1alpha2.EnrollRequest, username string) error {
	if enrollrequest.Spec.Tenant == "" {
		erm.InjectTenant(log, enrollrequest, username)
	} else if enrollrequest.Spec.Tenant != username {
		err := fmt.Errorf("tenant field is not empty and does not match the username")
		return err
	}
	return nil
}

func (erm *EnrollRequestMutator) CheckName(log logr.Logger, enrollrequest *clv1alpha2.EnrollRequest, username string) error {
	if enrollrequest.Name != username {
		err := fmt.Errorf("name field does not match the username")
		return err
	}
	return nil
}

func (erm *EnrollRequestMutator) InjectTenant(log logr.Logger, enrollrequest *clv1alpha2.EnrollRequest, username string) {
	enrollrequest.Spec.Tenant = username
	log.V(utils.LogDebugLevel).Info("tenant injected", "tenant", enrollrequest.Spec.Tenant)
}

func (erm *EnrollRequestMutator) CreatePatchResponse(ctx context.Context, req *admission.Request, enrollrequest *clv1alpha2.EnrollRequest) admission.Response {
	marshaledEnrollRequest, err := json.Marshal(enrollrequest)
	if err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "patch response creation failed")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledEnrollRequest)
}

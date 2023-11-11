package tenant_controller

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	crownlabsv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	crownlabsv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
)

type EnrollRequestReconciler struct {
	client.Client
	// This function, if configured, is deferred at the beginning of the Reconcile.
	// Specifically, it is meant to be set to GinkgoRecover during the tests,
	// in order to lead to a controlled failure in case the Reconcile panics.
	ReconcileDeferHook func()
}

func (r *EnrollRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.ReconcileDeferHook != nil {
		defer r.ReconcileDeferHook()
	}

	var er crownlabsv1alpha2.EnrollRequest
	if err := r.Get(ctx, req.NamespacedName, &er); client.IgnoreNotFound(err) != nil {
		klog.Errorf("error retrieving enrollrequest before starting reconcile: %s", err)
		return ctrl.Result{}, err
	} else if err != nil {
		klog.Infof("enrollrequest %s deleted", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	ws, err := r.RetriveWorkspace(ctx, req)
	if err != nil {
		klog.Errorf("error retrieving workspace before starting reconcile: %s", err)
		return ctrl.Result{}, err
	}

	tn, err := r.RetriveTenant(ctx, er.Spec.Tenant)
	if err != nil {
		klog.Errorf("error retrieving tenant before starting reconcile: %s", err)
		return ctrl.Result{}, err
	}

	if !r.WorkspaceAcceptAutoenroll(ws) {
		klog.Infof("workspace %s does not accept autoenroll", ws.Name)
		if err := r.Delete(ctx, &er); err != nil {
			klog.Errorf("error deleting enrollrequest %s: %s", er.Name, err)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// check if already enrolled

	if r.WorkspaceAcceptSelfenroll(ws) {
		klog.Infof("selfenrolling %s into %s", tn.Name, ws.Name)
		if err := r.EnrollTenant(ctx, tn, ws); err != nil {
			klog.Errorf("error enrolling tenant %s into workspace %s: %s", tn.Name, ws.Name, err)
			return ctrl.Result{}, err
		}
		if err := r.Delete(ctx, &er); err != nil {
			klog.Errorf("error deleting enrollrequest %s: %s", er.Name, err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers a new controller for Tenant resources.
func (r *EnrollRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crownlabsv1alpha2.EnrollRequest{}).
		Watches(&source.Kind{Type: &crownlabsv1alpha1.Workspace{}},
			handler.EnqueueRequestsFromMapFunc(r.workspaceToEnrollRequests)).
		Complete(r)
}

func (r *EnrollRequestReconciler) ExtractWorkspaceName(req ctrl.Request) (string, error) {
	ns := req.Namespace
	if strings.HasPrefix(ns, "workspace-") {
		return ns[10:], nil
	} else {
		return "", fmt.Errorf("namespace %s is not a workspace namespace", ns)
	}
}

func (r *EnrollRequestReconciler) RetriveWorkspace(ctx context.Context, req ctrl.Request) (*crownlabsv1alpha1.Workspace, error) {
	wsname, err := r.ExtractWorkspaceName(req)
	if err != nil {
		return nil, err
	}

	wsLookupKey := types.NamespacedName{Name: wsname}

	var ws crownlabsv1alpha1.Workspace
	if err := r.Get(ctx, wsLookupKey, &ws); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return &ws, nil
}

func (r *EnrollRequestReconciler) RetriveTenant(ctx context.Context, tenantName string) (*crownlabsv1alpha2.Tenant, error) {
	tenantLookupKey := types.NamespacedName{Name: tenantName}

	var tenant crownlabsv1alpha2.Tenant
	if err := r.Get(ctx, tenantLookupKey, &tenant); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return &tenant, nil
}

func (r *EnrollRequestReconciler) WorkspaceAcceptAutoenroll(ws *crownlabsv1alpha1.Workspace) bool {
	return ws.Spec.AutoEnroll.Enabled
}

func (r *EnrollRequestReconciler) WorkspaceAcceptSelfenroll(ws *crownlabsv1alpha1.Workspace) bool {
	return r.WorkspaceAcceptAutoenroll(ws) && ws.Spec.AutoEnroll.SelfEnroll
}

func (r *EnrollRequestReconciler) EnrollTenant(ctx context.Context, tn *crownlabsv1alpha2.Tenant, ws *crownlabsv1alpha1.Workspace) error {
	tn.Spec.Workspaces = append(tn.Spec.Workspaces, crownlabsv1alpha2.TenantWorkspaceEntry{
		Name: ws.Name,
		Role: crownlabsv1alpha2.User,
	})
	return r.Update(ctx, tn)
}

func (r *EnrollRequestReconciler) workspaceToEnrollRequests(o client.Object) []ctrl.Request {
	var ers crownlabsv1alpha2.EnrollRequestList
	wsname := fmt.Sprintf("workspace-%s", o.GetName())
	if err := r.List(context.Background(), &ers, client.InNamespace(wsname)); err != nil {
		klog.Errorf("error listing enrollrequests: %s", err)
		return nil
	}
	var enqueues []ctrl.Request = make([]ctrl.Request, len(ers.Items))
	for i := range ers.Items {
		enqueues[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ers.Items[i].GetName(),
				Namespace: ers.Items[i].GetNamespace(),
			},
		}
	}
	return enqueues
}

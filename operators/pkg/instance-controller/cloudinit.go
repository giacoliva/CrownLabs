// Copyright 2020-2021 Politecnico di Torino
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instance_controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clv1alpha1 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha1"
	clctx "github.com/netgroup-polito/CrownLabs/operators/pkg/context"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/utils"
)

const (
	// WebdavSecretUsernameKey -> the key of the webdav secret containing the username.
	WebdavSecretUsernameKey = "username"
	// WebdavSecretPasswordKey -> The key of the webdav secret containing the password.
	WebdavSecretPasswordKey = "password"

	// UserDataKey -> the key of the created secret containing the cloud-init userdata content.
	UserDataKey = "userdata"
)

// EnforceCloudInitSecret enforces the creation/update of a secret containing the cloud-init configuration,
// based on the information retrieved for the tenant object and its associated WebDav credentials.
func (r *InstanceReconciler) EnforceCloudInitSecret(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)

	// Retrieve the WebDav credentials.
	user, password, err := r.GetWebDavCredentials(ctx)
	if err != nil {
		log.Error(err, "unable to get webdav credentials")
		return err
	}
	log.V(utils.LogDebugLevel).Info("webdav credentials correctly retrieved")

	// Retrieve the public keys
	publicKeys, err := r.GetPublicKeys(ctx)
	if err != nil {
		log.Error(err, "unable to get public keys")
		return err
	}
	log.V(utils.LogDebugLevel).Info("public keys correctly retrieved")

	userdata, err := forge.CloudInitUserData(r.ServiceUrls.NextcloudBaseURL, user, password, publicKeys)
	if err != nil {
		log.Error(err, "unable to marshal secret content")
		return err
	}

	// Enforce the cloud-init secret presence
	instance := clctx.InstanceFrom(ctx)
	secret := corev1.Secret{ObjectMeta: forge.ObjectMeta(instance)}
	res, err := ctrl.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		secret.SetLabels(forge.InstanceObjectLabels(secret.GetLabels(), instance))
		secret.Data = map[string][]byte{UserDataKey: userdata}
		secret.Type = corev1.SecretTypeOpaque
		return ctrl.SetControllerReference(instance, &secret, r.Scheme)
	})

	if err != nil {
		log.Error(err, "failed to enforce cloud-init secret", "secret", klog.KObj(&secret))
		return err
	}

	log.V(utils.FromResult(res)).Info("cloud-init secret enforced", "secret", klog.KObj(&secret), "result", res)
	return nil
}

// GetWebDavCredentials extracts the credentials (i.e. username and password)
// required to mount the MyDrive disk of a given tenant from the associated secret.
func (r *InstanceReconciler) GetWebDavCredentials(ctx context.Context) (username, password string, err error) {
	instance := clctx.InstanceFrom(ctx)
	namespacedName := forge.NamespacedName(instance)
	secretName := types.NamespacedName{Namespace: namespacedName.Namespace, Name: r.WebdavSecretName}

	secret := corev1.Secret{}
	if err = r.Get(ctx, secretName, &secret); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to retrieve secret", "secret", secretName)
		return
	}

	var ok bool
	var userBytes, passBytes []byte

	if userBytes, ok = secret.Data[WebdavSecretUsernameKey]; !ok {
		err = fmt.Errorf("cannot find %v key in secret", WebdavSecretUsernameKey)
		ctrl.LoggerFrom(ctx).Error(err, "failed to retrieve credentials from secret", "secret", secretName)
		return
	}

	if passBytes, ok = secret.Data[WebdavSecretPasswordKey]; !ok {
		err = fmt.Errorf("cannot find %v key in secret", WebdavSecretPasswordKey)
		ctrl.LoggerFrom(ctx).Error(err, "failed to retrieve credentials from secret", "secret", secretName)
		return
	}

	return string(userBytes), string(passBytes), nil
}

// GetPublicKeys extracts and returns the set of public keys associated with a
// given tenant, along with the ones of the tenants having Manager role in the
// corresponding workspace.
func (r *InstanceReconciler) GetPublicKeys(ctx context.Context) ([]string, error) {
	log := ctrl.LoggerFrom(ctx)

	// Retrieve the public keys from the tenant owning the instance.
	tenant := clctx.TenantFrom(ctx)
	publicKeys := append(make([]string, 0), tenant.Spec.PublicKeys...)
	log.V(utils.LogDebugLevel).Info("public keys correctly retrieved", "number", len(publicKeys))

	// Retrieve the template associated with the instance to retrieve the name of the workspace.
	template := clctx.TemplateFrom(ctx)
	workspaceName := template.Spec.WorkspaceRef.Name
	labelSelector := map[string]string{clv1alpha1.WorkspaceLabelPrefix + workspaceName: string(clv1alpha1.Manager)}

	var managers clv1alpha1.TenantList
	if err := r.List(ctx, &managers, client.MatchingLabels(labelSelector)); err != nil {
		log.Error(err, "failed to retrieve managers for workspace", "workspace", workspaceName, "selector", labelSelector)
		return nil, err
	}

	log.V(utils.LogDebugLevel).Info("found managers for workspace", "number", len(managers.Items), "workspace", workspaceName)
	for i := range managers.Items {
		// Do not append if the instance owner is also a manager, to avoid duplicates.
		if managers.Items[i].Name != tenant.Name {
			publicKeys = append(publicKeys, managers.Items[i].Spec.PublicKeys...)
		}
	}

	return publicKeys, nil
}

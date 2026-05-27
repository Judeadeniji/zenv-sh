/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/Judeadeniji/zenv-sh/k8s/api/v1alpha1"
	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
)

const (
	zenvsecretFinalizer = "core.zenv.sh/finalizer"
)

// ZEnvSecretReconciler reconciles a ZEnvSecret object
type ZEnvSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.zenv.sh,resources=zenvsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.zenv.sh,resources=zenvsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.zenv.sh,resources=zenvsecrets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// zenvClient defines the methods we use from the zEnv SDK
type zenvClient interface {
	FetchAllSecrets(env string) (map[string]string, error)
}

// newZenvClientFactory allows overriding the client initialization in tests
var newZenvClientFactory = func(opts ...zenv.ClientOption) (zenvClient, error) {
	return zenv.NewClient(opts...)
}

func (r *ZEnvSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ZEnvSecret instance
	var zenvSecret corev1alpha1.ZEnvSecret
	if err := r.Get(ctx, req.NamespacedName, &zenvSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch credentials
	token, projectKey, err := r.resolveCredentials(ctx, &zenvSecret)
	if err != nil {
		log.Error(err, "Failed to resolve credentials")
		r.updateStatus(ctx, &zenvSecret, false, err.Error())
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil // Retry soon on auth error
	}

	// Initialize the ZEnv SDK client
	clientOpts := []zenv.ClientOption{
		zenv.WithToken(token),
		zenv.WithProjectID(zenvSecret.Spec.ProjectID),
		zenv.WithVaultKey(projectKey),
	}

	zClient, err := newZenvClientFactory(clientOpts...)
	if err != nil {
		log.Error(err, "Failed to initialize zEnv client")
		r.updateStatus(ctx, &zenvSecret, false, err.Error())
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Fetch the environment secrets
	secrets, err := zClient.FetchAllSecrets(zenvSecret.Spec.Environment)
	if err != nil {
		log.Error(err, "Failed to fetch environment from zEnv")
		r.updateStatus(ctx, &zenvSecret, false, err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil // Requeue anyway just in case API comes back
	}

	// Upsert the target v1.Secret
	if err := r.upsertTargetSecret(ctx, &zenvSecret, secrets); err != nil {
		log.Error(err, "Failed to upsert target Kubernetes Secret")
		r.updateStatus(ctx, &zenvSecret, false, err.Error())
		return ctrl.Result{}, err
	}

	log.Info("Successfully synced ZEnvSecret", "targetSecret", zenvSecret.Spec.TargetSecret)
	r.updateStatus(ctx, &zenvSecret, true, "Successfully synced secrets")

	// Requeue after 5 minutes to continuously sync
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ZEnvSecretReconciler) resolveCredentials(ctx context.Context, zenvSecret *corev1alpha1.ZEnvSecret) (token string, projectKey string, err error) {
	// Fall back to global env vars initially
	token = os.Getenv("ZENV_TOKEN")
	projectKey = os.Getenv("ZENV_PROJECT_KEY")

	// If a credentials reference is provided, prefer that
	if zenvSecret.Spec.CredentialsRef != nil {
		var credsSecret corev1.Secret
		err := r.Get(ctx, types.NamespacedName{
			Name:      zenvSecret.Spec.CredentialsRef.Name,
			Namespace: zenvSecret.Namespace,
		}, &credsSecret)
		if err != nil {
			return "", "", fmt.Errorf("failed to fetch credentialsRef secret: %w", err)
		}

		if t, ok := credsSecret.Data["token"]; ok {
			token = string(t)
		}
		if pk, ok := credsSecret.Data["projectKey"]; ok {
			projectKey = string(pk)
		}
	}

	if token == "" || projectKey == "" {
		return "", "", fmt.Errorf("missing credentials (token or projectKey not found)")
	}

	return token, projectKey, nil
}

func (r *ZEnvSecretReconciler) upsertTargetSecret(ctx context.Context, zenvSecret *corev1alpha1.ZEnvSecret, zEnvData map[string]string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zenvSecret.Spec.TargetSecret,
			Namespace: zenvSecret.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Type = corev1.SecretTypeOpaque

		// Set owner reference so it gets deleted when ZEnvSecret is deleted
		if err := controllerutil.SetControllerReference(zenvSecret, secret, r.Scheme); err != nil {
			return err
		}

		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}

		// Update data
		for k, v := range zEnvData {
			secret.Data[k] = []byte(v)
		}

		// Clean up keys that no longer exist in zEnv
		for k := range secret.Data {
			if _, exists := zEnvData[k]; !exists {
				delete(secret.Data, k)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if op != controllerutil.OperationResultNone {
		logf.FromContext(ctx).Info("Upserted target secret", "operation", op)
	}

	return nil
}

func (r *ZEnvSecretReconciler) updateStatus(ctx context.Context, zenvSecret *corev1alpha1.ZEnvSecret, ready bool, msg string) {
	status := metav1.ConditionFalse
	reason := "SyncFailed"
	if ready {
		status = metav1.ConditionTrue
		reason = "SyncSuccessful"

		now := metav1.Now()
		zenvSecret.Status.LastSyncedAt = &now
	}

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}

	// Update condition
	found := false
	for i, c := range zenvSecret.Status.Conditions {
		if c.Type == "Ready" {
			zenvSecret.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		zenvSecret.Status.Conditions = append(zenvSecret.Status.Conditions, condition)
	}

	_ = r.Status().Update(ctx, zenvSecret)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZEnvSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.ZEnvSecret{}).
		Owns(&corev1.Secret{}).
		Named("zenvsecret").
		Complete(r)
}

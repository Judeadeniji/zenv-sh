package controller

import (
	"context"
	"errors"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/Judeadeniji/zenv-sh/k8s/api/v1alpha1"
	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
)

type mockZenvClient struct {
	envData map[string]string
	err     error
}

func (m *mockZenvClient) FetchAllSecrets(env string) (map[string]string, error) {
	return m.envData, m.err
}

var _ = Describe("ZEnvSecret Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		AfterEach(func() {
			os.Unsetenv("ZENV_TOKEN")
			os.Unsetenv("ZENV_PROJECT_KEY")
		})

		It("should successfully reconcile with a credentialsRef", func() {
			resourceName := "test-resource-credsref"
			namespace := "default"
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			// Create the credentials secret
			credsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-creds",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"token":      []byte("ze_test_token"),
					"projectKey": []byte("pk_test_key"),
				},
			}
			Expect(k8sClient.Create(ctx, credsSecret)).To(Succeed())

			// Create the ZEnvSecret
			zenvsecret := &corev1alpha1.ZEnvSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: corev1alpha1.ZEnvSecretSpec{
					ProjectID:    "proj_123",
					Environment:  "production",
					TargetSecret: "my-target-secret",
					CredentialsRef: &corev1.LocalObjectReference{
						Name: "my-creds",
					},
				},
			}
			Expect(k8sClient.Create(ctx, zenvsecret)).To(Succeed())

			// Mock the SDK
			newZenvClientFactory = func(opts ...zenv.ClientOption) (zenvClient, error) {
				return &mockZenvClient{
					envData: map[string]string{
						"DB_PASSWORD": "supersecret",
					},
				}, nil
			}

			controllerReconciler := &ZEnvSecretReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(5 * time.Minute))

			// Verify target secret was created
			targetSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "my-target-secret", Namespace: namespace}, targetSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(targetSecret.Data["DB_PASSWORD"])).To(Equal("supersecret"))

			// Verify status
			err = k8sClient.Get(ctx, typeNamespacedName, zenvsecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(zenvsecret.Status.Conditions)).To(Equal(1))
			Expect(zenvsecret.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(zenvsecret.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
		})

		It("should fall back to global env vars if credentialsRef is omitted", func() {
			resourceName := "test-resource-globalenv"
			namespace := "default"
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			os.Setenv("ZENV_TOKEN", "ze_global_token")
			os.Setenv("ZENV_PROJECT_KEY", "pk_global_key")

			zenvsecret := &corev1alpha1.ZEnvSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: corev1alpha1.ZEnvSecretSpec{
					ProjectID:    "proj_456",
					Environment:  "development",
					TargetSecret: "my-target-secret-global",
				},
			}
			Expect(k8sClient.Create(ctx, zenvsecret)).To(Succeed())

			newZenvClientFactory = func(opts ...zenv.ClientOption) (zenvClient, error) {
				return &mockZenvClient{
					envData: map[string]string{
						"API_KEY": "global-secret",
					},
				}, nil
			}

			controllerReconciler := &ZEnvSecretReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify target secret was created
			targetSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "my-target-secret-global", Namespace: namespace}, targetSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(targetSecret.Data["API_KEY"])).To(Equal("global-secret"))
		})

		It("should set Error condition if SDK fetch fails", func() {
			resourceName := "test-resource-error"
			namespace := "default"
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			os.Setenv("ZENV_TOKEN", "ze_global_token")
			os.Setenv("ZENV_PROJECT_KEY", "pk_global_key")

			zenvsecret := &corev1alpha1.ZEnvSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: corev1alpha1.ZEnvSecretSpec{
					ProjectID:    "proj_error",
					Environment:  "development",
					TargetSecret: "my-target-secret-error",
				},
			}
			Expect(k8sClient.Create(ctx, zenvsecret)).To(Succeed())

			newZenvClientFactory = func(opts ...zenv.ClientOption) (zenvClient, error) {
				return &mockZenvClient{
					err: errors.New("network timeout"),
				}, nil
			}

			controllerReconciler := &ZEnvSecretReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// Controller returns nil error but sets status to Failed
			Expect(err).NotTo(HaveOccurred())

			// Target secret should not exist
			targetSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "my-target-secret-error", Namespace: namespace}, targetSecret)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())

			// Verify status
			err = k8sClient.Get(ctx, typeNamespacedName, zenvsecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(zenvsecret.Status.Conditions)).To(Equal(1))
			Expect(zenvsecret.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(zenvsecret.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(zenvsecret.Status.Conditions[0].Message).To(Equal("network timeout"))
		})
	})
})

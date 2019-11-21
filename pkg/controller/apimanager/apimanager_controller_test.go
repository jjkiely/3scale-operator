package apimanager

import (
	"context"
	"fmt"
	"testing"

	"github.com/3scale/3scale-operator/pkg/3scale/amp/operator"
	"github.com/3scale/3scale-operator/pkg/3scale/amp/product"
	appsv1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/3scale/3scale-operator/version"
	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestAPIManagerControllerCreate(t *testing.T) {
	var (
		name           = "example-apimanager"
		namespace      = "operator-unittest"
		wildcardDomain = "test.3scale.net"
	)

	apimanager := &appsv1alpha1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1alpha1.APIManagerSpec{
			APIManagerCommonSpec: appsv1alpha1.APIManagerCommonSpec{
				WildcardDomain: wildcardDomain,
			},
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{apimanager}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(appsv1alpha1.SchemeGroupVersion, apimanager)
	err := appsv1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Apps scheme: (%v)", err)
	}
	err = imagev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Image scheme: (%v)", err)
	}
	err = routev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Route scheme: (%v)", err)
	}

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	clientAPIReader := fake.NewFakeClient(objs...)

	baseReconciler := operator.NewBaseReconciler(cl, clientAPIReader, s, log)
	baseControllerReconciler := operator.NewBaseControllerReconciler(baseReconciler)

	// Create a ReconcileMemcached object with the scheme and fake client.
	r := ReconcileAPIManager{
		BaseControllerReconciler: baseControllerReconciler,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconcile did not requeue request as expected. Requeuing due to setting of defaults should have been performed")
	}

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	if res.Requeue {
		t.Error("reconcile did not finish end of reconciliation as expected. APIManager should have been reconciled at this point")
	}

	finalAPIManager := &appsv1alpha1.APIManager{}
	err = r.Client().Get(context.TODO(), req.NamespacedName, finalAPIManager)
	if err != nil {
		t.Fatalf("get APIManager: (%v)", err)
	}

	backendListenerExistingReplicas := finalAPIManager.Spec.Backend.ListenerSpec.Replicas
	if backendListenerExistingReplicas == nil {
		t.Errorf("APIManager's backend listener replicas does not have a default value set")

	}

	if *backendListenerExistingReplicas != 1 {
		t.Errorf("APIManager's backend listener replicas size (%d) is not the expected size (%d)", backendListenerExistingReplicas, 1)
	}
}

func TestAPIManagerControllerUpgrade(t *testing.T) {
	var (
		name           = "example-apimanager"
		namespace      = "operator-unittest"
		wildcardDomain = "test.3scale.net"
	)

	apimanager := &appsv1alpha1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				appsv1alpha1.OperatorVersionAnnotation: fmt.Sprintf("not_%s", version.Version),
			},
		},
		Spec: appsv1alpha1.APIManagerSpec{
			APIManagerCommonSpec: appsv1alpha1.APIManagerCommonSpec{
				WildcardDomain: wildcardDomain,
			},
		},
	}

	systemApp := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-app",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Strategy: appsv1.DeploymentStrategy{
				RollingParams: &appsv1.RollingDeploymentStrategyParams{
					Pre: &appsv1.LifecycleHook{
						ExecNewPod: &appsv1.ExecNewPodHook{
							Command: []string{},
							Env:     []corev1.EnvVar{},
						},
					},
				},
			},
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{apimanager, systemApp}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(appsv1alpha1.SchemeGroupVersion, apimanager)
	err := appsv1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Apps scheme: (%v)", err)
	}
	err = imagev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Image scheme: (%v)", err)
	}
	err = routev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("Unable to add Route scheme: (%v)", err)
	}

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	clientAPIReader := fake.NewFakeClient(objs...)

	baseReconciler := operator.NewBaseReconciler(cl, clientAPIReader, s, log)
	baseControllerReconciler := operator.NewBaseControllerReconciler(baseReconciler)

	// Create a ReconcileMemcached object with the scheme and fake client.
	r := ReconcileAPIManager{
		BaseControllerReconciler: baseControllerReconciler,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconcile did not requeue request as expected. Requeuing due to setting of defaults should have been performed")
	}

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	if !res.Requeue {
		t.Error("upgrade not expected to finish. Should requeue before update annotations")
	}

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	finalAPIManager := &appsv1alpha1.APIManager{}
	err = r.Client().Get(context.TODO(), req.NamespacedName, finalAPIManager)
	if err != nil {
		t.Fatalf("get APIManager: (%v)", err)
	}

	threeScaleVersion, ok := finalAPIManager.Annotations[appsv1alpha1.ThreescaleVersionAnnotation]
	if !ok {
		t.Errorf("APIManager cr does not have ThreescaleVersionAnnotation annotation set")
	}

	if threeScaleVersion != product.ThreescaleRelease {
		t.Errorf("APIManager cr ThreescaleVersionAnnotation value (%s) not the expected (%s)", threeScaleVersion, product.ThreescaleRelease)
	}

	operatorVersion, ok := finalAPIManager.Annotations[appsv1alpha1.OperatorVersionAnnotation]
	if !ok {
		t.Errorf("APIManager cr does not have OperatorVersionAnnotation annotation set")
	}

	if operatorVersion != version.Version {
		t.Errorf("APIManager cr OperatorVersionAnnotation value (%s) not the expected (%s)", operatorVersion, version.Version)
	}
}

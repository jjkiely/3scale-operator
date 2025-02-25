package operator

import (
	appsv1alpha1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	"github.com/3scale/3scale-operator/pkg/3scale/amp/component"
	"github.com/3scale/3scale-operator/pkg/helper"
	"github.com/3scale/3scale-operator/pkg/reconcilers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type BackendReconciler struct {
	*BaseAPIManagerLogicReconciler
}

func NewBackendReconciler(baseAPIManagerLogicReconciler *BaseAPIManagerLogicReconciler) *BackendReconciler {
	return &BackendReconciler{
		BaseAPIManagerLogicReconciler: baseAPIManagerLogicReconciler,
	}
}

func (r *BackendReconciler) Reconcile() (reconcile.Result, error) {
	backend, err := Backend(r.apiManager, r.Client())
	if err != nil {
		return reconcile.Result{}, err
	}

	// Cron DC
	cronConfigMutator := reconcilers.GenericBackendMutators()
	if r.apiManager.Spec.Backend.CronSpec.Replicas != nil {
		cronConfigMutator = append(cronConfigMutator, reconcilers.DeploymentConfigReplicasMutator)
	}

	err = r.ReconcileDeploymentConfig(backend.CronDeploymentConfig(), reconcilers.DeploymentConfigMutator(cronConfigMutator...))
	if err != nil {
		return reconcile.Result{}, err
	}

	// Listener DC
	backendRedisSecret := &v1.Secret{}
	r.Client().Get(r.Context(), types.NamespacedName{
		Name:      "backend-redis",
		Namespace: r.apiManager.Namespace,
	}, backendRedisSecret)
	redisQueuesUrl := strings.TrimSuffix(string(backendRedisSecret.Data["REDIS_QUEUES_URL"]), "1")
	redisStorageUrl := strings.TrimSuffix(string(backendRedisSecret.Data["REDIS_STORAGE_URL"]), "0")

	listenerConfigMutator := reconcilers.GenericBackendMutators()
	if redisStorageUrl != redisQueuesUrl {
		listenerConfigMutator = append(listenerConfigMutator, reconcilers.DeploymentConfigListenerEnvMutator)
		listenerConfigMutator = append(listenerConfigMutator, reconcilers.DeploymentConfigListenerArgsMutator)
	}
	if r.apiManager.Spec.Backend.ListenerSpec.Replicas != nil {
		listenerConfigMutator = append(listenerConfigMutator, reconcilers.DeploymentConfigReplicasMutator)
	}

	err = r.ReconcileDeploymentConfig(backend.ListenerDeploymentConfig(), reconcilers.DeploymentConfigMutator(listenerConfigMutator...))
	if err != nil {
		return reconcile.Result{}, err
	}

	// Listener Service
	err = r.ReconcileService(backend.ListenerService(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Listener Route
	err = r.ReconcileRoute(backend.ListenerRoute(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Worker DC
	workerConfigMutator := reconcilers.GenericBackendMutators()
	if redisStorageUrl != redisQueuesUrl {
		workerConfigMutator = append(workerConfigMutator, reconcilers.DeploymentConfigWorkerEnvMutator)
	}
	if r.apiManager.Spec.Backend.WorkerSpec.Replicas != nil {
		workerConfigMutator = append(workerConfigMutator, reconcilers.DeploymentConfigReplicasMutator)
	}

	err = r.ReconcileDeploymentConfig(backend.WorkerDeploymentConfig(), reconcilers.DeploymentConfigMutator(workerConfigMutator...))
	if err != nil {
		return reconcile.Result{}, err
	}

	// Environment ConfigMap
	err = r.ReconcileConfigMap(backend.EnvironmentConfigMap(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Internal API Secret
	err = r.ReconcileSecret(backend.InternalAPISecretForSystem(), reconcilers.DefaultsOnlySecretMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Listener Secret
	err = r.ReconcileSecret(backend.ListenerSecret(), reconcilers.DefaultsOnlySecretMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Worker PDB
	err = r.ReconcilePodDisruptionBudget(backend.WorkerPodDisruptionBudget(), reconcilers.GenericPDBMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Cron PDB
	err = r.ReconcilePodDisruptionBudget(backend.CronPodDisruptionBudget(), reconcilers.GenericPDBMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Listener PDB
	err = r.ReconcilePodDisruptionBudget(backend.ListenerPodDisruptionBudget(), reconcilers.GenericPDBMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ReconcilePodMonitor(backend.BackendWorkerPodMonitor(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ReconcilePodMonitor(backend.BackendListenerPodMonitor(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	sumRate, err := helper.SumRateForOpenshiftVersion(r.Context(), r.Client())
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ReconcileGrafanaDashboard(backend.BackendGrafanaDashboard(sumRate), reconcilers.GenericGrafanaDashboardsMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ReconcilePrometheusRules(backend.BackendWorkerPrometheusRules(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ReconcilePrometheusRules(backend.BackendListenerPrometheusRules(), reconcilers.CreateOnlyMutator)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func Backend(apimanager *appsv1alpha1.APIManager, client client.Client) (*component.Backend, error) {
	optsProvider := NewOperatorBackendOptionsProvider(apimanager, apimanager.Namespace, client)
	opts, err := optsProvider.GetBackendOptions()
	if err != nil {
		return nil, err
	}
	return component.NewBackend(opts), nil
}

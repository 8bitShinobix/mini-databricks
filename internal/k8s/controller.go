package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Controller struct {
	client *kubernetes.Clientset
}

func NewController() (*Controller, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &Controller{client: client}, nil
}

func (c *Controller) CreateWorkerJob(ctx context.Context, jobID, workspaceID, runID string, numWorkers int32) error {
	namespace := "default"
	jobName := fmt.Sprintf("worker-%s", runID[:8])

	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "mini-databricks-worker",
				"job-id":       jobID,
				"workspace-id": workspaceID,
				"run-id":       runID,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: &numWorkers,
			Completions: &numWorkers,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":    "mini-databricks-worker",
						"run-id": runID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "worker",
							Image:           "mini-databricks-worker:latest",
							ImagePullPolicy: corev1.PullNever,
							Env: []corev1.EnvVar{
								{Name: "RUN_ID", Value: runID},
								{Name: "JOB_ID", Value: jobID},
								{Name: "DB_URL", ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "mini-databricks-secrets"},
										Key:                  "db-url",
									},
								}},
								{Name: "KAFKA_BROKERS", ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "mini-databricks-secrets"},
										Key:                  "kafka-brokers",
									},
								}},
								{Name: "REDIS_URL", ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "mini-databricks-secrets"},
										Key:                  "redis-url",
									},
								}},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.client.BatchV1().Jobs(namespace).Create(ctx, k8sJob, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create k8s job: %w", err)
	}

	slog.Info("created k8s job", "job_name", jobName, "run_id", runID)
	return nil
}

func (c *Controller) DeleteWorkerJob(ctx context.Context, runID string) error {
	namespace := "default"
	jobName := fmt.Sprintf("worker-%s", runID[:8])

	propagation := metav1.DeletePropagationForeground
	err := c.client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		return fmt.Errorf("failed to delete k8s job: %w", err)
	}

	slog.Info("deleted k8s job", "job_name", jobName, "run_id", runID)
	return nil
}

func (c *Controller) EnsureNamespace(ctx context.Context, workspaceID string) error {
	namespace := fmt.Sprintf("workspace-%s", workspaceID[:8])

	_, err := c.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"workspace-id": workspaceID,
			},
		},
	}
	_, err = c.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	slog.Info("created namespace", "namespace", namespace, "workspace_id", workspaceID)

	if err := c.ApplyNetworkPolicy(ctx, workspaceID); err != nil {
		slog.Warn("failed to apply network policy", "error", err)
	}
	if err := c.ApplyResourceQuota(ctx, workspaceID); err != nil {
		slog.Warn("failed to apply resource quota", "error", err)
	}

	return nil
}

func (c *Controller) GetWorkerReplicas(ctx context.Context) (int32, error) {
	deployment, err := c.client.AppsV1().Deployments("default").Get(ctx, "mini-databricks-worker", metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get worker deployment: %w", err)
	}
	if deployment.Spec.Replicas == nil {
		return 1, nil
	}
	return *deployment.Spec.Replicas, nil
}

func (c *Controller) SetWorkerReplicas(ctx context.Context, replicas int32) error {
	deployment, err := c.client.AppsV1().Deployments("default").Get(ctx, "mini-databricks-worker", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get worker deployment: %w", err)
	}
	deployment.Spec.Replicas = &replicas
	_, err = c.client.AppsV1().Deployments("default").Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update worker replicas: %w", err)
	}
	slog.Info("set worker replicas", "replicas", replicas)
	return nil
}

func (c *Controller) ApplyNetworkPolicy(ctx context.Context, workspaceID string) error {
	namespace := fmt.Sprintf("workspace-%s", workspaceID[:8])
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deny-cross-tenant",
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{PodSelector: &metav1.LabelSelector{}},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{PodSelector: &metav1.LabelSelector{}},
					},
				},
			},
		},
	}
	_, err := c.client.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil {
		if isAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to apply network policy: %w", err)
	}
	slog.Info("applied network policy", "workspace_id", workspaceID)
	return nil
}

func (c *Controller) ApplyResourceQuota(ctx context.Context, workspaceID string) error {
	namespace := fmt.Sprintf("workspace-%s", workspaceID[:8])
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant-quota",
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("10"),
			},
		},
	}
	_, err := c.client.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
	if err != nil {
		if isAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to apply resource quota: %w", err)
	}
	slog.Info("applied resource quota", "workspace_id", workspaceID)
	return nil
}

func isAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

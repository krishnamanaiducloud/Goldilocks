package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpaclient "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
)

const collectionTimeout = 10 * time.Second

// VPACollector exports the recommendations calculated by Kubernetes VPA.
// Values are read at scrape time so Prometheus sees the current VPA status
// instead of a second, potentially stale recommendation calculation.
type VPACollector struct {
	client vpaclient.Interface
	cpu    *prometheus.Desc
	memory *prometheus.Desc
	up     *prometheus.Desc
}

func NewVPACollector(client vpaclient.Interface) *VPACollector {
	labels := []string{"namespace", "vpa", "target_kind", "target_name", "container", "bound"}
	return &VPACollector{
		client: client,
		cpu: prometheus.NewDesc(
			"goldilocks_vpa_recommendation_cpu_cores",
			"Current CPU recommendation calculated by Kubernetes VPA in cores.",
			labels,
			nil,
		),
		memory: prometheus.NewDesc(
			"goldilocks_vpa_recommendation_memory_bytes",
			"Current memory recommendation calculated by Kubernetes VPA in bytes.",
			labels,
			nil,
		),
		up: prometheus.NewDesc(
			"goldilocks_vpa_collector_up",
			"Whether the most recent VPA recommendation collection succeeded.",
			nil,
			nil,
		),
	}
}

func (c *VPACollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpu
	ch <- c.memory
	ch <- c.up
}

func (c *VPACollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectionTimeout)
	defer cancel()

	vpas, err := c.client.AutoscalingV1().VerticalPodAutoscalers(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

	for _, item := range vpas.Items {
		targetKind, targetName := "", ""
		if item.Spec.TargetRef != nil {
			targetKind = item.Spec.TargetRef.Kind
			targetName = item.Spec.TargetRef.Name
		}
		if item.Status.Recommendation == nil {
			continue
		}
		for _, container := range item.Status.Recommendation.ContainerRecommendations {
			bounds := []struct {
				name      string
				resources corev1.ResourceList
			}{
				{"lower_bound", container.LowerBound},
				{"target", container.Target},
				{"upper_bound", container.UpperBound},
				{"uncapped_target", container.UncappedTarget},
			}
			for _, bound := range bounds {
				labels := []string{item.Namespace, item.Name, targetKind, targetName, container.ContainerName, bound.name}
				if quantity, ok := bound.resources[corev1.ResourceCPU]; ok {
					ch <- prometheus.MustNewConstMetric(c.cpu, prometheus.GaugeValue, quantity.AsApproximateFloat64(), labels...)
				}
				if quantity, ok := bound.resources[corev1.ResourceMemory]; ok {
					ch <- prometheus.MustNewConstMetric(c.memory, prometheus.GaugeValue, quantity.AsApproximateFloat64(), labels...)
				}
			}
		}
	}
}

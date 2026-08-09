package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpafake "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/fake"
)

func TestVPACollectorExportsBaseUnits(t *testing.T) {
	client := vpafake.NewSimpleClientset(&vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "goldilocks-api", Namespace: "production"},
		Spec: vpav1.VerticalPodAutoscalerSpec{TargetRef: &autoscalingv1.CrossVersionObjectReference{
			Kind: "Deployment", Name: "api",
		}},
		Status: vpav1.VerticalPodAutoscalerStatus{Recommendation: &vpav1.RecommendedPodResources{
			ContainerRecommendations: []vpav1.RecommendedContainerResources{{
				ContainerName: "api",
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}},
		}},
	})

	collector := NewVPACollector(client)
	expected := `
# HELP goldilocks_vpa_collector_up Whether the most recent VPA recommendation collection succeeded.
# TYPE goldilocks_vpa_collector_up gauge
goldilocks_vpa_collector_up 1
# HELP goldilocks_vpa_recommendation_cpu_cores Current CPU recommendation calculated by Kubernetes VPA in cores.
# TYPE goldilocks_vpa_recommendation_cpu_cores gauge
goldilocks_vpa_recommendation_cpu_cores{bound="target",container="api",namespace="production",target_kind="Deployment",target_name="api",vpa="goldilocks-api"} 0.25
# HELP goldilocks_vpa_recommendation_memory_bytes Current memory recommendation calculated by Kubernetes VPA in bytes.
# TYPE goldilocks_vpa_recommendation_memory_bytes gauge
goldilocks_vpa_recommendation_memory_bytes{bound="target",container="api",namespace="production",target_kind="Deployment",target_name="api",vpa="goldilocks-api"} 1.34217728e+08
`
	if err := testutil.CollectAndCompare(collector, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}

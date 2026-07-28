package kube

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// istio VirtualService GVR
var virtualServiceGVR = schema.GroupVersionResource{
	Group:    "networking.istio.io",
	Version:  "v1beta1",
	Resource: "virtualservices",
}

// DiscoverBasePathFromVirtualService queries the Istio VirtualService by name
// and namespace, then extracts the first HTTP route URI prefix to use as
// the dashboard base path.
//
// Returns the discovered path (e.g. "/goldilocks") or empty string if the
// VirtualService is not found or has no prefix match.
func DiscoverBasePathFromVirtualService(vsName, vsNamespace string) (string, error) {
	if vsName == "" || vsNamespace == "" {
		return "", fmt.Errorf("virtual service name and namespace are required")
	}

	dynClient := GetDynamicInstance()
	vs, err := dynClient.Client.Resource(virtualServiceGVR).Namespace(vsNamespace).Get(
		context.TODO(), vsName, metav1.GetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("error fetching VirtualService %s/%s: %w", vsNamespace, vsName, err)
	}

	// Navigate: spec.http[].match[].uri.prefix
	spec, ok := vs.Object["spec"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("VirtualService %s/%s has no spec", vsNamespace, vsName)
	}

	httpRoutes, ok := spec["http"].([]interface{})
	if !ok || len(httpRoutes) == 0 {
		return "", fmt.Errorf("VirtualService %s/%s has no http routes", vsNamespace, vsName)
	}

	// Search through HTTP routes for the first uri prefix match
	for _, route := range httpRoutes {
		routeMap, ok := route.(map[string]interface{})
		if !ok {
			continue
		}

		matches, ok := routeMap["match"].([]interface{})
		if !ok {
			continue
		}

		for _, match := range matches {
			matchMap, ok := match.(map[string]interface{})
			if !ok {
				continue
			}

			uri, ok := matchMap["uri"].(map[string]interface{})
			if !ok {
				continue
			}

			// Check for prefix match (most common for path-based routing)
			if prefix, ok := uri["prefix"].(string); ok && prefix != "" {
				path := normalizePath(prefix)
				klog.V(2).Infof("Discovered base path %q from VirtualService %s/%s", path, vsNamespace, vsName)
				return path, nil
			}

			// Also check for exact match
			if exact, ok := uri["exact"].(string); ok && exact != "" {
				path := normalizePath(exact)
				klog.V(2).Infof("Discovered base path %q (exact) from VirtualService %s/%s", path, vsNamespace, vsName)
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no URI prefix found in VirtualService %s/%s", vsNamespace, vsName)
}

// DiscoverBasePathFromVirtualServiceLabel finds a VirtualService in the given
// namespace that has a label matching goldilocks, and extracts its path prefix.
// This is useful when you don't know the exact VirtualService name.
func DiscoverBasePathFromVirtualServiceLabel(vsNamespace, labelSelector string) (string, error) {
	if vsNamespace == "" {
		return "", fmt.Errorf("virtual service namespace is required")
	}
	if labelSelector == "" {
		labelSelector = "app=goldilocks"
	}

	dynClient := GetDynamicInstance()
	vsList, err := dynClient.Client.Resource(virtualServiceGVR).Namespace(vsNamespace).List(
		context.TODO(), metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return "", fmt.Errorf("error listing VirtualServices in %s with selector %s: %w", vsNamespace, labelSelector, err)
	}

	if len(vsList.Items) == 0 {
		return "", fmt.Errorf("no VirtualService found in %s with selector %s", vsNamespace, labelSelector)
	}

	// Use the first matching VirtualService
	vs := vsList.Items[0]
	klog.V(2).Infof("Found VirtualService %s/%s via label selector %s", vsNamespace, vs.GetName(), labelSelector)
	return DiscoverBasePathFromVirtualService(vs.GetName(), vsNamespace)
}

// normalizePath ensures the path starts with / and ends with /
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	return p
}

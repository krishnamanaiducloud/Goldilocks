// Copyright 2019 FairwindsOps Inc
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

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/dashboard"
	"github.com/fairwindsops/goldilocks/pkg/kube"
)

var (
	serverPort   int
	showAllVPAs  bool
	basePath     string
	insightsHost string
	enableCost   bool
	vsName       string
	vsNamespace  string
	vsLabel      string
)

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 8080, "The port to serve the dashboard on.")
	dashboardCmd.PersistentFlags().StringVarP(&excludeContainers, "exclude-containers", "e", "", "Comma delimited list of containers to exclude from recommendations.")
	dashboardCmd.PersistentFlags().BoolVar(&onByDefault, "on-by-default", false, "Display every namespace that isn't explicitly excluded.")
	dashboardCmd.PersistentFlags().BoolVar(&showAllVPAs, "show-all", false, "Display every VPA, even if it isn't managed by Goldilocks")
	dashboardCmd.PersistentFlags().StringVar(&basePath, "base-path", "/", "Path on which the dashboard is served.")
	dashboardCmd.PersistentFlags().BoolVar(&enableCost, "enable-cost", true, "If set to false, the cost integration will be disabled on the dashboard.")
	dashboardCmd.PersistentFlags().StringVar(&insightsHost, "insights-host", "https://insights.fairwinds.com", "Insights host for retrieving optional cost data.")
	dashboardCmd.PersistentFlags().StringVar(&vsName, "vs-name", "", "Istio VirtualService name to discover the base path from. Overrides --base-path if set. [GOLDILOCKS_VS_NAME]")
	dashboardCmd.PersistentFlags().StringVar(&vsNamespace, "vs-namespace", "", "Namespace of the Istio VirtualService. Auto-detects pod namespace if not set. [GOLDILOCKS_VS_NAMESPACE]")
	dashboardCmd.PersistentFlags().StringVar(&vsLabel, "vs-label", "", "Label selector to auto-discover the VirtualService (e.g. app=goldilocks). [GOLDILOCKS_VS_LABEL]")

	// Environment variable overrides for VirtualService flags
	envOverrides := map[string]string{
		"GOLDILOCKS_VS_NAME":      "vs-name",
		"GOLDILOCKS_VS_NAMESPACE": "vs-namespace",
		"GOLDILOCKS_VS_LABEL":     "vs-label",
		"GOLDILOCKS_BASE_PATH":    "base-path",
	}
	for env, flag := range envOverrides {
		if value := os.Getenv(env); value != "" {
			if err := dashboardCmd.PersistentFlags().Set(flag, value); err != nil {
				klog.Errorf("Error setting flag %s from env %s: %v", flag, env, err)
			}
		}
	}
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Run the goldilocks dashboard that will show recommendations.",
	Long:  `Run the goldilocks dashboard that will show recommendations.`,
	Run: func(cmd *cobra.Command, args []string) {
		resolvedBasePath := resolveBasePath(basePath, vsName, vsNamespace, vsLabel)
		var validBasePath = validateBasePath(resolvedBasePath)
		router := dashboard.GetRouter(
			dashboard.OnPort(serverPort),
			dashboard.BasePath(validBasePath),
			dashboard.ExcludeContainers(sets.New[string](strings.Split(excludeContainers, ",")...)),
			dashboard.OnByDefault(onByDefault),
			dashboard.ShowAllVPAs(showAllVPAs),
			dashboard.InsightsHost(insightsHost),
			dashboard.EnableCost(enableCost),
			dashboard.WithVersion(version),
			dashboard.WithCommit(commit),
		)
		http.Handle("/", router)
		klog.Infof("Starting goldilocks dashboard server on port %d and basePath %v", serverPort, validBasePath)
		klog.Fatalf("%v", http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil))
	},
}

// resolveBasePath determines the dashboard base path. It tries to discover
// the path from an Istio VirtualService first (by name or label selector),
// falling back to the CLI --base-path flag.
func resolveBasePath(fallback, vsName, vsNamespace, vsLabel string) string {
	// Auto-detect namespace from Kubernetes downward API if not explicitly set
	if vsNamespace == "" && (vsName != "" || vsLabel != "") {
		vsNamespace = detectPodNamespace()
		if vsNamespace != "" {
			klog.V(2).Infof("Auto-detected pod namespace: %s", vsNamespace)
		}
	}

	// 1. Try explicit VirtualService name lookup
	if vsName != "" && vsNamespace != "" {
		path, err := kube.DiscoverBasePathFromVirtualService(vsName, vsNamespace)
		if err != nil {
			klog.Warningf("Could not discover base path from VirtualService %s/%s: %v — falling back to --base-path", vsNamespace, vsName, err)
		} else {
			klog.Infof("Discovered base path %q from VirtualService %s/%s", path, vsNamespace, vsName)
			return path
		}
	}

	// 2. Try label-based VirtualService discovery
	if vsLabel != "" && vsNamespace != "" {
		path, err := kube.DiscoverBasePathFromVirtualServiceLabel(vsNamespace, vsLabel)
		if err != nil {
			klog.Warningf("Could not discover base path from VirtualService label %q in %s: %v — falling back to --base-path", vsLabel, vsNamespace, err)
		} else {
			klog.Infof("Discovered base path %q from VirtualService (label=%s) in %s", path, vsLabel, vsNamespace)
			return path
		}
	}

	// 3. Fallback to CLI flag
	return fallback
}

// detectPodNamespace reads the namespace from the Kubernetes downward API
// service account mount, which is available inside any pod.
func detectPodNamespace() string {
	// Standard location for the pod's namespace in Kubernetes
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		klog.V(3).Infof("Could not auto-detect pod namespace: %v", err)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func validateBasePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	return path
}

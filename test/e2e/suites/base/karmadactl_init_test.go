/*
Copyright 2022 The Karmada Authors.

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

package base

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/karmada-io/karmada/pkg/util/names"
	"github.com/karmada-io/karmada/test/e2e/framework"
	"github.com/karmada-io/karmada/test/helper"
)

const (
	KarmadactlInitTimeOut = time.Minute * 10
)

// ComponentType defines the type of Kubernetes resource
type ComponentType string

const (
	ComponentTypeDeployment  ComponentType = "deployment"
	ComponentTypeStatefulSet ComponentType = "statefulset"
)

// ComponentConfig holds configuration for a Karmada component
type ComponentConfig struct {
	Name      string
	Type      ComponentType
	Flag      string
	ExtraArgs []string
	Selector  string
}

// KarmadaComponents holds all component configurations
type KarmadaComponents struct {
	LocalEtcd                  ComponentConfig
	KarmadaAPIServer           ComponentConfig
	KarmadaAggregatedAPIServer ComponentConfig
	KubeControllerManager      ComponentConfig
	KarmadaControllerManager   ComponentConfig
	KarmadaScheduler           ComponentConfig
	KarmadaWebhook             ComponentConfig
}

// NewKarmadaComponents creates a new instance of KarmadaComponents with default configurations
func NewKarmadaComponents() *KarmadaComponents {
	return &KarmadaComponents{
		LocalEtcd: ComponentConfig{
			Name:      "etcd",
			Type:      ComponentTypeStatefulSet,
			Flag:      "--etcd-extra-args",
			ExtraArgs: []string{"--snapshot-count=5000", "--heartbeat-interval=100"},
			Selector:  "app=etcd",
		},
		KarmadaAPIServer: ComponentConfig{
			Name:      "karmada-apiserver",
			Type:      ComponentTypeDeployment,
			Flag:      "--karmada-apiserver-extra-args",
			ExtraArgs: []string{"--tls-min-version=VersionTLS12", "--audit-log-path=-"},
			Selector:  "app=karmada-apiserver",
		},
		KarmadaAggregatedAPIServer: ComponentConfig{
			Name: "karmada-aggregated-apiserver",
			Type: ComponentTypeDeployment,
			Flag: "--karmada-aggregated-apiserver-extra-args",
			ExtraArgs: []string{
				"--tls-min-version=VersionTLS12",
				"--audit-log-maxbackup=10",
				"--v=2",
				"--enable-pprof",
			},
			Selector: "app=karmada-aggregated-apiserver",
		},
		KubeControllerManager: ComponentConfig{
			Name: "kube-controller-manager",
			Type: ComponentTypeDeployment,
			Flag: "--kube-controller-manager-extra-args",
			ExtraArgs: []string{
				"--v=2",
				"--node-monitor-grace-period=50s",
				"--node-monitor-period=5s",
			},
			Selector: "app=kube-controller-manager",
		},
		KarmadaControllerManager: ComponentConfig{
			Name: "karmada-controller-manager",
			Type: ComponentTypeDeployment,
			Flag: "--karmada-controller-manager-extra-args",
			ExtraArgs: []string{
				"--v=2",
				"--enable-pprof",
				"--skipped-propagating-namespaces=kube-system,default,my-ns",
				"--vmodule=scheduler*=3,controller*=2",
			},
			Selector: "app=karmada-controller-manager",
		},
		KarmadaScheduler: ComponentConfig{
			Name: "karmada-scheduler",
			Type: ComponentTypeDeployment,
			Flag: "--karmada-scheduler-extra-args",
			ExtraArgs: []string{
				"--v=2",
				"--enable-pprof",
				"--scheduler-name=test-scheduler",
			},
			Selector: "app=karmada-scheduler",
		},
		KarmadaWebhook: ComponentConfig{
			Name:      "karmada-webhook",
			Type:      ComponentTypeDeployment,
			Flag:      "--karmada-webhook-extra-args",
			ExtraArgs: []string{"--v=2", "--enable-pprof"},
			Selector:  "app=karmada-webhook",
		},
	}
}

// AllComponents returns all component configurations
func (kc *KarmadaComponents) AllComponents() []ComponentConfig {
	return []ComponentConfig{
		kc.LocalEtcd,
		kc.KarmadaAPIServer,
		kc.KarmadaAggregatedAPIServer,
		kc.KubeControllerManager,
		kc.KarmadaControllerManager,
		kc.KarmadaScheduler,
		kc.KarmadaWebhook,
	}
}

// GetAllExtraArgs returns all extra arguments for karmadactl init command
func (kc *KarmadaComponents) GetAllExtraArgs() []string {
	var allArgs []string
	for _, comp := range kc.AllComponents() {
		allArgs = append(allArgs, wrapArgs(comp.Flag, comp.ExtraArgs...)...)
	}
	return allArgs
}

// KarmadaTestContext holds the test context
type KarmadaTestContext struct {
	MemberKubeconfig string
	Components       *KarmadaComponents
}

// NewKarmadaTestContext creates a new test context
func NewKarmadaTestContext() *KarmadaTestContext {
	memberKubeconfig := os.Getenv("MEMBER_KUBECONFIG") // Set as an environment variable.
	if memberKubeconfig == "" {
		memberKubeconfig = "/root/.kube/members.config"
		klog.Infof("The environment variable MEMBER_KUBECONFIG is not set, using the default path: %s", memberKubeconfig)
	} else {
		klog.Infof("Use the environment variable MEMBER_KUBECONFIG: %s", memberKubeconfig)
	}

	// Verify if the kubeconfig file exists.
	if _, err := os.Stat(memberKubeconfig); os.IsNotExist(err) {
		klog.Warningf("The kubeconfig file does not exist: %s", memberKubeconfig)
	} else {
		klog.Infof("The kubeconfig file exists: %s", memberKubeconfig)
	}

	return &KarmadaTestContext{
		MemberKubeconfig: memberKubeconfig,
		Components:       NewKarmadaComponents(),
	}
}

var _ = ginkgo.Describe("Karmadactl Init Testing", func() {
	var (
		testCtx       *KarmadaTestContext
		member1       string
		member1Client kubernetes.Interface
	)

	ginkgo.BeforeEach(func() {
		testCtx = NewKarmadaTestContext()

		// Verify whether all cluster contexts exist.
		availableContexts := testCtx.getAvailableContexts()
		klog.Infof("Available context: %v", availableContexts)

		member1 = framework.ClusterNames()[0]
		klog.Infof("Select member cluster: %s", member1)
		if !contains(availableContexts, member1) {
			ginkgo.Fail(fmt.Sprintf("The context for cluster %s does not exist in the kubeconfig.", member1))
		}

		klog.Infof("Successfully selected member cluster: %s", member1)

		member1Client = framework.GetClusterClient(member1)
		defaultConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
		defaultConfigFlags.Context = &karmadaContext
	})

	ginkgo.Context("Test Karmadactl init Custom Control Plane", func() {
		var namespace string

		ginkgo.BeforeEach(func() {
			namespace = fmt.Sprintf("karmadatest-%s", rand.String(RandomStrLength))
		})

		ginkgo.AfterEach(func() {
			testCtx.cleanup(member1Client, namespace)
		})

		ginkgo.It("Test Custom Control Plane —— All Component", func() {
			// step1 Create a namespace in the member1 cluster.
			ginkgo.By(fmt.Sprintf("Creating namespace %s in member cluster %s", member1, namespace), func() {
				namespaceObj := helper.NewNamespace(namespace)
				framework.CreateNamespace(member1Client, namespaceObj)
			})

			// step2 Execute command
			testCtx.execKarmadactlInit(member1, namespace, testCtx.Components.GetAllExtraArgs()...)

			// step3 Check startup parameters.
			testCtx.checkAllComponentStatus(member1Client, namespace)

			// step4 Waiting for components to be ready.
			testCtx.waitForAllComponents(member1Client, namespace)
		})
	})
})

// cleanup handles the test cleanup
func (ctx *KarmadaTestContext) cleanup(client kubernetes.Interface, namespace string) {
	namespaceGVK := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	}

	cppName := names.GeneratePolicyName("", namespace, namespaceGVK.String())

	framework.RemoveNamespace(client, namespace)
	framework.RemoveClusterPropagationPolicy(karmadaClient, cppName)
}

// execKarmadactlInit Execute the command karmadactl init.
func (ctx *KarmadaTestContext) execKarmadactlInit(clusterName, namespace string, args ...string) {
	ginkgo.By("Execute the command karmadactl init.", func() {
		// Switch the specified cluster context
		contextManager := NewContextManager(ctx.MemberKubeconfig)
		defer contextManager.Restore()

		contextManager.SwitchTo(clusterName)

		baseArgs := []string{"init", "--namespace", namespace}
		allArgs := append(baseArgs, args...)

		cmd := framework.NewKarmadactlCommand(
			ctx.MemberKubeconfig,
			"",
			karmadactlPath,
			"",
			KarmadactlInitTimeOut,
			allArgs...,
		)
		_, err := cmd.ExecOrDie()
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

// checkAllComponentStatus Check the status of all components.
func (ctx *KarmadaTestContext) checkAllComponentStatus(client kubernetes.Interface, namespace string) {
	componentCount := len(ctx.Components.AllComponents())
	klog.Infof("Total number of components to be checked: %d", componentCount)

	for i, comp := range ctx.Components.AllComponents() {
		klog.Infof("Check component progress: %d/%d - %s", i+1, componentCount, comp.Name)
		ctx.checkComponentStatus(client, namespace, comp)
	}
	klog.Infof("All component status checks are complete.")
}

// checkComponentStatus Check the status of a single component.
func (ctx *KarmadaTestContext) checkComponentStatus(client kubernetes.Interface, namespace string, comp ComponentConfig) {
	ginkgo.By(fmt.Sprintf("Detecting %s: %s launch parameters", comp.Type, comp.Name), func() {
		gomega.Eventually(func() bool {
			if len(comp.ExtraArgs) == 0 {
				klog.Infof("Component %s has no additional parameters to check.", comp.Name)
				return true
			}

			commands, err := ctx.getComponentCommands(client, namespace, comp)
			if err != nil {
				return false
			}

			return ctx.validateExtraArgs(comp.Name, commands, comp.ExtraArgs)
		}, pollTimeout, pollInterval).Should(gomega.Equal(true))
	})
}

// getComponentCommands retrieves the command arguments for a component
func (ctx *KarmadaTestContext) getComponentCommands(client kubernetes.Interface, namespace string, comp ComponentConfig) ([]string, error) {
	// Get the component object based on the component type.
	switch comp.Type {
	case ComponentTypeDeployment:
		deployment, err := client.AppsV1().Deployments(namespace).
			Get(context.TODO(), comp.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if len(deployment.Spec.Template.Spec.Containers) == 0 {
			return nil, fmt.Errorf("no containers found in deployment %s", comp.Name)
		}

		return deployment.Spec.Template.Spec.Containers[0].Command, nil

	case ComponentTypeStatefulSet:
		statefulset, err := client.AppsV1().StatefulSets(namespace).
			Get(context.TODO(), comp.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if len(statefulset.Spec.Template.Spec.Containers) == 0 {
			return nil, fmt.Errorf("no containers found in statefulset %s", comp.Name)
		}

		return statefulset.Spec.Template.Spec.Containers[0].Command, nil

	default:
		return nil, fmt.Errorf("unsupported component type: %s", comp.Type)
	}
}

// validateExtraArgs validates that all expected extra arguments are present
func (ctx *KarmadaTestContext) validateExtraArgs(component string, commands, expectedArgs []string) bool {
	flags := make(map[string]bool, len(expectedArgs))
	for _, arg := range expectedArgs {
		flags[arg] = false
	}

	for _, cmd := range commands {
		if _, exists := flags[cmd]; exists {
			flags[cmd] = true
		}
	}

	for _, found := range flags {
		if !found {
			return false
		}
	}
	klog.Infof("The startup parameters for %s are normal.", component)
	return true
}

// waitForAllComponents Wait for all components to be ready.
func (ctx *KarmadaTestContext) waitForAllComponents(client kubernetes.Interface, namespace string) {
	componentCount := len(ctx.Components.AllComponents())
	klog.Infof("Total number of components to wait for: %d", componentCount)
	for i, comp := range ctx.Components.AllComponents() {
		klog.Infof("Waiting for component progress: %d/%d - %s", i+1, componentCount, comp.Name)
		ctx.waitForComponent(client, namespace, comp)
	}
	klog.Infof("All components are ready.")
}

// waitForComponent Waiting for a single component to be ready.
func (ctx *KarmadaTestContext) waitForComponent(client kubernetes.Interface, namespace string, comp ComponentConfig) {
	ginkgo.By(fmt.Sprintf("Waiting for %s %s to be ready", comp.Type, comp.Name), func() {
		gomega.Eventually(func() bool {
			// Check resource readiness
			if !ctx.isResourceReady(client, namespace, comp) {
				return false
			}
			klog.Infof("Component %s Resource ready", comp.Name)

			// Check pods readiness
			if ctx.arePodsReady(client, namespace, comp.Selector) {
				klog.Infof("Component %s pods are ready.", comp.Name)
				return true
			}
			return false
		}, pollTimeout, pollInterval).Should(gomega.Equal(true))
	})
}

// isResourceReady checks if the Kubernetes resource (Deployment/StatefulSet) is ready
func (ctx *KarmadaTestContext) isResourceReady(client kubernetes.Interface, namespace string, comp ComponentConfig) bool {
	switch comp.Type {
	case ComponentTypeDeployment:
		deployment, err := client.AppsV1().Deployments(namespace).
			Get(context.TODO(), comp.Name, metav1.GetOptions{})
		if err != nil {
			klog.Infof("Deployment Component %s Not Found, err %v", comp.Name, err)
			return false
		}
		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas

	case ComponentTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(namespace).
			Get(context.TODO(), comp.Name, metav1.GetOptions{})
		if err != nil {
			klog.Infof("StatefulSet Component %s Not Found, err %v", comp.Name, err)
			return false
		}
		return statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas

	default:
		return false
	}
}

// arePodsReady checks if all pods matching the selector are running
func (ctx *KarmadaTestContext) arePodsReady(client kubernetes.Interface, namespace, selector string) bool {
	pods, err := client.CoreV1().Pods(namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil || len(pods.Items) == 0 {
		return false
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return false
		}
	}
	return true
}

// ContextManager handles kubectl context operations
type ContextManager struct {
	kubeconfig      string
	originalContext string
}

// NewContextManager creates a new context manager
func NewContextManager(kubeconfig string) *ContextManager {
	return &ContextManager{
		kubeconfig:      kubeconfig,
		originalContext: getCurrentContext(kubeconfig),
	}
}

// SwitchTo switches to the specified context
func (cm *ContextManager) SwitchTo(clusterName string) {
	ginkgo.By(fmt.Sprintf("Switching to the context of cluster %s", clusterName), func() {
		if cm.originalContext == clusterName {
			klog.Infof("The target context is the same as the current context, no need to switch. %s", clusterName)
			return
		}

		// #nosec G204
		cmd := exec.Command("kubectl", "config", "use-context", clusterName,
			"--kubeconfig="+cm.kubeconfig)

		output, err := cmd.Output()
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to switch context")
		klog.Infof("switchToClusterContext success, output: %s", string(output))

		cm.verifyContextSwitch(clusterName)
	})
}

// verifyContextSwitch verifies that the context switch was successful
func (cm *ContextManager) verifyContextSwitch(expectedContext string) {
	actualContext := getCurrentContext(cm.kubeconfig)
	gomega.Expect(actualContext).To(gomega.Equal(expectedContext),
		fmt.Sprintf("Failed to switch cluster context, expected %s but got %s.", expectedContext, actualContext))
}

// Restore restores the original context
func (cm *ContextManager) Restore() {
	if cm.originalContext == "" {
		klog.Warning("The original context is empty and cannot be restored.")
		return
	}

	if cm.originalContext == "karmada-host" || cm.originalContext == "karmada-apiserver" {
		klog.Infof("No need to restore as the context in the kubeconfig has not been modified.")
		return
	}

	currentContext := getCurrentContext(cm.kubeconfig)
	if currentContext == cm.originalContext {
		klog.Infof("The current context is already the original context, no need to restore it. %s", cm.originalContext)
		return
	}

	ginkgo.By(fmt.Sprintf("Restore to the original context %s", cm.originalContext), func() {
		// #nosec G204
		cmd := exec.Command("kubectl", "config", "use-context", cm.originalContext,
			"--kubeconfig="+cm.kubeconfig)
		output, err := cmd.Output()
		if err != nil {
			klog.Errorf("restoreContext error: %v", err)
		} else {
			klog.Infof("restoreContext success, output: %s", string(output))
		}
	})
}

// getCurrentContext Get the current context.
func getCurrentContext(kubeconfig string) string {
	// #nosec G204
	cmd := exec.Command("kubectl", "config", "current-context",
		"--kubeconfig="+kubeconfig)
	output, err := cmd.Output()
	if err != nil {
		klog.Errorf("getCurrentContext error: %v", err)
		return ""
	}
	currentContext := strings.TrimSpace(string(output))
	klog.Infof("currentContext: %s", currentContext)
	return currentContext
}

// getAvailableContexts Get all available contexts.
func (ctx *KarmadaTestContext) getAvailableContexts() []string {
	// #nosec G204
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name",
		"--kubeconfig="+ctx.MemberKubeconfig)

	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	contexts := strings.Split(strings.TrimSpace(string(output)), "\n")
	return contexts
}

// Tool functions
// contains Check if the string slice contains the specified string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// wrapArgs Encapsulation parameters
func wrapArgs(flag string, args ...string) []string {
	if len(args) == 0 {
		klog.Infof("The parameter for the flag %s is empty.", flag)
		return []string{}
	}

	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = flag + "=" + arg
	}
	return out
}

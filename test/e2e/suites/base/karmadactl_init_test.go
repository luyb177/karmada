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
	"os/exec"
	"strings"
	"time"

	"github.com/karmada-io/karmada/pkg/util/names"
	"github.com/karmada-io/karmada/test/e2e/framework"
	"github.com/karmada-io/karmada/test/helper"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

//
//// TempCluster 临时集群结构体
//type TempCluster struct {
//	Provider         *cluster.Provider
//	Name             string
//	Context          string
//	KubeConfig       string
//	Client           kubernetes.Interface
//	KarmadaNamespace string
//}
//
//func NewTempCluster() *TempCluster {
//	tc := &TempCluster{
//		Provider:         cluster.NewProvider(),
//		Name:             fmt.Sprintf("karmadactltest-%s", rand.String(RandomStrLength)),
//		KarmadaNamespace: fmt.Sprintf("karmadactltest-%s", rand.String(RandomStrLength)),
//	}
//	tc.Context = fmt.Sprintf("kind-%s", tc.Name)
//
//	// 使用临时的 kubeconfig 文件
//	tempDir := os.TempDir()
//	tc.KubeConfig = fmt.Sprintf("%s/%s.config", tempDir, tc.Name)
//
//	return tc
//}
//
//// SetUp 设置临时集群
//func (tc *TempCluster) SetUp() {
//	ginkgo.By(fmt.Sprintf("创建临时集群 %s", tc.Name), func() {
//		err := createCluster(tc.Name, tc.KubeConfig, fmt.Sprintf("%s-control-plane", tc.Name), tc.Context)
//		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
//	})
//
//	ginkgo.By("初始化集群客户端", func() {
//		config, err := framework.LoadRESTClientConfig(tc.KubeConfig, tc.Context)
//		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
//
//		tc.Client, err = kubernetes.NewForConfig(config)
//		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
//	})
//
//	ginkgo.By(fmt.Sprintf("创建命名空间 %s", tc.KarmadaNamespace), func() {
//		namespaceObj := helper.NewNamespace(tc.KarmadaNamespace)
//		framework.CreateNamespace(tc.Client, namespaceObj)
//	})
//}
//
//// CleanUp 清理临时集群
//func (tc *TempCluster) CleanUp() {
//	ginkgo.By(fmt.Sprintf("清理临时集群 %s", tc.Name), func() {
//		if tc.Provider != nil && tc.Name != "" {
//			err := deleteCluster(tc.Name, tc.KubeConfig)
//			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
//		}
//
//		// 删除临时文件
//		if tc.KubeConfig != "" {
//			os.Remove(tc.KubeConfig)
//		}
//	})
//}
////
////// ExecKarmadactlInit 执行 karmadactl init 命令
////func (tc *TempCluster) ExecKarmadactlInit(args ...string) {
////	ginkgo.By("执行 karmadactl init 命令", func() {
////		baseArgs := []string{
////			"init",
////			"--namespace", tc.KarmadaNamespace,
////		}
////		allArs := append(baseArgs, args...)
////
////		cmd := framework.NewKarmadactlCommand(
////			tc.KubeConfig,
////			"",
////			karmadactlPath,
////			"",
////			KarmadactlInitTimeOut,
////			allArs...,
////		)
////		_, err := cmd.ExecOrDie()
////		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
////	})
////}
////
////// WaitForComponent 等待 Deployment StatefulSet Pod 组件就绪
////func (tc *TempCluster) WaitForComponent(componentType string, componentName string, selector string) {
////	ginkgo.By(fmt.Sprintf("等待 %s %s 就绪", componentType, componentName), func() {
////		gomega.Eventually(func() bool {
////			switch componentType {
////			case "deployment":
////				deployment, err := tc.Client.AppsV1().Deployments(tc.KarmadaNamespace).
////					Get(context.TODO(), componentName, metav1.GetOptions{})
////				if err != nil {
////					ginkgo.GinkgoLogr.Info("Deployment Component Not Found", "component", componentName)
////					return false
////				}
////
////				// 确保副本与期望副本数一致
////				if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
////					return false
////				}
////
////			case "statefulset":
////				statefulSet, err := tc.Client.AppsV1().StatefulSets(tc.KarmadaNamespace).
////					Get(context.TODO(), componentName, metav1.GetOptions{})
////				if err != nil {
////					ginkgo.GinkgoLogr.Info("StatefulSet Component Not Found", "component", componentName)
////					return false
////				}
////				if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
////					return false
////				}
////
////			default:
////				return false
////			}
////
////			// 检查 Pods
////			pods, err := tc.Client.CoreV1().Pods(tc.KarmadaNamespace).
////				List(context.TODO(), metav1.ListOptions{
////					LabelSelector: selector,
////				})
////			if err != nil || len(pods.Items) == 0 {
////				return false
////			}
////
////			for _, pod := range pods.Items {
////				if pod.Status.Phase != corev1.PodRunning {
////					return false
////				}
////			}
////			return true
////		}, pollTimeout*2, pollInterval).Should(gomega.Equal(true))
////	})
////}
//
//var _ = ginkgo.Describe("Karmadactl Init Testing", func() {
//	var tempCluster *TempCluster
//
//	ginkgo.Context("Test Karmadactl Init 自定义控制面板", func() {
//		ginkgo.BeforeEach(func() {
//			// 创建和设置临时集群
//			tempCluster = NewTempCluster()
//			tempCluster.SetUp()
//		})
//
//		// 暂时不清理，方便测试失败查看为什么
//		//ginkgo.AfterEach(func() {
//		//	if tempCluster != nil {
//		//		tempCluster.CleanUp()
//		//	}
//		//})
//
//		ginkgo.It("Test 自定义控制面板组件 - karmada-api-server", func() {
//
//			// step1 执行命令
//			//tempCluster.ExecKarmadactlInit(
//			//	"--karmada-apiserver-extra-args", "--tls-min-version=VersionTLS12",
//			//	"--karmada-apiserver-extra-args", "--audit-log-path=-",
//			//)
//
//			// step2 检查启动参数
//			ginkgo.By("检查 Deployment 是否有对应参数", func() {
//				gomega.Eventually(func() bool {
//					deployment, err := tempCluster.Client.AppsV1().Deployments(tempCluster.KarmadaNamespace).
//						Get(context.TODO(), "karmada-apiserver", metav1.GetOptions{})
//					if err != nil {
//						return false
//					}
//
//					if len(deployment.Spec.Template.Spec.Containers) == 0 {
//						return false
//					}
//
//					commands := deployment.Spec.Template.Spec.Containers[0].Command
//
//					hasTLSVersion := false
//					hasAuditPath := false
//
//					for _, arg := range commands {
//						if strings.Contains(arg, "--tls-min-version=VersionTLS12") {
//							hasTLSVersion = true
//						}
//						if strings.Contains(arg, "--audit-log-path=-") {
//							hasAuditPath = true
//						}
//					}
//
//					return hasTLSVersion && hasAuditPath
//				}, pollTimeout, pollInterval).Should(gomega.Equal(true))
//
//				// step3 等待组件就绪
//				//tempCluster.WaitForComponent("deployment", "karmada-apiserver", "app=karmada-apiserver")
//
//			})
//		})
//	})
//})

const (
	KarmadactlInitTimeOut = time.Minute * 10
)

var (
	memberKubeconfig = "/root/.kube/members.config" // todo 先写死，后期看怎么改成环境变量
	hostKubeconfig   = "/root/.kube/karmada.config"
)
var _ = ginkgo.Describe("Karmadactl Init Testing", func() {
	var member1 string
	var member1Client kubernetes.Interface

	ginkgo.BeforeEach(func() {
		// 验证所有集群上下文是否存在
		availableContexts := getAvailableContexts()
		klog.Infof("可用上下文: %v", availableContexts)

		member1 = framework.ClusterNames()[0]
		if !contains(availableContexts, member1) {
			ginkgo.Fail(fmt.Sprintf("集群 %s 的上下文不存在于 kubeconfig 中", member1))
		}
		klog.Infof("使用集群: %s", member1)

		member1Client = framework.GetClusterClient(member1)
		defaultConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
		defaultConfigFlags.Context = &karmadaContext
	})

	ginkgo.Context("Test Karmadactl init 自定义控制平面", func() {
		var namespace string

		ginkgo.BeforeEach(func() {
			namespace = fmt.Sprintf("karmadatest-%s", rand.String(RandomStrLength))
		})

		ginkgo.AfterEach(func() {
			namespaceGVK := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Namespace",
			}

			cppName := names.GeneratePolicyName("", namespace, namespaceGVK.String())

			framework.RemoveNamespace(member1Client, namespace)
			framework.RemoveClusterPropagationPolicy(karmadaClient, cppName)
		})

		ginkgo.It("Test 自定义控制平面 - karmada-api-server", func() {
			// step1 在成员集群1创建命名空间
			ginkgo.By(fmt.Sprintf("在成员集群%s创建命名空间%s", member1, namespace), func() {
				namespaceObj := helper.NewNamespace(namespace)
				framework.CreateNamespace(member1Client, namespaceObj)
			})

			// step2 执行命令
			ExecKarmadactlInit(member1, namespace,
				`--karmada-apiserver-extra-args="--tls-min-version=VersionTLS12"`,
				`--karmada-apiserver-extra-args="--audit-log-path=-"`,
			)

			// step3 检查启动参数
			ginkgo.By("检查 Deployment 是否有对应参数", func() {
				gomega.Eventually(func() bool {
					deployment, err := member1Client.AppsV1().Deployments(namespace).
						Get(context.TODO(), "karmada-apiserver", metav1.GetOptions{})
					if err != nil {
						return false
					}

					if len(deployment.Spec.Template.Spec.Containers) == 0 {
						return false
					}

					commands := deployment.Spec.Template.Spec.Containers[0].Command

					hasTLSVersion := false
					hasAuditPath := false

					for _, arg := range commands {
						if strings.Compare(arg, "--tls-min-version=VersionTLS12") == 0 {
							klog.Infof("存在参数 --tls-min-version=VersionTLS12")
							hasTLSVersion = true
						}
						if strings.Compare(arg, "--audit-log-path=-") == 0 {
							klog.Infof("存在参数 --audit-log-path=-")
							hasAuditPath = true
						}
					}

					return hasTLSVersion && hasAuditPath
				}, pollTimeout, pollInterval).Should(gomega.Equal(true))

				// step4 等待组件就绪
				WaitForComponent(member1Client, namespace, "deployment", "karmada-apiserver", "app=karmada-apiserver")

			})

		})
	})

})

// ExecKarmadactlInit 执行 karmadactl init 命令
func ExecKarmadactlInit(clusterName, namespace string, args ...string) {
	ginkgo.By("执行 karmadactl init 命令", func() {
		// 保存当前上下文
		currentContext := getCurrentContext()
		defer restoreContext(currentContext) // 执行完成后恢复上下文

		// 切换到目标集群上下文
		switchToClusterContext(clusterName)

		baseArgs := []string{
			"init",
			"--namespace", namespace,
		}
		allArs := append(baseArgs, args...)
		cmd := framework.NewKarmadactlCommand(
			memberKubeconfig,
			"",
			karmadactlPath,
			"",
			KarmadactlInitTimeOut,
			allArs...,
		)
		_, err := cmd.ExecOrDie()
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
}

// WaitForComponent 等待 Deployment StatefulSet Pod 组件就绪
func WaitForComponent(member1Client kubernetes.Interface, namespace string, componentType string, componentName string, selector string) {
	ginkgo.By(fmt.Sprintf("等待 %s %s 就绪", componentType, componentName), func() {
		gomega.Eventually(func() bool {
			switch componentType {
			case "deployment":
				deployment, err := member1Client.AppsV1().Deployments(namespace).
					Get(context.TODO(), componentName, metav1.GetOptions{})
				if err != nil {
					//ginkgo.GinkgoLogr.Info("Deployment Component Not Found", "component", componentName)
					klog.Infof("Deployment Component %s Not Found, err %v", componentName, err)
					return false
				}

				// 确保副本与期望副本数一致
				if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
					return false
				}

			case "statefulset":
				statefulSet, err := member1Client.AppsV1().StatefulSets(namespace).
					Get(context.TODO(), componentName, metav1.GetOptions{})
				if err != nil {
					//ginkgo.GinkgoLogr.Info("StatefulSet Component Not Found", "component", componentName)
					klog.Infof("StatefulSet Component %s Not Found,err %v ", componentName, err)
					return false
				}
				if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
					return false
				}
			}

			// 检查 Pods
			pods, err := member1Client.CoreV1().Pods(namespace).
				List(context.TODO(), metav1.ListOptions{
					LabelSelector: selector,
				})
			if err != nil || len(pods.Items) == 0 {
				return false
			}

			for _, pod := range pods.Items {
				if pod.Status.Phase != corev1.PodRunning {
					return false
				}
			}
			return true
		}, pollTimeout*2, pollInterval).Should(gomega.Equal(true))
	})
}

// getCurrentContext 获取当前成员上下文
func getCurrentContext() string {
	cmd := exec.Command("kubectl", "config", "current-context",
		"--kubeconfig="+memberKubeconfig)
	output, err := cmd.Output()
	if err != nil {
		klog.Errorf("getCurrentContext error: %v", err)
		return ""
	}
	currentContext := strings.TrimSpace(string(output))
	klog.Infof("currentContext: %s", currentContext)
	return currentContext
}

// switchToClusterContext 切换到成员集群的context
func switchToClusterContext(clusterName string) {
	ginkgo.By(fmt.Sprintf("切换到集群%s的context", clusterName), func() {
		cmd := exec.Command("kubectl", "config", "use-context", clusterName,
			"--kubeconfig="+memberKubeconfig)

		output, err := cmd.Output()
		if err != nil {
			klog.Errorf("switchToClusterContext error: %v", err)
			return
		}
		klog.Infof("switchToClusterContext success, output: %s", string(output))

		// 验证切换是否成功
		verifyContextSwitch(clusterName)
	})
}

// verifyContextSwitch 验证切换是否成功
func verifyContextSwitch(expectedContext string) {
	actualContext := getCurrentContext()
	gomega.Expect(actualContext).To(gomega.Equal(expectedContext), fmt.Sprintf("切换集群context失败，期望 %s 实际 %s", expectedContext, actualContext))
}

// restoreContext 恢复到原始的context
func restoreContext(originalContext string) {
	if originalContext == "" {
		klog.Warning("原始context为空，无法恢复")
		return
	}

	ginkgo.By(fmt.Sprintf("恢复到原始context %s", originalContext), func() {

		if originalContext == "karmada-host" || originalContext == "karmada-apiserver" {
			// 因为 switchToClusterContext 修改上下文也只是修改 memberKubeconfig 里面的，所以这里 默认的 kubeconfig的 上下文是没有切换的
			klog.Infof("因未修改该 kubeconfig 中的 context，无需恢复")
			return
		}

		cmd := exec.Command("kubectl", "config", "use-context", originalContext,
			"--kubeconfig="+memberKubeconfig)
		output, err := cmd.Output()
		if err != nil {
			klog.Errorf("restoreContext error: %v", err)
		} else {
			klog.Infof("restoreContext success, output: %s", string(output))
		}

	})
}

// getAvailableContexts 获取成员集群所有可用的上下文
func getAvailableContexts() []string {
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name",
		"--kubeconfig="+memberKubeconfig)

	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	contexts := strings.Split(strings.TrimSpace(string(output)), "\n")
	return contexts
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

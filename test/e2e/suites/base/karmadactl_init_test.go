/*
Copyright 2020 The Karmada Authors.

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
	"github.com/karmada-io/karmada/test/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"os"
	"sigs.k8s.io/kind/pkg/cluster"
	"strings"
	"time"
)

const (
	KarmadactlInitTimeOut = time.Minute * 10
)

// TempCluster 临时集群结构体
type TempCluster struct {
	Provider         *cluster.Provider
	Name             string
	Context          string
	KubeConfig       string
	Client           kubernetes.Interface
	KarmadaNamespace string
}

func NewTempCluster() *TempCluster {
	tc := &TempCluster{
		Provider:         cluster.NewProvider(),
		Name:             fmt.Sprintf("karmadactl-init-test-%s", rand.String(RandomStrLength)),
		KarmadaNamespace: fmt.Sprintf("karmadactl-init-test-%s", rand.String(RandomStrLength)),
	}
	tc.Context = fmt.Sprintf("kind-%s", tc.Name)

	// 使用临时的 kubeconfig 文件
	tempDir := os.TempDir()
	tc.KubeConfig = fmt.Sprintf("%s/%s-kubeconfig", tempDir, tc.Name)

	return tc
}

// SetUp 设置临时集群
func (tc *TempCluster) SetUp() {
	ginkgo.By(fmt.Sprintf("创建临时集群 %s", tc.Name), func() {
		err := createCluster(tc.Name, tc.KubeConfig, fmt.Sprintf("%s-control-plane", tc.Name), tc.Context)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.By("初始化集群客户端", func() {
		config, err := framework.LoadRESTClientConfig(tc.KubeConfig, tc.Context)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		tc.Client, err = kubernetes.NewForConfig(config)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
}

// CleanUp 清理临时集群
func (tc *TempCluster) CleanUp() {
	ginkgo.By(fmt.Sprintf("清理临时集群 %s", tc.Name), func() {
		if tc.Provider != nil && tc.Name != "" {
			err := deleteCluster(tc.Name, tc.KubeConfig)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}

		// 删除临时文件
		if tc.KubeConfig != "" {
			os.Remove(tc.KubeConfig)
		}
	})
}

// ExecKarmadactlInit 执行 karmadactl init 命令
func (tc *TempCluster) ExecKarmadactlInt(args ...string) {
	ginkgo.By("执行 karmadactl init 命令", func() {
		baseArgs := []string{
			"init",
			"--namespace", tc.KarmadaNamespace,
		}
		allArs := append(baseArgs, args...)

		cmd := framework.NewKarmadactlCommand(
			tc.KubeConfig,
			tc.Context,
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
func (tc *TempCluster) WaitForComponent(componentType string, componentName string, selector string) {
	ginkgo.By(fmt.Sprintf("等待 %s %s 就绪", componentType, componentName), func() {
		gomega.Eventually(func() bool {
			switch componentType {
			case "deployment":
				deployment, err := tc.Client.AppsV1().Deployments(tc.KarmadaNamespace).
					Get(context.TODO(), componentName, metav1.GetOptions{})
				if err != nil {
					ginkgo.GinkgoLogr.Info("Deployment Component Not Found", "component", componentName)
					return false
				}

				// 确保副本与期望副本数一致
				if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
					return false
				}

			case "statefulset":
				statefulSet, err := tc.Client.AppsV1().StatefulSets(tc.KarmadaNamespace).
					Get(context.TODO(), componentName, metav1.GetOptions{})
				if err != nil {
					ginkgo.GinkgoLogr.Info("StatefulSet Component Not Found", "component", componentName)
					return false
				}
				if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
					return false
				}

			default:
				return false
			}

			// 检查 Pods
			pods, err := tc.Client.CoreV1().Pods(tc.KarmadaNamespace).
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

var _ = ginkgo.Describe("Karmadactl Init Testing", func() {
	var tempCluster *TempCluster

	ginkgo.Context("Test Karmadactl Init 自定义控制面板", func() {
		ginkgo.BeforeEach(func() {
			// 创建和设置临时集群
			tempCluster = NewTempCluster()
			tempCluster.SetUp()
		})

		ginkgo.AfterEach(func() {
			if tempCluster != nil {
				tempCluster.CleanUp()
			}
		})

		ginkgo.It("Test 自定义控制面板组件 - karmada-api-server", func() {

			// step1 执行命令
			tempCluster.ExecKarmadactlInt(
				"--karmada-apiserver-extra-args", "--tls-min-version=VersionTLS12",
				"--karmada-apiserver-extra-args", "--audit-log-path=-",
			)

			// step2 检查启动参数
			ginkgo.By("检查 Deployment 是否有对应参数", func() {
				gomega.Eventually(func() bool {
					deployment, err := tempCluster.Client.AppsV1().Deployments(tempCluster.KarmadaNamespace).
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
						if strings.Contains(arg, "--tls-min-version=VersionTLS12") {
							hasTLSVersion = true
						}
						if strings.Contains(arg, "--audit-log-path=-") {
							hasAuditPath = true
						}
					}

					return hasTLSVersion && hasAuditPath
				}, pollTimeout, pollInterval).Should(gomega.Equal(true))

				// step3 等待组件就绪
				tempCluster.WaitForComponent("deployment", "karmada-apiserver", "app=karmada-apiserver")

			})
		})
	})
})

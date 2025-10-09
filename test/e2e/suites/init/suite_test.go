/*
Copyright 2025 The Karmada Authors.

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
package init

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/karmada-io/karmada/pkg/util"
	"github.com/karmada-io/karmada/test/e2e/framework"
	"github.com/karmada-io/karmada/test/helper"
)

const (
	// RandomStrLength represents the random string length to combine names.
	RandomStrLength = 5
	// KarmadaInstanceNamePrefix the prefix of the karmada instance name.
	KarmadaInstanceNamePrefix = "karmadatest-"
)

var (
	// pollInterval defines the interval time for a poll operation.
	pollInterval time.Duration
	// pollTimeout defines the time after which the poll operation times out.
	pollTimeout time.Duration
)

var (
	hostContext     string
	kubeconfig      string
	karmadactlPath  string
	restConfig      *rest.Config
	kubeClient      kubernetes.Interface
	testNamespace   string
	clusterProvider *cluster.Provider
	crdsPath        string
)

// image
var (
	karmadaAggregatedAPIServerImage string
	karmadaControllerManagerImage   string
	karmadaSchedulerImage           string
	karmadaWebhookImage             string
)

func init() {
	// usage ginkgo -- --poll-interval=5s --poll-timeout=5m
	// eg. ginkgo -v --race --trace --fail-fast -p --randomize-all ./test/e2e/ -- --poll-interval=5s --poll-timeout=5m
	flag.DurationVar(&pollInterval, "poll-interval", 5*time.Second, "poll-interval defines the interval time for a poll operation")
	flag.DurationVar(&pollTimeout, "poll-timeout", 300*time.Second, "poll-timeout defines the time which the poll operation times out")
	flag.StringVar(&hostContext, "host-context", "karmada-host", "Name of the host cluster context in control plane kubeconfig file.")
}

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E Init Suite")
}

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte { return nil }, func([]byte) {
	kubeconfig = os.Getenv("KUBECONFIG")
	gomega.Expect(kubeconfig).ShouldNot(gomega.BeEmpty())

	crdsPath = os.Getenv("CRDs_PATH")
	gomega.Expect(crdsPath).ShouldNot(gomega.BeEmpty())

	karmadaAggregatedAPIServerImage = os.Getenv("KARMADA_AGGREGATED_APISERVER_IMAGE")
	gomega.Expect(karmadaAggregatedAPIServerImage).ShouldNot(gomega.BeEmpty())

	karmadaControllerManagerImage = os.Getenv("KARMADA_CONTROLLER_MANAGER_IMAGE")
	gomega.Expect(karmadaControllerManagerImage).ShouldNot(gomega.BeEmpty())

	karmadaSchedulerImage = os.Getenv("KARMADA_SCHEDULER_IMAGE")
	gomega.Expect(karmadaSchedulerImage).ShouldNot(gomega.BeEmpty())

	karmadaWebhookImage = os.Getenv("KARMADA_WEBHOOK_IMAGE")
	gomega.Expect(karmadaWebhookImage).ShouldNot(gomega.BeEmpty())

	goPathCmd := exec.Command("go", "env", "GOPATH")
	goPath, err := goPathCmd.CombinedOutput()
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	formatGoPath := strings.Trim(string(goPath), "\n")
	karmadactlPath = formatGoPath + "/bin/karmadactl"
	gomega.Expect(karmadactlPath).ShouldNot(gomega.BeEmpty())

	clusterProvider = cluster.NewProvider()

	restConfig, err = framework.LoadRESTClientConfig(kubeconfig, hostContext)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	kubeClient, err = kubernetes.NewForConfig(restConfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	testNamespace = fmt.Sprintf("init-test-%s", rand.String(RandomStrLength))
	err = setupTestNamespace(testNamespace, kubeClient)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	framework.WaitNamespacePresentOnClusters(framework.ClusterNames(), testNamespace)
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	// cleanup all namespaces we created both in control plane and member clusters.
	// It will not return error even if there is no such namespace in there that may happen in case setup failed.
	if testNamespace != "" && kubeClient != nil {
		err := cleanupTestNamespace(testNamespace, kubeClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	}
}, func() {})

// setupTestNamespace will create a namespace in control plane and all member clusters, most of cases will run against it.
// The reason why we need a separated namespace is it will make it easier to cleanup resources deployed by the testing.
func setupTestNamespace(namespace string, kubeClient kubernetes.Interface) error {
	namespaceObj := helper.NewNamespace(namespace)
	_, err := util.CreateNamespace(kubeClient, namespaceObj)
	if err != nil {
		return err
	}
	return nil
}

// cleanupTestNamespace will remove the namespace we setup before for the whole testing.
func cleanupTestNamespace(namespace string, kubeClient kubernetes.Interface) error {
	err := util.DeleteNamespace(kubeClient, namespace)
	if err != nil {
		return err
	}
	return nil
}

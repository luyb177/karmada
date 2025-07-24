package cmdinit

// ComponentType 定义支持的组件类型
type ComponentType string

const (
	ComponentEtcd                       ComponentType = "etcd"              // statefulSet
	ComponentKarmadaAPIServer           ComponentType = "karmada-apiserver" // deployment
	ComponentKarmadaScheduler           ComponentType = "karmada-scheduler"
	ComponentKubeControllerManager      ComponentType = "karmada-kube-controller-manager"
	ComponentKarmadaControllerManager   ComponentType = "karmada-controller-manager"
	ComponentKarmadaWebhook             ComponentType = "karmada-webhook"
	ComponentKarmadaAggregatedAPIServer ComponentType = "karmada-aggregated-apiserver"
)

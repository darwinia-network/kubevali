package nodesvc

import (
	"context"
	"os"

	"github.com/darwinia-network/kubevali/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	annotationKey   = "kubeva.li/managed-by"
	annotationValue = "kubevali"
	portName        = "p2p"
	portValue       = 30333
)

func CreateOrUpdate(conf *config.Config) {
	if !conf.NodeService.Enabled {
		conf.Logger.Infof("Node service is disabled, skipped sync the service")
		return
	}

	podName := os.Getenv("HOSTNAME")
	if podName == "" {
		conf.Logger.Fatalf("Empty Pod name (environment variable $HOSTNAME)")
	}

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		conf.Logger.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		conf.Logger.Fatal(err)
	}

	ns, _, err := kubeconfig.Namespace()
	if err != nil {
		conf.Logger.Fatal(err)
	}

	pod, err := clientset.CoreV1().Pods(ns).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		conf.Logger.Fatal(err)
	}

	svcName := podName
	svc, err := clientset.CoreV1().Services(ns).Get(context.Background(), svcName, metav1.GetOptions{})
	svcExists := true
	if err != nil {
		if errors.IsNotFound(err) {
			svcExists = false
		} else {
			conf.Logger.Fatal(err)
		}
	}

	if svcExists && !conf.NodeService.ForceUpdate {
		if v, ok := svc.Labels[annotationKey]; !ok {
			conf.Logger.Infof("Service exists but not managed by kubevali, skipped")
			return
		} else if v != annotationValue {
			conf.Logger.Infof("Service exists but managed by %s, skipped", v)
			return
		}
	}

	svcLabelSelector := pod.Labels
	delete(svcLabelSelector, "controller-revision-hash")
	conf.Logger.Debugf("Generated service label selectors are %v", svcLabelSelector)

	if !svcExists {
		svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   podName,
				Labels: svcLabelSelector,
			},
		}
	}

	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[annotationKey] = annotationValue

	svc.Spec.Type = "NodePort"
	svc.Spec.Selector = svcLabelSelector

	svcPort := corev1.ServicePort{
		Name:       portName,
		Port:       portValue,
		NodePort:   int32(conf.NodeService.NodePort),
		TargetPort: intstr.FromInt(conf.NodeService.NodePort),
	}

	portExists := false
	for i, p := range svc.Spec.Ports {
		if p.Name == svcPort.Name {
			portExists = true
			svc.Spec.Ports[i] = svcPort
		}
	}
	if !portExists {
		svc.Spec.Ports = append(svc.Spec.Ports, svcPort)
	}

	if !svcExists {

		_, err = clientset.CoreV1().Services(ns).Create(context.Background(), svc, metav1.CreateOptions{})
		if err != nil {
			conf.Logger.Fatal(err)
		} else {
			conf.Logger.Infof("Created node service %s", svc.Name)
		}

	} else {

		_, err = clientset.CoreV1().Services(ns).Update(context.Background(), svc, metav1.UpdateOptions{})
		if err != nil {
			conf.Logger.Fatal(err)
		} else {
			conf.Logger.Infof("Updated node service %s", svc.Name)
		}

	}
}

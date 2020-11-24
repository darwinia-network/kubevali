package config

import (
	"fmt"
	"math/rand"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

const recursionMaxNums = 1000

// Copied and modified from https://github.com/helm/helm/blob/5b42157335d6e52551d246f233dd66972d5fda09/pkg/engine/engine.go#L112-L125
func initTemplateFuncMap(t *template.Template) {
	f := sprig.TxtFuncMap()
	includedNames := make(map[string]int)

	f["include"] = func(name string, data interface{}) (string, error) {
		var buf strings.Builder
		if v, ok := includedNames[name]; ok {
			if v > recursionMaxNums {
				return "", fmt.Errorf("unable to render template with a nested reference name: %s", name)
			}
			includedNames[name]++
		} else {
			includedNames[name] = 1
		}
		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--
		return buf.String(), err
	}

	f["getRandomNodeIP"] = getRandomNodeIP
	f["getNodeIPWithIndex"] = getNodeIPWithIndex

	t.Funcs(f)
}

func listNodesExternalIPs() ([]string, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	var externalIPs []string

	nodelist, _ := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	for _, n := range nodelist.Items {
		for _, a := range n.Status.Addresses {
			if a.Type == "ExternalIP" {
				externalIPs = append(externalIPs, a.Address)
			}
		}
	}

	if len(externalIPs) == 0 {
		return nil, fmt.Errorf("listNodesExternalIPs: No external node IP found")
	}

	return externalIPs, nil
}

func getRandomNodeIP() (string, error) {
	ips, err := listNodesExternalIPs()
	if err != nil {
		return "", err
	}

	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(ips))
	return ips[idx], nil
}

func getNodeIPWithIndex(idx int) (string, error) {
	ips, err := listNodesExternalIPs()
	if err != nil {
		return "", err
	}

	return ips[idx%len(ips)], nil
}

package util

import (
	"flag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func GetConfig(isOutOfCluster bool, kubeConfigPath string) (*rest.Config, error) {

	// creates the in-cluster config
	if isOutOfCluster == false {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		return config, nil
	}

	// create out-cluster config from specified kubeconfig file
	if kubeConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			panic(err.Error())
		}
		return config, nil
	}

	// creates the out-cluster config from HOME
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return config, nil
}

func Int32Ptr(i int32) *int32 { return &i }

func StringPtr(s string) *string { return &s }

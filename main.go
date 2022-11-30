package main

import (
	"flag"

	"github.com/avithe-great/kube-ctrl-service/src/controller"
)

func main() {
	//flag set with config file path
	kubeconfig := flag.String("kubeconfig", "conf/kube/config", "(optional) absolute path to the kubeconfig file")

	flag.Parse()
	controller.Start(kubeconfig)
}

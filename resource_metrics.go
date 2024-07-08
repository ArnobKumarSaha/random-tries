package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	rsmetrics "kmodules.xyz/resource-metrics"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ResourceMetrics() {
	config := getRESTConfig()
	kc, err := client.New(config, client.Options{Scheme: scm})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	var mongodbs dbapi.MongoDBList
	err = kc.List(context.TODO(), &mongodbs)
	if err != nil {
		klog.Error(err, "unable to list MongoDBs")
		os.Exit(1)
	}

	for _, m := range mongodbs.Items {
		calcRes(m)
	}
	fmt.Println("Done")
}

func calcRes(m dbapi.MongoDB) {
	klog.Infof("%s/%s", m.Namespace, m.Name)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&m)
	if err != nil {
		klog.Error(err, "to unstructured object")
		return
	}

	requests, err := rsmetrics.PodResourceRequests(obj)
	if err != nil {
		klog.Error(err, "unable to get pod resource requests")
		return
	}
	klog.Infof("Pod resource requests: %v", requests)
}

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	kubedbv1alpha2 "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
)

func FieldIndexer() {
	config := getRESTConfig()
	mgr, err := manager.New(config, manager.Options{
		Scheme: scm,
	})
	if err != nil {
		os.Exit(1)
	}

	// Set up field indexer
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kubedbv1alpha2.MongoDB{}, "metadata.annotations.hello", func(rawObj client.Object) []string {
		mongoDB := rawObj.(*kubedbv1alpha2.MongoDB)
		if val, ok := mongoDB.Annotations["hello"]; ok {
			return []string{val}
		}
		return nil
	}); err != nil {
		os.Exit(1)
	}

	go func() {
		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			klog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(context.Background()) {
		os.Exit(1)
	}

	kc := mgr.GetClient()
	var mongodbs kubedbv1alpha2.MongoDBList
	err = kc.List(context.TODO(), &mongodbs, client.MatchingFields{
		"metadata.annotations.hello": "world",
	})
	if err != nil {
		klog.Error(err, "unable to list MongoDBs")
		os.Exit(1)
	}

	for _, m := range mongodbs.Items {
		klog.Infof("%s/%s", m.Namespace, m.Name)
	}

	fmt.Println("Done")
}

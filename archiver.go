package main

import (
	"context"
	"fmt"
	"github.com/gobuffalo/flect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	kmapi "kmodules.xyz/client-go/api/v1"
	archiverapi "kubedb.dev/apimachinery/apis/archiver/v1alpha1"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubestash.dev/apimachinery/apis"
	"kubestash.dev/apimachinery/apis/core/v1alpha1"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Archiver() {
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
		klog.Infof("%s/%s", m.Namespace, m.Name)
	}

	fmt.Println("Done")

	a := MongoArchiver("az-vsc")
	//klog.Infof("%+v %+v %+v %+v \n %+v %+v %+v \n", *a.Spec.Databases, *a.Spec.RetentionPolicy, *a.Spec.EncryptionSecret, *a.Spec.FullBackup,
	//*a.Spec.ManifestBackup, *a.Spec.BackupStorage, *a.Spec.DeletionPolicy)
	err = kc.Create(context.TODO(), a)
	if err != nil {
		klog.Error(err, "unable to create MongoDB Archiver")
	}
}

const (
	ArchiverNamespace = "kubedb"
	StashNamespace    = "stash"

	defaultRetentionPolicy        = "keep-1mo"
	defaultEncryptionSecret       = "default-encryption-secret"
	defaultFullBackupSchedule     = "*/50 * * * *"
	defaultManifestBackupSchedule = "0 */2 * * *"
	defaultBackupStorage          = "default"
	SessionHistoryLimit           = 2
)

var defaultDBLabelSelector = map[string]string{
	"kubedb.com/archiver": "true",
}

func MongoArchiver(svcName string) *archiverapi.MongoDBArchiver {
	return &archiverapi.MongoDBArchiver{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      flect.Dasherize(svcName),
			Namespace: ArchiverNamespace,
		},
		Spec: archiverapi.MongoDBArchiverSpec{
			Databases: &dbapi.AllowedConsumers{
				Namespaces: &dbapi.ConsumerNamespaces{
					From: ptr.To(dbapi.NamespacesFromAll),
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: defaultDBLabelSelector,
				},
			},
			Pause: false,
			RetentionPolicy: &kmapi.ObjectReference{
				Namespace: StashNamespace,
				Name:      defaultRetentionPolicy,
			},
			EncryptionSecret: &kmapi.ObjectReference{
				Namespace: StashNamespace,
				Name:      defaultEncryptionSecret,
			},
			FullBackup: &archiverapi.FullBackupOptions{
				Driver: apis.DriverVolumeSnapshotter,
				//Task: &archiverapi.Task{Params: &runtime.RawExtension{
				//	Raw: []byte(`{"volumeSnapshotClassName":"svcName"}`),
				//}},
				Task: &archiverapi.Task{
					Params: &runtime.RawExtension{
						Raw: func() []byte {
							return []byte(fmt.Sprintf(`{"volumeSnapshotClassName":"%s"}`, svcName))
						}(),
						Object: nil,
					},
				},
				Scheduler: &archiverapi.SchedulerOptions{
					Schedule:                   defaultFullBackupSchedule,
					ConcurrencyPolicy:          "",
					JobTemplate:                v1alpha1.JobTemplate{},
					SuccessfulJobsHistoryLimit: ptr.To(int32(1)),
					FailedJobsHistoryLimit:     ptr.To(int32(1)),
				},
				ContainerRuntimeSettings: nil,
				JobTemplate:              nil,
				RetryConfig:              nil,
				Timeout:                  nil,
				SessionHistoryLimit:      SessionHistoryLimit,
			},
			WalBackup: nil,
			ManifestBackup: &archiverapi.ManifestBackupOptions{
				Scheduler: &archiverapi.SchedulerOptions{
					Schedule:                   defaultManifestBackupSchedule,
					ConcurrencyPolicy:          "",
					JobTemplate:                v1alpha1.JobTemplate{},
					SuccessfulJobsHistoryLimit: ptr.To(int32(1)),
					FailedJobsHistoryLimit:     ptr.To(int32(1)),
				},
				ContainerRuntimeSettings: nil,
				JobTemplate:              nil,
				RetryConfig:              nil,
				Timeout:                  nil,
				SessionHistoryLimit:      SessionHistoryLimit,
			},
			BackupStorage: &archiverapi.BackupStorage{
				Ref: &kmapi.ObjectReference{
					Namespace: StashNamespace,
					Name:      defaultBackupStorage,
				},
				SubDir: "",
			},
			DeletionPolicy: ptr.To(archiverapi.DeletionPolicyWipeOut),
		},
	}
}

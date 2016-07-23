package util

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

// CreateClient creates a client for Kubernetes cluster
func CreateClient(host string) (k *client.Client, err error) {
	var config *restclient.Config

	if len(host) == 0 {
		config, err = restclient.InClusterConfig()
		if err != nil {
			return k, err
		}
	} else {
		config = &restclient.Config{
			Host: host,
		}
	}

	k, err = client.New(config)
	return
}

// CopyJob creates a copy of k8s batch Job
func CopyJob(job batch.Job) *batch.Job {
	copy := batch.Job{}
	copy.Spec.Template.Spec = job.Spec.Template.Spec

	genName := "kron-" + job.GetName() + "-"

	copy.ObjectMeta.SetGenerateName(genName)
	copy.ObjectMeta.SetLabels(map[string]string{
		"origin":   "kron",
		"template": job.GetName(),
	})

	return &copy
}

// ListJobs gets kron jobs using a label
func ListJobs(k *client.Client, namespace string) (jobs *batch.JobList, err error) {
	kronSelector, err := labels.Parse("kron = true")
	if err != nil {
		return
	}

	opts := api.ListOptions{}
	opts.LabelSelector = kronSelector
	jobs, err = k.Batch().Jobs(namespace).List(opts)

	return
}

// Copyright © 2016 Sabaka OU <hello@sabaka.io>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package util

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type KronClient struct {
	K         *client.Client
	namespace string
}

// CreateClient creates a client for Kubernetes cluster
func NewClient(host string, namespace string) (k *KronClient, err error) {
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

	k8sClient, err := client.New(config)
	if err != nil {
		return k, err
	}

	k = &KronClient{
		namespace: namespace,
		K:         k8sClient,
	}
	return
}

// CopyJob creates a copy of k8s batch Job
func CopyJob(job *batch.Job) *batch.Job {
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

// ListJobExecutions finds all jobs scheduled by kron
func (k *KronClient) ListJobExecutions() (jobs *batch.JobList, err error) {
	kronSelector, err := labels.Parse("origin=kron")
	if err != nil {
		return
	}

	opts := api.ListOptions{}
	opts.LabelSelector = kronSelector
	jobs, err = k.K.Batch().Jobs(k.namespace).List(opts)

	return
}

// ListJobs finds all job templates
func (k *KronClient) ListJobs() (jobs *batch.JobList, err error) {
	kronSelector, err := labels.Parse("kron=true")
	if err != nil {
		return
	}

	opts := api.ListOptions{}
	opts.LabelSelector = kronSelector
	jobs, err = k.K.Batch().Jobs(k.namespace).List(opts)

	return
}

// WatchJobs watches jobs
func (k *KronClient) WatchJobs() (watcher watch.Interface, err error) {
	kronSelector, err := labels.Parse("kron=true")
	if err != nil {
		return
	}

	opts := api.ListOptions{}
	opts.LabelSelector = kronSelector
	opts.Watch = true
	watcher, err = k.K.Batch().Jobs(k.namespace).Watch(opts)

	return
}

func (k *KronClient) Jobs() client.JobInterface {
	return k.K.Batch().Jobs(k.namespace)
}

// DeletePodsInJob deletes all Pods which were created for a Job
func (k *KronClient) DeletePodsInJob(job *batch.Job) (err error) {
	deleteOpts := api.DeleteOptions{}
	listOpts := api.ListOptions{}
	uid := job.GetObjectMeta().GetUID()
	label := "controller-uid=" + fmt.Sprintf("%s", uid)

	selector, err := labels.Parse(label)
	if err != nil {
		log.Errorln(fmt.Sprintf("Error parsing label, skipping GC for job %s", job.GetName()), err)
		return
	}

	listOpts.LabelSelector = selector
	pods, err := k.K.Pods(k.namespace).List(listOpts)
	if err != nil {
		log.Errorln(fmt.Sprintf("Error getting pods, skipping GC for job %s", job.GetName()), err)
		return
	}

	for _, pod := range pods.Items {
		err := k.K.Pods(k.namespace).Delete(pod.GetName(), &deleteOpts)
		if err != nil {
			log.Errorln(fmt.Sprintf("Error deleting pod %s", pod.GetName()), err)
		}
	}

	return
}

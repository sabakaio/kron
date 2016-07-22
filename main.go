package main

import (
	"fmt"
	"log"

	"github.com/robfig/cron"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

func main() {
	k := CreateClient()
	jobs := ListJobs(k)

	cr := cron.New()

	for j, _ := range jobs.Items {
		fmt.Println("Scheduling job", jobs.Items[j].GetName())
		scheduledJob := CopyJob(jobs.Items[j])

		cr.AddFunc(jobs.Items[j].GetAnnotations()["schedule"], func() {
			createdJob, err := k.Batch().Jobs(api.NamespaceDefault).Create(scheduledJob)
			if err != nil {
				fmt.Println("xxx")
				log.Fatalln("Can't create Job:", err)
			}

			fmt.Println(createdJob.Name)
		})
	}

	fmt.Println("Starting kron")
	cr.Start()
	select {}
}

func CopyJob(job batch.Job) *batch.Job {
	newjob := batch.Job{}
	newjob.Spec.Template.Spec = job.Spec.Template.Spec
	newjob.ObjectMeta.SetGenerateName("kron-")
	return &newjob
}

func CreateClient() (k *client.Client) {
	config := &restclient.Config{
		Host: "http://localhost:8001",
	}
	k, err := client.New(config)

	if err != nil {
		log.Fatalln("Can't connect to Kubernetes API:", err)
	}

	return
}

func ListJobs(k *client.Client) (jobs *batch.JobList) {
	kronSelector, err := labels.Parse("origin = krontab")
	if err != nil {
		log.Fatalln("Can't parse selector:", err)
	}

	opts := api.ListOptions{}
	opts.LabelSelector = kronSelector
	jobs, err = k.Batch().Jobs(api.NamespaceDefault).List(opts)

	if err != nil {
		log.Fatalln("Can't get jobs from Kubernetes API:", err)
	}

	return
}

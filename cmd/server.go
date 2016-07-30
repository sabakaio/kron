// Copyright © 2016 Arseny Zarechnev <arseny@sabaka.io>
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

package cmd

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/sabakaio/kron/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a kron server",
	Long:  `Start a kron server`,
	Run:   serverFn,
}

var jobMapping map[string]cron.EntryID
var namespace string
var noGc bool
var gcInterval int
var gcAge float64

func serverFn(cmd *cobra.Command, args []string) {
	namespace = cmd.Flag("namespace").Value.String()
	jobMapping = map[string]cron.EntryID{}

	k, err := util.CreateClient(cmd.Flag("host").Value.String())
	if err != nil {
		log.Fatalln("Can't connect to Kubernetes API:", err)
	}

	watcher, err := util.WatchJobs(k, namespace)
	if err != nil {
		log.Fatalln("Can't start watching Jobs on Kubernetes API:", err)
	}

	cr := cron.New()
	log.Infoln("Starting kron")
	cr.Start()

	if noGc != true {
		go garbageCollect(k)
	}
	for event := range watcher.ResultChan() {
		eventListener(k, cr, event)
	}
}

func garbageCollect(k *client.Client) {
	log.Debugln("GC enabled, interval", gcInterval)
	for {
		log.Debugln("Collecting garbage")
		jobs, err := util.ListJobExecutions(k, namespace)
		if err != nil {
			log.Fatalln("Can't start watching Jobs on Kubernetes API:", err)
		}
		for _, job := range jobs.Items {
			t := job.GetCreationTimestamp()
			since := time.Since(t.Time).Hours()
			log.Debugln("Found job", job.GetName())
			log.Debugln("Since: ", since)
			if since > gcAge {
				deleteOpts := api.DeleteOptions{}
				listOpts := api.ListOptions{}
				uid := job.GetObjectMeta().GetUID()
				label := "controller-uid=" + fmt.Sprintf("%s", uid)
				selector, err := labels.Parse(label)
				if err != nil {
					log.Debugln(err)
				}
				listOpts.LabelSelector = selector
				pods, err := k.Pods(namespace).List(listOpts)
				if err != nil {
					log.Debugln(err)
				}
				for _, pod := range pods.Items {
					k.Pods(namespace).Delete(pod.GetName(), &deleteOpts)
				}
				k.Batch().Jobs(namespace).Delete(job.GetName(), &deleteOpts)
			}
		}
		time.Sleep(time.Duration(gcInterval) * time.Minute)
	}
}

func eventListener(k *client.Client, cr *cron.Cron, event watch.Event) {
	log.Debugln("Got event", event.Type)

	ref, err := api.GetReference(event.Object)
	if err != nil {
		log.Fatalln(err)
	}
	log.Debugln(ref.Name)

	switch event.Type {
	case watch.Deleted:
		cr.Remove(jobMapping[ref.Name])
		delete(jobMapping, ref.Name)
		log.Println(len(cr.Entries()))
		return
	case watch.Modified:
		cr.Remove(jobMapping[ref.Name])
		delete(jobMapping, ref.Name)
	case watch.Error:
		log.Panicln(event.Object)
		return
	}

	job, err := k.Batch().Jobs(namespace).Get(ref.Name)
	if err != nil {
		log.Panicln(err)
	}

	schedule := job.GetAnnotations()["schedule"]
	scheduledJob := util.CopyJob(job)

	id, _ := cr.AddFunc(schedule, func() {
		createdJob, err := k.Batch().Jobs(namespace).Create(scheduledJob)
		if err != nil {
			log.Fatalln("Can't create Job:", err)
		}

		log.Infoln("Scheduled a job:", createdJob.GetName())
	})

	jobMapping[ref.Name] = id
	log.Debugln("Total kron entries:", len(cr.Entries()))
}

func init() {
	serverCmd.Flags().BoolVar(&noGc, "no-gc", false, "Disable garbage collector")
	serverCmd.Flags().IntVar(&gcInterval, "gc-interval", 1, "Garbage collection interval in minutes")
	serverCmd.Flags().Float64Var(&gcAge, "gc-age", 0.1, "Garbage collect jobs older than this value in hours")
	RootCmd.AddCommand(serverCmd)
}

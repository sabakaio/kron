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

package cmd

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/sabakaio/kron/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/robfig/cron.v2"
	"k8s.io/kubernetes/pkg/api"
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
var k *util.KronClient

func serverFn(cmd *cobra.Command, args []string) {
	namespace = cmd.Flag("namespace").Value.String()
	jobMapping = map[string]cron.EntryID{}

	k, err := util.NewClient(cmd.Flag("host").Value.String(), namespace)
	if err != nil {
		log.Fatalln("Can't connect to Kubernetes API:", err)
	}

	watcher, err := k.WatchJobs()
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

func garbageCollect(k *util.KronClient) {
	log.Debugln("GC enabled, interval", gcInterval)
	for {
		log.Debugln("Collecting garbage")
		jobs, err := k.ListJobExecutions()
		if err != nil {
			log.Errorln("Can't list job executions during GC, skipping gc cycle:", err)
			continue // skip GC cycle
		}
		for _, job := range jobs.Items {
			t := job.GetCreationTimestamp()
			since := time.Since(t.Time).Hours()

			log.Debugln("Found job", job.GetName())

			if since > gcAge {
				log.Debugln("Job should be removed", job.GetName())
				err := k.DeletePodsInJob(&job)
				if err != nil {
					log.Errorln(fmt.Sprintf("Job %s will not be deleted, errors occured during pods deletion:", job.GetName()), err)
					continue // next job
				}
				deleteOpts := api.DeleteOptions{}
				err = k.Jobs().Delete(job.GetName(), &deleteOpts)

				if err != nil {
					log.Errorln(fmt.Sprintf("Error deleting job %s:", job.GetName()), err)
					continue // next job
				}

				log.Debugln("Job removed")
			}
		}
		time.Sleep(time.Duration(gcInterval) * time.Minute)
	}
}

func eventListener(k *util.KronClient, cr *cron.Cron, event watch.Event) {
	log.Debugln("Got event", event.Type)

	ref, err := api.GetReference(event.Object)
	if err != nil {
		log.Errorln("Error getting object reference, aborting event processing:", err)
		return
	}

	if len(ref.Name) == 0 {
		return
	}

	switch event.Type {
	case watch.Deleted:
		cr.Remove(jobMapping[ref.Name])
		delete(jobMapping, ref.Name)
		log.Debugln(len(cr.Entries()))
		return
	case watch.Modified:
		cr.Remove(jobMapping[ref.Name])
		delete(jobMapping, ref.Name)
	case watch.Error:
		log.Errorln("Got Error event:", event.Object)
		return
	}

	job, err := k.Jobs().Get(ref.Name)
	if err != nil {
		log.Errorln(fmt.Sprintf("Error getting job %s, aborting action:", ref.Name), err)
		return
	}

	schedule := job.GetAnnotations()["schedule"]
	scheduledJob := util.CopyJob(job)

	id, err := cr.AddFunc(schedule, func() {
		createdJob, err := k.Jobs().Create(scheduledJob)
		if err != nil {
			log.Errorln("Can't create Job:", err)
			return
		}

		log.Infoln("Schedulede a job ", createdJob.GetName())
	})

	if err != nil {
		log.Errorln(fmt.Sprintf("Error scheduling job %s, aborting action:", ref.Name), err)
		return
	}

	jobMapping[ref.Name] = id
	log.Debugln("Total kron entries:", len(cr.Entries()))
}

func init() {
	serverCmd.Flags().BoolVar(&noGc, "no-gc", false, "Disable garbage collector")
	serverCmd.Flags().IntVar(&gcInterval, "gc-interval", 5, "Garbage collection interval in minutes")
	serverCmd.Flags().Float64Var(&gcAge, "gc-age", 24, "Garbage collect jobs older than this value in hours")
	RootCmd.AddCommand(serverCmd)
}

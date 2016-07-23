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
	"log"

	"github.com/robfig/cron"
	"github.com/sabakaio/kron/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a kron server",
	Long:  `Start a kron server`,
	Run:   serverFn,
}

func serverFn(cmd *cobra.Command, args []string) {
	namespace := cmd.Flag("namespace").Value.String()
	k, err := util.CreateClient(cmd.Flag("host").Value.String())
	if err != nil {
		log.Fatalln("Can't connect to Kubernetes API:", err)
	}

	jobs, err := util.ListJobs(k, namespace)
	if err != nil {
		log.Fatalln("Can't get jobs from Kubernetes API:", err)
	}

	cr := cron.New()

	for j := range jobs.Items {
		schedule := jobs.Items[j].GetAnnotations()["schedule"]
		scheduledJob := util.CopyJob(jobs.Items[j])

		cr.AddFunc(schedule, func() {
			createdJob, err := k.Batch().Jobs(namespace).Create(scheduledJob)
			if err != nil {
				log.Fatalln("Can't create Job:", err)
			}

			log.Println("Scheduled a job:", createdJob.GetName())
		})
	}

	log.Println("Starting kron")
	cr.Start()
	select {}
}

func init() {
	RootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringP("host", "H", "", "Kubernetes host to connect to")
	serverCmd.Flags().StringP("namespace", "n", api.NamespaceDefault, "Kubernetes namespace for jobs")
}

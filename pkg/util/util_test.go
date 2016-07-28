package util

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/kubernetes/pkg/api"
	"testing"
)

func TestListJobs(t *testing.T) {
	Convey("List jobs", t, func() {
		k, err := CreateClient("http://localhost:8001")
		So(err, ShouldBeNil)

		jobs, err := ListJobs(k, "default")
		So(err, ShouldBeNil)

		name := jobs.Items[0].GetName()
		So(name, ShouldEqual, "test-job")
	})
}

func TestWatchJobs(t *testing.T) {
	Convey("Watch jobs", t, func() {
		k, err := CreateClient("http://localhost:8001")
		So(err, ShouldBeNil)

		watcher, err := WatchJobs(k, "default")
		So(err, ShouldBeNil)

		for {
			event, ok := <-watcher.ResultChan()
			ref, err := api.GetReference(event.Object)
			if err != nil {
				t.Fail()
			}
			job, err := k.Batch().Jobs(api.NamespaceDefault).Get(ref.Name)
			if err != nil {
				t.Fail()
			}
			fmt.Println(ok, event.Type, ref.ResourceVersion, ref.Name, job.GetName())
			// t.Log(event.Type)
			// t.Log(job.GetName())

		}

		// So(err, ShouldBeNil)

		// So(job.GetName(), ShouldEqual, "test-job")
	})
}

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

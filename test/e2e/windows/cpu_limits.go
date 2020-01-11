/*
Copyright 2019, 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package windows

import (
	"time"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	imageutils "k8s.io/kubernetes/test/utils/image"

	"github.com/onsi/ginkgo"
)

var _ = SIGDescribe("[Feature:Windows] Cpu Resources", func() {

	// framework.TestContext.GatherKubeSystemResourceUsageData = "true" // TODO: Or should this be GatherMetricsAfterTest
	// framework.TestContext.GatherMetricsAfterTest = "true"

	f := framework.NewDefaultFramework("cpu-resources-test-windows")

	powershellImage := imageutils.GetConfig(imageutils.BusyBox)

	ginkgo.Context("Container limits", func() {
		ginkgo.It("should not be exceeded after waiting 2 minutes", func() {
			ginkgo.By("Creating one pod with limit set to '0.5'")
			podsDecimal := newCpuBurnPods(1, powershellImage, "0.5", "1Gi")
			f.PodClient().CreateBatch(podsDecimal)
			ginkgo.By("Creating one pod with limit set to '500m'")
			podsMilli := newCpuBurnPods(1, powershellImage, "500m", "1Gi")
			f.PodClient().CreateBatch(podsMilli)
			// TODO: verify resource usage from framework.
			ginkgo.By("Waiting 2 minutes")
			time.Sleep(2 * time.Minute)
			// TODO: verify resulting config
			ginkgo.By("Ensuring pods are still running")
			for _, p := range podsDecimal {
				pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(p.Name,
					metav1.GetOptions{})
				framework.ExpectNoError(err, "Error retrieving pod")
				framework.ExpectEqual(pod.Status.Phase, v1.PodRunning)
			}
			for _, p := range podsMilli {
				pod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Get(p.Name,
					metav1.GetOptions{})
				framework.ExpectNoError(err, "Error retrieving pod")
				framework.ExpectEqual(pod.Status.Phase, v1.PodRunning)
			}
			ginkgo.By("Gathering pod metrics")


			ginkgo.By("Ensuring cpu doesn't exceed limit +/- 5%")
		})
	})
})

// newMemLimitTestPods creates a list of pods (specification) for test.
func newCpuBurnPods(numPods int, image imageutils.Config, cpuLimit string, memoryLimit string) []*v1.Pod {
	var pods []*v1.Pod

	memLimitQuantity, err := resource.ParseQuantity(memoryLimit)
	framework.ExpectNoError(err)

	cpuLimitQuantity, err := resource.ParseQuantity(cpuLimit)
	framework.ExpectNoError(err)

	for i := 0; i < numPods; i++ {

		podName := "cpulimittest-" + string(uuid.NewUUID())
		pod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
				Labels: map[string]string{
					"name":    podName,
					"testapp": "cpuburn",
				},
			},
			Spec: v1.PodSpec{
				// Restart policy is always (default).
				Containers: []v1.Container{
					{
						Image: image.GetE2EImage(),
						Name:  podName,
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceMemory: memLimitQuantity,
								v1.ResourceCPU: cpuLimitQuantity,
							},
						},
						Command: []string{
							"powershell.exe",
							"-Command",
							"foreach ($loopnumber in 1..8) { Start-Job -ScriptBlock { $result = 1; foreach($mm in 1..2147483647){$res1=1;foreach($num in 1..2147483647){$res1=$mm*$num*1340371};$res1} } } ; get-job | wait-job",
						},
					},
				},
				NodeSelector: map[string]string{
					"beta.kubernetes.io/os": "windows",
				},
			},
		}

		pods = append(pods, &pod)
	}

	return pods
}

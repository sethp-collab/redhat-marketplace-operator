// Copyright 2020 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	. "github.com/redhat-marketplace/redhat-marketplace-operator/test/rectest"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("node controller", func() {
		testNewNode(GinkgoT())
		testNodeLabelsAbsent(GinkgoT())
		testNodeDiffLabelsPresent(GinkgoT())
		testNodeUnknown(GinkgoT())
		testMultipleNodes(GinkgoT())
	})
})

var (
	name             = "new-node"
	nameLabelsAbsent = "new-node-labels-absent"
	nameLabelsDiff   = "new-node-diff-labels"
	kind             = "Node"

	node = corev1.Node{
		TypeMeta: v1.TypeMeta{
			Kind: "Node",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				watchResourceTag: watchResourceValue,
			},
		},
		Spec: corev1.NodeSpec{},
	}

	nodeLabelsAbsent = corev1.Node{
		TypeMeta: v1.TypeMeta{
			Kind: "Node",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:   nameLabelsAbsent,
			Labels: map[string]string{},
		},
		Spec: corev1.NodeSpec{},
	}

	nodeLabelsDiff = corev1.Node{
		TypeMeta: v1.TypeMeta{
			Kind: "Node",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: nameLabelsDiff,
			Labels: map[string]string{
				"testKey": "testValue",
			},
		},
		Spec: corev1.NodeSpec{},
	}
)

func generateOpts(name string) []StepOption {
	return []StepOption{
		WithRequest(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: name,
			},
		}),
	}
}

func setup(r *ReconcilerTest) error {
	r.Client = fake.NewFakeClient(r.GetGetObjects()...)
	r.Reconciler = &ReconcileNode{client: r.Client, scheme: scheme.Scheme}
	return nil
}

func testNewNode(t GinkgoTInterface) {
	t.Parallel()
	reconcilerTest := NewReconcilerTest(setup, node.DeepCopyObject())
	reconcilerTest.TestAll(t,
		// Reconcile to create obj
		ReconcileStep(generateOpts(name),
			ReconcileWithExpectedResults(DoneResult)),
		// List and check results
		ListStep(generateOpts(name),
			ListWithObj(&corev1.NodeList{}),
			ListWithFilter(
				client.MatchingLabels(map[string]string{
					watchResourceTag: watchResourceValue,
				})),
			ListWithCheckResult(func(r *ReconcilerTest, t ReconcileTester, i runtime.Object) {
				list, ok := i.(*corev1.NodeList)

				assert.Truef(t, ok, "expected node list got type %T", i)
				assert.Equal(t, 1, len(list.Items))

			}),
		),
	)
}

func testNodeLabelsAbsent(t GinkgoTInterface) {
	t.Parallel()
	reconcilerTest := NewReconcilerTest(setup, nodeLabelsAbsent.DeepCopyObject())
	reconcilerTest.TestAll(t,
		// Reconcile to create obj
		ReconcileStep(generateOpts(nameLabelsAbsent),
			ReconcileWithExpectedResults(DoneResult)),
		// List and check results
		ListStep(generateOpts(nameLabelsAbsent),
			ListWithObj(&corev1.NodeList{}),
			ListWithFilter(
				client.MatchingLabels(map[string]string{
					watchResourceTag: watchResourceValue,
				})),
			ListWithCheckResult(func(r *ReconcilerTest, t ReconcileTester, i runtime.Object) {
				list, ok := i.(*corev1.NodeList)

				assert.Truef(t, ok, "expected node list got type %T", i)
				assert.Equal(t, 1, len(list.Items))
			}),
		),
	)
}

func testNodeDiffLabelsPresent(t GinkgoTInterface) {
	t.Parallel()
	reconcilerTest := NewReconcilerTest(setup, nodeLabelsDiff.DeepCopyObject())
	reconcilerTest.TestAll(t,
		// Reconcile to create obj
		ReconcileStep(generateOpts(nameLabelsDiff),
			ReconcileWithExpectedResults(DoneResult)),
		// List and check results
		ListStep(generateOpts(nameLabelsDiff),
			ListWithObj(&corev1.NodeList{}),
			ListWithFilter(
				client.MatchingLabels(map[string]string{
					watchResourceTag: watchResourceValue,
				})),
			ListWithCheckResult(func(r *ReconcilerTest, t ReconcileTester, i runtime.Object) {
				list, ok := i.(*corev1.NodeList)

				assert.Truef(t, ok, "expected node list got type %T", i)
				assert.Equal(t, 1, len(list.Items))
				assert.Equal(t, "testValue", list.Items[0].GetLabels()["testKey"])
			}),
		),
	)
}

func testNodeUnknown(t GinkgoTInterface) {
	t.Parallel()
	reconcilerTest := NewReconcilerTest(setup, nodeLabelsDiff.DeepCopyObject())
	reconcilerTest.TestAll(t,
		// Reconcile to create obj
		ReconcileStep(generateOpts("DUMMY"),
			ReconcileWithExpectedResults(DoneResult)),
		// List and check results
		ListStep(generateOpts("DUMMY"),
			ListWithObj(&corev1.NodeList{}),
			ListWithFilter(
				client.MatchingLabels(map[string]string{
					watchResourceTag: watchResourceValue,
				})),
			ListWithCheckResult(func(r *ReconcilerTest, t ReconcileTester, i runtime.Object) {
				list, ok := i.(*corev1.NodeList)

				assert.Truef(t, ok, "expected node list got type %T", i)
				assert.Equal(t, 0, len(list.Items))
			}),
		),
	)
}

func testMultipleNodes(t GinkgoTInterface) {
	t.Parallel()
	reconcilerTest := NewReconcilerTest(setup, node.DeepCopyObject(), nodeLabelsAbsent.DeepCopyObject())
	reconcilerTest.TestAll(t,
		// Reconcile to create obj
		ReconcileStep([]StepOption{generateOpts(name)[0], generateOpts(nameLabelsAbsent)[0]},
			ReconcileWithExpectedResults(DoneResult)),
		// List and check results
		ListStep([]StepOption{},
			ListWithObj(&corev1.NodeList{}),
			ListWithFilter(
				client.MatchingLabels(map[string]string{
					watchResourceTag: watchResourceValue,
				})),
			ListWithCheckResult(func(r *ReconcilerTest, t ReconcileTester, i runtime.Object) {
				list, ok := i.(*corev1.NodeList)

				assert.Truef(t, ok, "expected node list got type %T", i)
				assert.Equal(t, 2, len(list.Items))

			}),
		),
	)
}

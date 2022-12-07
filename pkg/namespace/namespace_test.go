package namespace

import (
	"testing"

	"github.com/EdgeNet-project/edgenet/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetAsOwnerReference(t *testing.T) {
	cases := []struct {
		name     string
		expected []metav1.OwnerReference
	}{
		{
			"test-1",
			[]metav1.OwnerReference{
				{
					Kind:       "Namespace",
					Name:       "test-1",
					APIVersion: "v1",
				},
			},
		},
		{
			"test1",
			[]metav1.OwnerReference{
				{
					Kind:       "Namespace",
					Name:       "test1",
					APIVersion: "v1",
				},
			},
		},
		{
			"test-2",
			[]metav1.OwnerReference{
				{
					Kind:       "Namespace",
					Name:       "test-2",
					APIVersion: "v1",
				},
			},
		},
	}
	for _, tc := range cases {
		namespaceObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tc.name}}
		result := SetAsOwnerReference(namespaceObj)
		tc.expected[0].Controller = result[0].Controller
		tc.expected[0].BlockOwnerDeletion = result[0].BlockOwnerDeletion

		util.Equals(t, tc.expected, result)
	}
}

package utils

import (
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *KubernetesTestClient) CreateNamespace(name string, updateIfExists bool) (*v1.Namespace, error) {
	namespace, err := c.
		Clientset.
		CoreV1().
		Namespaces().
		Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	if err != nil && apierrors.IsAlreadyExists(err) && updateIfExists {
		return c.UpdateNamespace(name)
	}
	return namespace, err
}

func (c *KubernetesTestClient) UpdateNamespace(name string) (*v1.Namespace, error) {
	return c.
		Clientset.
		CoreV1().
		Namespaces().
		Update(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
}

func (c *KubernetesTestClient) DeleteNamespace(name string) error {
	deleteGracePeriod := int64(1)
	deletePolicy := metav1.DeletePropagationForeground
	return c.
		Clientset.
		CoreV1().
		Namespaces().
		Delete(name, &metav1.DeleteOptions{
			PropagationPolicy:  &deletePolicy,
			GracePeriodSeconds: &deleteGracePeriod,
		})
}

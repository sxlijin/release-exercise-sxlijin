/*
Copyright 2023.

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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// Using `api` instead of `releaseexercisesourcegraphcomv1` for readability. Not sure what I would use in practice.
	api "release-exercise/api/v1"
)

// HelloWorldReconciler reconciles a HelloWorld object
type HelloWorldReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=release-exercise.sourcegraph.com.release-exercise.sourcegraph.com,resources=helloworlds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=release-exercise.sourcegraph.com.release-exercise.sourcegraph.com,resources=helloworlds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=release-exercise.sourcegraph.com.release-exercise.sourcegraph.com,resources=helloworlds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelloWorld object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *HelloWorldReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciler is running", "logged-namespace", req.NamespacedName)

	newPod := createPod(&req.NamespacedName)
	existingPod := &corev1.Pod{}
	if getPodErr := r.Client.Get(ctx, types.NamespacedName{Name: newPod.Name, Namespace: newPod.Namespace}, existingPod); getPodErr == nil {
		if err := r.Client.Delete(ctx, existingPod); err != nil {
			// TODO: what is the correct error behavior here?
			return ctrl.Result{}, err
		}
		// TODO: best-effort block/defer until the pod is actually deleted
		// creation after deletion right now appears to work because the controller will retry reconciliation,
		// but I'm not clear on exactly what the retry policy semantics are
	}

	var helloworld api.HelloWorld
	if err := r.Client.Get(ctx, req.NamespacedName, &helloworld); err != nil {
		retErr := client.IgnoreNotFound(err)
		if retErr != nil {
			logger.Error(retErr, "unable to fetch HelloWorld resource")
		}
		return ctrl.Result{}, retErr
	}
	newPod.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:    "echo-container",
				Image:   "ubuntu",
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("echo %s && tail -f /dev/null", helloworld.Spec.Message)},
			},
		},
	}
	// Set the owner reference to the custom resource
	if err := controllerutil.SetControllerReference(&helloworld, newPod, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Create(ctx, newPod); err != nil {
		// This error will recur until the above described pod deletion succeeds
		logger.Error(err, "failed to create new pod")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func createPod(helloworldName *types.NamespacedName) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pod", helloworldName.Name),
			Namespace: helloworldName.Namespace,
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelloWorldReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.HelloWorld{}).
		Complete(r)
}

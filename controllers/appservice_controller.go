/*
Copyright 2021 cnych.

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

package controllers

import (
	"context"
	appv1beta1 "github.com/cnych/opdemo/v2/api/v1beta1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.ydzs.io,resources=appservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.ydzs.io,resources=appservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.ydzs.io,resources=appservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("appservice", req.NamespacedName)

	// your logic here
	var appService appv1beta1.AppService
	err := r.Get(ctx, req.NamespacedName, &appService)
	if err != nil {
		//myapp 被删除的时候 忽略没有找到的错误
		//第二个返回值为 err 会重新入队列 为 nil 意味着处理完成 则会被 forget
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("fetch appService object", "appService", appService)

	//得到 appService 后去创建对应的 Deployment 和 Service
	// createOrUpdate
	// 观察当前状态与期望状态进行对比
	//	调谐 获取到当前的一个状态 然后和我们期望的状态进行对比
	var deploy appsv1.Deployment
	deploy.Name = appService.Name
	deploy.Namespace = appService.Namespace
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		MutateDeployment(&appService, &deploy)
		return controllerutil.SetControllerReference(&appService, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Deployment", or)

	var svc corev1.Service
	svc.Name = appService.Name
	svc.Namespace = appService.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		MutateService(&appService, &svc)
		return controllerutil.SetControllerReference(&appService, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Service", or)

	return ctrl.Result{}, nil
}

func MutateService(app *appv1beta1.AppService, svc *corev1.Service) {
	svc.Spec = corev1.ServiceSpec{
		Ports:     app.Spec.Ports,
		Selector:  map[string]string{"myapp": app.Name},
		ClusterIP: svc.Spec.ClusterIP,
		Type:      corev1.ServiceTypeNodePort,
	}
}

func MutateDeployment(app *appv1beta1.AppService, deploy *appsv1.Deployment) {
	lables := map[string]string{"myapp": app.Name}
	selector := &metav1.LabelSelector{
		MatchLabels: lables,
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: app.Spec.Size,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: lables,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
		Selector: selector,
	}

}

func newContainers(app *appv1beta1.AppService) []corev1.Container {
	var containerPorts []corev1.ContainerPort
	for _, servicePort := range app.Spec.Ports {
		cPort := corev1.ContainerPort{}
		cPort.ContainerPort = servicePort.TargetPort.IntVal
		containerPorts = append(containerPorts, cPort)
	}

	return []corev1.Container{{
		Name:            app.Name,
		Image:           app.Spec.Image,
		Ports:           containerPorts,
		Env:             app.Spec.Envs,
		Resources:       corev1.ResourceRequirements{},
		ImagePullPolicy: corev1.PullIfNotPresent,
	}}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.AppService{}).
		Owns(&appsv1.Deployment{}). //不添加删除 deployment 时不会监听到事件
		Owns(&corev1.Service{}).
		Complete(r)
}

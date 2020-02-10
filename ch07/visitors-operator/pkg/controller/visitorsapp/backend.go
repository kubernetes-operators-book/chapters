package visitorsapp

import (
	"context"
	"time"

	examplev1 "github.com/jdob/visitors-operator/pkg/apis/example/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const backendPort = 8000
const backendServicePort = 30685
const backendImage = "jdob/visitors-service:1.0.0"

func backendDeploymentName(v *examplev1.VisitorsApp) string {
	return v.Name + "-backend"
}

func backendServiceName(v *examplev1.VisitorsApp) string {
	return v.Name + "-backend-service"
}

func (r *ReconcileVisitorsApp) backendDeployment(v *examplev1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "backend")
	size := v.Spec.Size

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key: "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key: "password",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:		backendDeploymentName(v),
			Namespace: 	v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:	backendImage,
						ImagePullPolicy: corev1.PullAlways,
						Name:	"visitors-service",
						Ports:	[]corev1.ContainerPort{{
							ContainerPort: 	backendPort,
							Name:			"visitors",
						}},
						Env:	[]corev1.EnvVar{
							{
								Name:	"MYSQL_DATABASE",
								Value:	"visitors",
							},
							{
								Name:	"MYSQL_SERVICE_HOST",
								Value:	mysqlServiceName(),
							},
							{
								Name:	"MYSQL_USERNAME",
								ValueFrom: userSecret,
							},
							{
								Name:	"MYSQL_PASSWORD",
								ValueFrom: passwordSecret,
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, dep, r.scheme)
	return dep
}

func (r *ReconcileVisitorsApp) backendService(v *examplev1.VisitorsApp) *corev1.Service {
	labels := labels(v, "backend")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:		backendServiceName(v),
			Namespace: 	v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol: corev1.ProtocolTCP,
				Port: backendPort,
				TargetPort: intstr.FromInt(backendPort),
				NodePort: 30685,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, s, r.scheme)
	return s
}

func (r *ReconcileVisitorsApp) updateBackendStatus(v *examplev1.VisitorsApp) (error) {
	v.Status.BackendImage = backendImage
	err := r.client.Status().Update(context.TODO(), v)
	return err
}

func (r *ReconcileVisitorsApp) handleBackendChanges(v *examplev1.VisitorsApp) (*reconcile.Result, error) {
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      backendDeploymentName(v),
		Namespace: v.Namespace,
	}, found)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &reconcile.Result{RequeueAfter:5 * time.Second}, err
	}

	size := v.Spec.Size

	if size != *found.Spec.Replicas {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return &reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return &reconcile.Result{Requeue: true}, nil
	}

	return nil, nil
}
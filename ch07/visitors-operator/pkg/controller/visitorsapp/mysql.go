package visitorsapp

import (
	"context"

	examplev1 "github.com/jdob/visitors-operator/pkg/apis/example/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func mysqlDeploymentName() string {
	return "mysql"
}

func mysqlServiceName() string {
	return "mysql-service"
}

func mysqlAuthName() string {
	return "mysql-auth"
}

func (r *ReconcileVisitorsApp) mysqlAuthSecret(v *examplev1.VisitorsApp) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:		mysqlAuthName(),
			Namespace:	v.Namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"username": "visitors-user",
			"password": "visitors-pass",
		},
	}
	controllerutil.SetControllerReference(v, secret, r.scheme)
	return secret
}

func (r *ReconcileVisitorsApp) mysqlDeployment(v *examplev1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "mysql")
	size := int32(1)

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
			Name:		mysqlDeploymentName(),
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
						Image:	"mysql:5.7",
						Name:	"visitors-mysql",
						Ports:	[]corev1.ContainerPort{{
							ContainerPort: 	3306,
							Name:			"mysql",
						}},
						Env:	[]corev1.EnvVar{
							{
								Name:	"MYSQL_ROOT_PASSWORD",
								Value: 	"password",
							},
							{
								Name:	"MYSQL_DATABASE",
								Value:	"visitors",
							},
							{
								Name:	"MYSQL_USER",
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

func (r *ReconcileVisitorsApp) mysqlService(v *examplev1.VisitorsApp) *corev1.Service {
	labels := labels(v, "mysql")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:		mysqlServiceName(),
			Namespace: 	v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: 	labels,
			Ports: 		[]corev1.ServicePort{{
				Port: 		3306,
			}},
			ClusterIP:	"None",
		},
	}

	controllerutil.SetControllerReference(v, s, r.scheme)
	return s
}

// Returns whether or not the MySQL deployment is running
func (r *ReconcileVisitorsApp) isMysqlUp(v *examplev1.VisitorsApp) (bool) {
	deployment := &appsv1.Deployment{}

	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name: mysqlDeploymentName(),
		Namespace: v.Namespace,
		}, deployment)

	if err != nil {
		log.Error(err, "Deployment mysql not found")
		return false
	}

	if deployment.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}

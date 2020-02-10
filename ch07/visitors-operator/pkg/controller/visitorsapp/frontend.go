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

const frontendPort = 3000
const frontendServicePort = 30686
const frontendImage = "jdob/visitors-webui:1.0.0"

func frontendDeploymentName(v *examplev1.VisitorsApp) string {
	return v.Name + "-frontend"
}

func frontendServiceName(v *examplev1.VisitorsApp) string {
	return v.Name + "-frontend-service"
}

func (r *ReconcileVisitorsApp) frontendDeployment(v *examplev1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "frontend")
	size := int32(1)

	// If the header was specified, add it as an env variable
	env := []corev1.EnvVar{}
	if v.Spec.Title != "" {
		env = append(env, corev1.EnvVar{
			Name:	"REACT_APP_TITLE",
			Value:	v.Spec.Title,
		})
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:		frontendDeploymentName(v),
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
						Image:	frontendImage,
						Name:	"visitors-webui",
						Ports:	[]corev1.ContainerPort{{
							ContainerPort: 	frontendPort,
							Name:			"visitors",
						}},
						Env:	env,
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, dep, r.scheme)
	return dep
}

func (r *ReconcileVisitorsApp) frontendService(v *examplev1.VisitorsApp) *corev1.Service {
	labels := labels(v, "frontend")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:		frontendServiceName(v),
			Namespace: 	v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol: corev1.ProtocolTCP,
				Port: frontendPort,
				TargetPort: intstr.FromInt(frontendPort),
				NodePort: frontendServicePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	log.Info("Service Spec", "Service.Name", s.ObjectMeta.Name)

	controllerutil.SetControllerReference(v, s, r.scheme)
	return s
}

func (r *ReconcileVisitorsApp) updateFrontendStatus(v *examplev1.VisitorsApp) (error) {
	v.Status.FrontendImage = frontendImage
	err := r.client.Status().Update(context.TODO(), v)
	return err
}

func (r *ReconcileVisitorsApp) handleFrontendChanges(v *examplev1.VisitorsApp) (*reconcile.Result, error) {
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      frontendDeploymentName(v),
		Namespace: v.Namespace,
	}, found)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &reconcile.Result{RequeueAfter:5 * time.Second}, err
	}

	title := v.Spec.Title
	existing := (*found).Spec.Template.Spec.Containers[0].Env[0].Value

	if title != existing {
		(*found).Spec.Template.Spec.Containers[0].Env[0].Value = title
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
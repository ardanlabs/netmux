package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8SController struct {
}

const (
	NetmuxName = "netmux"
	VolumeName = "netmux-tokens"
)

var def *K8SController

func DefaultK8sController() *K8SController {
	if def == nil {
		def = new(K8SController)
	}
	return def
}

func NewK8SController() *K8SController {
	var ret = &K8SController{}
	return ret
}

func (k *K8SController) TestConn(ctx *Context) error {
	fmt.Println("Get Kubernetes pods")

	fmt.Printf("Using kubeconfig: %s\n", ctx.Runtime.Kubeconfig)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", ctx.Runtime.Kubeconfig)
	if err != nil {
		fmt.Printf("error getting Kubernetes config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		fmt.Printf("error getting Kubernetes clientset: %v\n", err)
		os.Exit(1)
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(context.Background(), v1.ListOptions{})
	if err != nil {
		fmt.Printf("error getting pods: %v\n", err)
		os.Exit(1)
	}
	for _, pod := range pods.Items {
		fmt.Printf("Pod name: %s\n", pod.Name)
	}
	return nil
}

func (k *K8SController) deployCreateNamespace(ctx context.Context, cli *kubernetes.Clientset) error {
	var dep = &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: NetmuxName,
		},
		Spec: corev1.NamespaceSpec{},
	}
	_, err := cli.CoreV1().Namespaces().Create(ctx, dep, v1.CreateOptions{})
	return err
}

func (k *K8SController) deployCreateRBACForCluster(ctx context.Context, cli *kubernetes.Clientset) error {

	/*
		apiVersion: v1
		kind: ServiceAccount
		metadata:
		  name: netmux
		  namespace: netmux
	*/

	var serviceAccount = &corev1.ServiceAccount{

		ObjectMeta: v1.ObjectMeta{
			Namespace: NetmuxName,
			Name:      NetmuxName,
		},
	}

	_, err := cli.CoreV1().ServiceAccounts(NetmuxName).Create(ctx, serviceAccount, v1.CreateOptions{})
	if err != nil {
		return err
	}

	/*
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  # "namespace" omitted since ClusterRoles are not namespaced
		  name: netmux
		rules:
		  - apiGroups: [ "" ]
		    resources: [ "nodes", "services", "pods", "endpoints" ]
		    verbs: [ "get", "list", "watch" ]
		  - apiGroups: [ "extensions" ]
		    resources: [ "deployments" ]
		    verbs: [ "get", "list", "watch" ]

	*/

	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name: NetmuxName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"nodes", "services", "pods", "endpoints"},
			},
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets"},
			},
		},
	}
	_, err = cli.RbacV1().ClusterRoles().Create(ctx, clusterRole, v1.CreateOptions{})
	if err != nil {
		return err
	}

	/*
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  name: netmux
		subjects:
		  - kind: ServiceAccount
		    name: netmux # name of your service account
		    namespace: netmux # this is the namespace your service account is in
		roleRef: # referring to your ClusterRole
		  kind: ClusterRole
		  name: view
		  apiGroup: rbac.authorization.k8s.io
	*/

	var clusterRoleBinding = &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: NetmuxName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      NetmuxName,
				Namespace: NetmuxName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     NetmuxName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = cli.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, v1.CreateOptions{})
	if err != nil {
		return err
	}
	return err
}

func (k *K8SController) deployCreateRBACForNS(ctx context.Context, cli *kubernetes.Clientset, ns string) error {

	/*
		apiVersion: v1
		kind: ServiceAccount
		metadata:
		  name: netmux
		  namespace: netmux
	*/

	var serviceAccount = &corev1.ServiceAccount{

		ObjectMeta: v1.ObjectMeta{
			Namespace: ns,
			Name:      NetmuxName,
		},
	}

	_, err := cli.CoreV1().ServiceAccounts(ns).Create(ctx, serviceAccount, v1.CreateOptions{})
	if err != nil {
		logrus.Warnf("Error creating ns: %s", err.Error())
	}

	/*
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  # "namespace" omitted since ClusterRoles are not namespaced
		  name: netmux
		rules:
		  - apiGroups: [ "" ]
		    resources: [ "nodes", "services", "pods", "endpoints" ]
		    verbs: [ "get", "list", "watch" ]
		  - apiGroups: [ "extensions" ]
		    resources: [ "deployments" ]
		    verbs: [ "get", "list", "watch" ]

	*/

	nsrole := &rbacv1.Role{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      NetmuxName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"nodes", "services", "pods", "endpoints"},
			},
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets"},
			},
		},
	}
	_, err = cli.RbacV1().Roles(ns).Create(ctx, nsrole, v1.CreateOptions{})
	if err != nil {
		logrus.Warnf("Error creating role: %s", err.Error())
	}

	/*
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  name: netmux
		subjects:
		  - kind: ServiceAccount
		    name: netmux # name of your service account
		    namespace: netmux # this is the namespace your service account is in
		roleRef: # referring to your ClusterRole
		  kind: ClusterRole
		  name: view
		  apiGroup: rbac.authorization.k8s.io
	*/

	var nsrRoleBinding = &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: NetmuxName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      NetmuxName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     NetmuxName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = cli.RbacV1().RoleBindings(ns).Create(ctx, nsrRoleBinding, v1.CreateOptions{})
	if err != nil {
		logrus.Warnf("Error creating role: %s", err.Error())
	}
	return err
}

func (k *K8SController) deployCreateSecrets(ctx context.Context, cli *kubernetes.Clientset, ns string) error {

	/*
		apiVersion: v1
		kind: Secret
		metadata:
		 namespace: netmux
		 name: netmux-passwords
		stringData:
		  passwd.yaml: |-
		    entries:
		    - user: nx #Pass is nx
		      hash: $argon2id$v=19$m=65536,t=3,p=2$UjSoh5FMTxk8zbn9hfrPVQ$r6b/zn051cycC7aYJEDnft19VEsLOL5BMUhrMnfa1ag
		    - user: bill #Pass is kennedy
		      hash: $argon2id$v=19$m=65536,t=3,p=2$/GJ+09Oz31CpG9oPgfs/KQ$NKP5TmP8IL1d3Gli7hRCzbiIseDl+4OVdSSu1Eot0gk

	*/

	var secData = `entries:
  - user: nx #Pass is nx
    hash: $argon2id$v=19$m=65536,t=3,p=2$UjSoh5FMTxk8zbn9hfrPVQ$r6b/zn051cycC7aYJEDnft19VEsLOL5BMUhrMnfa1ag`

	var secret = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: ns,
			Name:      "netmux-passwords",
		},
		StringData: map[string]string{"passwd.yaml": secData},
	}

	_, err := cli.CoreV1().Secrets(ns).Create(ctx, secret, v1.CreateOptions{})

	return err
}

func (k *K8SController) deployCreatePVC(ctx context.Context, cli *kubernetes.Clientset, ns string) error {

	qntd, err := resource.ParseQuantity("10Mi")
	if err != nil {
		return err
	}

	var dep = &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      VolumeName,
			Namespace: ns,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: qntd,
				},
			},
		},
	}
	_, err = cli.CoreV1().PersistentVolumeClaims(ns).Create(ctx, dep, v1.CreateOptions{})
	return err
}

func (k *K8SController) deployCreateDeployment(ctx context.Context, cli *kubernetes.Clientset, arch string, ns string, global bool) error {

	/*
		kind: Deployment
		apiVersion: apps/v1
		metadata:
		  name: netmux
		  namespace: netmux
		spec:
		  replicas: 1
		  selector:
		    matchLabels:
		      app: netmux
		  template:
		    metadata:
		      labels:
		        app: netmux
		    spec:
		      volumes:
		        - name: passwd
		          secret:
		            secretName: netmux-passwords
		        - name: tokens
		          persistentVolumeClaim:
		            claimName: netmux-tokens
		      serviceAccountName: netmux

		      containers:
		        - name: netmux
		          image: digitalcircle/netmux:arm64
		          imagePullPolicy: IfNotPresent
		          ports:
		            - containerPort: 48080
		          volumeMounts:
		            - mountPath: /app/etc/passwd.yaml
		              subPath: passwd.yaml
		              name: passwd
		            - name: tokens
		              mountPath: /app/persistence


	*/
	var replicasRaw = int32(1)
	var dep = &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      NetmuxName,
			Namespace: ns,
			Labels:    map[string]string{"app": NetmuxName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicasRaw,
			Selector: &v1.LabelSelector{
				MatchLabels:      map[string]string{"app": NetmuxName},
				MatchExpressions: nil,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"app": NetmuxName},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: NetmuxName,
					Volumes: []corev1.Volume{
						{
							Name: "passwd",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "netmux-passwords",
								},
							},
						}, {
							Name: "tokens",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "netmux-tokens",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  NetmuxName,
							Image: "digitalcircle/netmux:" + arch,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 48080,
									Protocol:      "TCP"},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tokens",
									MountPath: "/app/persistence",
								},
								{
									Name:      "passwd",
									MountPath: "/app/etc/passwd.yaml",
									SubPath:   "passwd.yaml",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
		},
	}

	if !global {
		dep.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  "NS",
				Value: ns,
			},
		}
	}
	_, err := cli.AppsV1().Deployments(ns).Create(ctx, dep, v1.CreateOptions{})

	//cli.AppsV1().Deployments(NetmuxName).List(ctx, v1.ListOptions{
	//	FieldSelector: "metadata.name = netmux",
	//})

	return err
}

func (k *K8SController) deployCreateService(ctx context.Context, cli *kubernetes.Clientset, ns string) error {

	/*
		apiVersion: v1
		kind: Service
		metadata:
		  name: netmux
		  namespace: netmux
		spec:
		  ports:
		    - port: 48080
		      name: netmux
		      protocol: TCP
		  selector:
		    app: netmux

	*/

	var dep = &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      NetmuxName,
			Namespace: ns,
			Labels:    map[string]string{"app": NetmuxName},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     48080,
					Protocol: "TCP",
				},
			},
			Selector: map[string]string{"app": NetmuxName},
		},
	}
	_, err := cli.CoreV1().Services(ns).Create(ctx, dep, v1.CreateOptions{})
	return err
}

func (k *K8SController) setupDeployGlobal(ctx context.Context, clientset *kubernetes.Clientset, nmxctx *Context, ns string, kctx string, arch string) error {
	err := k.deployCreateNamespace(ctx, clientset)

	if err != nil {
		logrus.Warnf("Error creating namespace for global control: %s", err)
	}

	err = k.deployCreateRBACForCluster(ctx, clientset)

	if err != nil {
		logrus.Warnf("Error creating rbac for cluster: %s", err)
	}

	err = k.deployCreateSecrets(ctx, clientset, NetmuxName)

	if err != nil {
		logrus.Warnf("Error creating secrets: %s", err)
	}
	err = k.deployCreatePVC(ctx, clientset, NetmuxName)

	if err != nil {
		logrus.Warnf("Error creating pvc: %s", err)
	}
	err = k.deployCreateDeployment(ctx, clientset, arch, NetmuxName, true)

	if err != nil {
		logrus.Warnf("Error creating deployment: %s", err)
	}
	err = k.deployCreateService(ctx, clientset, NetmuxName)

	if err != nil {
		logrus.Warnf("Error creating service: %s", err)
	}
	return nil
}

func (k *K8SController) setupDeployNS(ctx context.Context, clientset *kubernetes.Clientset, nmxctx *Context, ns string, kctx string, arch string) error {
	err := k.deployCreateRBACForNS(ctx, clientset, ns)

	if err != nil {
		logrus.Warnf("Error creating rbac: %s", err)
	}

	err = k.deployCreateSecrets(ctx, clientset, ns)

	if err != nil {
		logrus.Warnf("Error creating secrets: %s", err)
	}
	err = k.deployCreatePVC(ctx, clientset, ns)

	if err != nil {
		logrus.Warnf("Error creating pvc: %s", err)
	}
	err = k.deployCreateDeployment(ctx, clientset, arch, ns, false)

	if err != nil {
		logrus.Warnf("Error creating deployment: %s", err)
	}
	err = k.deployCreateService(ctx, clientset, ns)

	if err != nil {
		logrus.Warnf("Error creating service: %s", err)
	}
	return nil
}

func (k *K8SController) checkDeployReady(ctx context.Context, clientset *kubernetes.Clientset, ns string) {

	tc := time.NewTicker(time.Second * 5)
	dl := time.NewTicker(time.Minute * 2)

	for {
		select {
		case <-tc.C:
			list, err := clientset.AppsV1().Deployments(ns).List(ctx, v1.ListOptions{FieldSelector: "metadata.name=" + NetmuxName})
			if err != nil {
				logrus.Warnf("Error checking cluster: %s", err.Error())
			}
			for _, v := range list.Items {
				if v.Status.ReadyReplicas > 0 {
					logrus.Infof("Deployment is up - will proceed")
					dl.Stop()
					tc.Stop()
					return
				}
			}
		case <-dl.C:
			dl.Stop()
			tc.Stop()
			logrus.Warnf("It took too long for deploying the stack - please check your cluster")
			return
		}
	}
}

func (k *K8SController) SetupDeploy(ctx context.Context, nmxctx *Context, ns string, kctx string, arch string) error {

	kubeConfig, err := resolveClientConfig(nmxctx.Runtime.Kubeconfig, nmxctx.Runtime.Kubecontext)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	if ns == "*" {
		k.setupDeployGlobal(ctx, clientset, nmxctx, ns, kctx, arch)
		ns = NetmuxName
	} else {
		k.setupDeployNS(ctx, clientset, nmxctx, ns, kctx, arch)

	}
	k.checkDeployReady(ctx, clientset, ns)

	return nil
}

func (k *K8SController) TearDownDeploy(ctx context.Context, nmxctx *Context) error {
	fmt.Println("Get Kubernetes pods")

	var rt = nmxctx.Runtime
	logrus.Debugf("Using kubeconfig: %s\n", rt.Kubeconfig)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", rt.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes clientset: %w", err)
	}

	err = clientset.CoreV1().Namespaces().Delete(ctx, NetmuxName, v1.DeleteOptions{})
	if err != nil {
		logrus.Warnf("error deleting namespace: %s", err.Error())
	}

	err = clientset.RbacV1().ClusterRoles().Delete(ctx, NetmuxName, v1.DeleteOptions{})
	if err != nil {
		logrus.Warnf("error ClusterRole namespace: %s", err.Error())
	}
	err = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, NetmuxName, v1.DeleteOptions{})
	if err != nil {
		logrus.Warnf("error ClusterRole namespace: %s", err.Error())
	}

	return nil
}

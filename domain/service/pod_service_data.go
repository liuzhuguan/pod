package service

import (
	"context"
	"errors"
	"git.imooc.com/coding-535/common"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pod/domain/model"
	"pod/domain/repository"
	"pod/proto/pod"
	"strconv"

	"k8s.io/client-go/kubernetes"
)

type IPodDataService interface {
	AddPod(*model.Pod) (int64, error)
	DeletePod(int64) error
	UpdatePod(*model.Pod) error
	FindPodByID(int64) (*model.Pod, error)
	FindAllPod() ([]model.Pod, error)
	CreateToK8s(*pod.PodInfo) error
	DeleteFromK8s(*model.Pod) error
	UpdateToK8s(*pod.PodInfo) error
}

type PodDataService struct {
	PodRepository repository.IPodRepository
	K8sClientSet  *kubernetes.Clientset
	deployment    *v1.Deployment
}

func NewPodDataService(podRepository repository.IPodRepository, k8sClientSet *kubernetes.Clientset) IPodDataService {
	return &PodDataService{
		PodRepository: podRepository,
		K8sClientSet:  k8sClientSet,
		deployment:    &v1.Deployment{},
	}
}

func (p *PodDataService) AddPod(pod *model.Pod) (int64, error) {
	return p.PodRepository.CreatePod(pod)
}

func (p *PodDataService) DeletePod(podId int64) error {
	return p.PodRepository.DeletePodByID(podId)
}

func (p *PodDataService) UpdatePod(pod *model.Pod) error {
	return p.PodRepository.UpdatePod(pod)
}

func (p *PodDataService) FindPodByID(podId int64) (*model.Pod, error) {
	return p.PodRepository.FindPodByID(podId)
}

func (p *PodDataService) FindAllPod() ([]model.Pod, error) {
	return p.PodRepository.FindAll()
}

func (p *PodDataService) CreateToK8s(podInfo *pod.PodInfo) error {
	p.setDeployment(podInfo)
	// 查询不到现有的pod信息，需要创建
	if _, getErr := p.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Get(context.TODO(), podInfo.PodName, v1meta.GetOptions{}); getErr != nil {
		if _, createErr := p.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Create(context.TODO(), p.deployment, v1meta.CreateOptions{}); createErr != nil {
			common.Error("create pod err :", createErr.Error())
			return createErr
		}
	} else {
		common.Error("create pod err, pod is exist")
		return errors.New("Pod " + podInfo.PodName + " already exist")
	}
	return nil
}

func (p *PodDataService) DeleteFromK8s(podInfo *model.Pod) error {
	if deleteErr := p.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Delete(context.TODO(), podInfo.PodName, v1meta.DeleteOptions{}); deleteErr != nil {
		common.Error(deleteErr)
		return deleteErr
	} else {
		if err := p.DeletePod(podInfo.ID); err != nil {
			common.Error(err)
			return err
		}
		common.Info("删除Pod ID：" + strconv.FormatInt(podInfo.ID, 10) + " 成功！")
	}
	return nil
}

func (p *PodDataService) UpdateToK8s(podInfo *pod.PodInfo) error {
	p.setDeployment(podInfo)
	if _, getErr := p.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Get(context.TODO(), podInfo.PodName, v1meta.GetOptions{}); getErr != nil {
		common.Error(getErr)
		return errors.New("Pod " + podInfo.PodName + " pod not exist")
	} else {
		//如果存在
		if _, updateErr := p.K8sClientSet.AppsV1().Deployments(podInfo.PodNamespace).Update(context.TODO(), p.deployment, v1meta.UpdateOptions{}); updateErr != nil {
			common.Error(updateErr)
			return updateErr
		}
		common.Info(podInfo.PodName + " 更新成功")
		return nil
	}
}

func (p *PodDataService) setDeployment(podInfo *pod.PodInfo) {
	deployment := &v1.Deployment{}

	deployment.TypeMeta = v1meta.TypeMeta{
		Kind:       "deployment",
		APIVersion: "v1",
	}

	deployment.ObjectMeta = v1meta.ObjectMeta{
		Name:      podInfo.PodName,
		Namespace: podInfo.PodNamespace,
		Labels: map[string]string{
			"app-name": podInfo.PodName,
			"author":   "lzg",
		},
	}

	deployment.Name = podInfo.PodName
	deployment.Spec = v1.DeploymentSpec{
		Replicas: &podInfo.PodReplicas,
		Selector: &v1meta.LabelSelector{
			MatchLabels: map[string]string{
				"app-name": podInfo.PodName,
			},
			MatchExpressions: nil,
		},
		Template: v1core.PodTemplateSpec{
			ObjectMeta: v1meta.ObjectMeta{
				Labels: map[string]string{
					"app-name": podInfo.PodName,
				},
			},
			Spec: v1core.PodSpec{
				Containers: []v1core.Container{
					{
						Name:            podInfo.PodName,
						Image:           podInfo.PodImage,
						Ports:           p.getContainerPort(podInfo),
						Env:             p.getEnv(podInfo),
						Resources:       p.getResources(podInfo),
						ImagePullPolicy: p.getImagePullPolicy(podInfo),
					},
				},
			},
		},
		Strategy:                v1.DeploymentStrategy{},
		MinReadySeconds:         0,
		RevisionHistoryLimit:    nil,
		Paused:                  false,
		ProgressDeadlineSeconds: nil,
	}

	p.deployment = deployment
}

func (p *PodDataService) getContainerPort(podInfo *pod.PodInfo) (containerPortList []v1core.ContainerPort) {
	for _, v := range podInfo.PodPort {
		containerPortList = append(containerPortList, v1core.ContainerPort{
			Name:          "port-" + strconv.FormatInt(int64(v.ContainerPort), 10),
			ContainerPort: v.ContainerPort,
			Protocol:      p.getProtocol(v.Protocol),
		})
	}
	return
}

func (p *PodDataService) getProtocol(protocol string) v1core.Protocol {
	switch protocol {
	case "TCP":
		return "TCP"
	case "UDP":
		return "UDP"
	case "SCTP":
		return "SCTP"
	default:
		return "TCP"
	}
}

func (p *PodDataService) getEnv(podInfo *pod.PodInfo) (envList []v1core.EnvVar) {
	for _, v := range podInfo.PodEnv {
		envList = append(envList, v1core.EnvVar{
			Name:      v.EnvKey,
			Value:     v.EnvValue,
			ValueFrom: nil,
		})
	}
	return
}

func (p *PodDataService) getResources(podInfo *pod.PodInfo) (source v1core.ResourceRequirements) {
	//最大能够使用多少资源
	source.Limits = v1core.ResourceList{
		"cpu":    resource.MustParse(strconv.FormatFloat(float64(podInfo.PodCpuMax), 'f', 6, 64)),
		"memory": resource.MustParse(strconv.FormatFloat(float64(podInfo.PodMemoryMax), 'f', 6, 64)),
	}
	//满足最少使用的资源量
	source.Requests = v1core.ResourceList{
		"cpu":    resource.MustParse(strconv.FormatFloat(float64(podInfo.PodCpuMax/2), 'f', 6, 64)),
		"memory": resource.MustParse(strconv.FormatFloat(float64(podInfo.PodMemoryMax/2), 'f', 6, 64)),
	}
	return
}

func (p *PodDataService) getImagePullPolicy(podInfo *pod.PodInfo) v1core.PullPolicy {
	switch podInfo.PodPullPolicy {
	case "Always":
		return "Always"
	case "Never":
		return "Never"
	case "IfNotPresent":
		return "IfNotPresent"
	default:
		return "Always"
	}
}

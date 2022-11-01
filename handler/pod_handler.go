package handler

import (
	"context"
	"git.imooc.com/coding-535/common"
	"github.com/liuzhuguan/pod/domain/model"
	"github.com/liuzhuguan/pod/domain/service"
	"github.com/liuzhuguan/pod/proto/pod"
	"strconv"
)

type PodHandler struct {
	PodDataService service.IPodDataService
}

func (p *PodHandler) AddPod(ctx context.Context, info *pod.PodInfo, response *pod.Response) error {
	common.Info("AddPod, podInfo:", info)

	podModel := &model.Pod{}
	err := common.SwapTo(info, podModel)
	if err != nil {
		common.Error("AddPod marshal error: ", err.Error())
		response.Msg = err.Error()
		return err
	}

	if err := p.PodDataService.CreateToK8s(info); err != nil {
		common.Error("AddPod CreateToK8s error: ", err.Error())
		response.Msg = err.Error()
		return err
	} else {
		podID, err := p.PodDataService.AddPod(podModel)
		if err != nil {
			common.Error("AddPod insert DB error: ", err.Error())
			response.Msg = err.Error()
			return err
		}
		common.Info("Pod 添加成功数据库ID号为：" + strconv.FormatInt(podID, 10))
		response.Msg = "Pod 添加成功数据库ID号为：" + strconv.FormatInt(podID, 10)
	}
	return nil
}

func (p *PodHandler) DeletePod(ctx context.Context, req *pod.PodId, response *pod.Response) error {
	common.Info("DeletePod, podId:", req.Id)

	var (
		podModel *model.Pod
		err      error
	)

	if podModel, err = p.PodDataService.FindPodByID(req.Id); err != nil {
		common.Error("DeletePod FindPodByID error: ", err.Error())
		response.Msg = err.Error()
		return err
	}

	if err = p.PodDataService.DeleteFromK8s(podModel); err != nil {
		common.Error(err)
		return err
	}
	return nil
}

func (p *PodHandler) FindPodByID(ctx context.Context, req *pod.PodId, info *pod.PodInfo) error {
	//查询pod数据
	podModel, err := p.PodDataService.FindPodByID(req.Id)
	if err != nil {
		common.Error(err)
		return err
	}
	err = common.SwapTo(podModel, info)
	if err != nil {
		common.Error(err)
		return err
	}
	return nil
}

func (p *PodHandler) UpdatePod(ctx context.Context, info *pod.PodInfo, response *pod.Response) error {
	//先更新k8s中的pod信息
	err := p.PodDataService.UpdateToK8s(info)
	if err != nil {
		common.Error(err)
		return err
	}
	//查询数据库中的pod
	podModel, err := p.PodDataService.FindPodByID(info.Id)
	if err != nil {
		common.Error(err)
		return err
	}
	err = common.SwapTo(info, podModel)
	if err != nil {
		common.Error(err)
		return err
	}
	return p.PodDataService.UpdatePod(podModel)
}

func (p *PodHandler) FindAllPod(ctx context.Context, all *pod.FindAll, rsp *pod.AllPod) error {
	//查询所有pod
	allPod, err := p.PodDataService.FindAllPod()
	if err != nil {
		common.Error(err)
		return err
	}
	//整理格式
	for _, v := range allPod {
		podInfo := &pod.PodInfo{}
		err := common.SwapTo(v, podInfo)
		if err != nil {
			common.Error(err)
			return err
		}
		rsp.PodInfo = append(rsp.PodInfo, podInfo)
	}
	return nil
}

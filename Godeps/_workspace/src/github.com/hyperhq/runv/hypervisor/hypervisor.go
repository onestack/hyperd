package hypervisor

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/hyperhq/runv/hypervisor/network"
	"github.com/hyperhq/runv/hypervisor/types"
)

func (ctx *VmContext) startSocks() {
	go waitInitReady(ctx)
	go waitPts(ctx)
	if glog.V(1) {
		go watchVmConsole(ctx)
	}
}

func (ctx *VmContext) loop() {
	for ctx.handler != nil {
		ev, ok := <-ctx.Hub
		if !ok {
			glog.Error("hub chan has already been closed")
			break
		} else if ev == nil {
			glog.V(1).Info("got nil event.")
			continue
		}
		glog.V(1).Infof("vm %s: main event loop got message %d(%s)", ctx.Id, ev.Event(), EventString(ev.Event()))
		ctx.handler(ctx, ev)
	}
}

func (ctx *VmContext) Launch() {

	//launch routines
	ctx.startSocks()
	ctx.DCtx.Launch(ctx)

	ctx.loop()
}

func VmAssociate(vmId string, hub chan VmEvent, client chan *types.VmResponse, pack []byte) (*VmContext, error) {

	if glog.V(1) {
		glog.Infof("VM %s trying to reload with serialized data: %s", vmId, string(pack))
	}

	pinfo, err := vmDeserialize(pack)
	if err != nil {
		return nil, err
	}

	if pinfo.Id != vmId {
		return nil, fmt.Errorf("VM ID mismatch, %v vs %v", vmId, pinfo.Id)
	}

	context, err := pinfo.vmContext(hub, client)
	if err != nil {
		return nil, err
	}

	context.DCtx.Associate(context)

	go waitPts(context)
	go connectToInit(context)
	if glog.V(1) {
		go watchVmConsole(context)
	}

	context.Become(stateRunning, StateRunning)

	//for _, c := range context.vmSpec.Containers {
	//	context.ptys.ptyConnect(true, c.Process.Terminal, c.Process.Stdio, c.Process.Stderr, nil)
	//	context.ptys.startStdin(c.Process.Stdio, c.Process.Terminal)
	//}

	go context.loop()
	return context, nil
}

func InitNetwork(bIface, bIP string, disableIptables bool) error {
	if HDriver.BuildinNetwork() {
		return HDriver.InitNetwork(bIface, bIP, disableIptables)
	}

	return network.InitNetwork(bIface, bIP, disableIptables)
}

func SupportLazyMode() bool {
	return HDriver.SupportLazyMode()
}

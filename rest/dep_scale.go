package rest

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

func (s *Server) newScaleUpHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodeName := params.ByName("nodeName")

	kv := s.consulClient.KV()

	if len(nodeName) == 0 {
		log.Panic("You must provide a nodename")
	} else if ok, err := deployments.HasScalableCapability(kv, id, nodeName); err != nil {
		log.Panic(err)
	} else if !ok {
		log.Panic("The given nodename must be scalable")
	}

	var positiveDelta uint32

	if value, ok := r.URL.Query()["add"]; ok {
		if val, err := strconv.Atoi(value[0]); err != nil {
			log.Panic(err)
		} else if val > 0 {
			positiveDelta = uint32(val)
		} else {
			log.Panic("You need to provide a positive non zero value as add parameter")
		}
	} else {
		log.Panic("You need to provide a add parameter")
	}

	_, maxInstances, err := deployments.GetMaxNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}
	_, currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodeName)
	if err != nil {
		log.Panic(err)
	}

	if currentNbInstance+positiveDelta > maxInstances {
		log.Debug("The delta is too high, the max instances number is choosen")
		positiveDelta = maxInstances - currentNbInstance
	}

	// NOTE: all those stuff on requirements should probably go into deployments.CreateNewNodeStackInstances
	var req []string

	req, err = deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "network")
	if err != nil {
		log.Panic(err)
	}

	if tmp, err := deployments.GetRequirementsKeysByNameForNode(kv, id, nodeName, "local_storage"); err != nil {
		log.Panic(err)
	} else {
		req = append(req, tmp...)
	}

	var reqNameArr []string
	for _, reqPath := range req {
		reqName, _, err := kv.Get(path.Join(reqPath, "node"), nil)
		if err != nil {
			log.Panic(err)
		}
		reqNameArr = append(reqNameArr, string(reqName.Value))
		// TODO: for now the link between the requirement instance ID and the node instance ID is a kind of black magic. We should found a way to make it rational...
		_, err = deployments.CreateNewNodeStackInstances(kv, id, string(reqName.Value), int(positiveDelta))
		if err != nil {
			log.Panic(err)
		}
	}

	newInstanceID, err := deployments.CreateNewNodeStackInstances(kv, id, nodeName, int(positiveDelta))
	if err != nil {
		log.Panic(err)
	}

	err = deployments.SetNbInstancesForNode(kv, id, nodeName, currentNbInstance+positiveDelta)
	if err != nil {
		log.Panic(err)
	}

	data := make(map[string]string)

	data["node"] = nodeName
	data["new_instances_ids"] = strings.Join(newInstanceID, ",")
	data["current_instances_number"] = strconv.Itoa(int(currentNbInstance + positiveDelta))
	data["req"] = strings.Join(reqNameArr, ",")

	taskID, err := s.tasksCollector.RegisterTaskWithData(id, tasks.ScaleUp, data)

	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskID))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) newScaleDownHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value("params").(httprouter.Params)
	id := params.ByName("id")
	nodename := params.ByName("nodeName")

	kv := s.consulClient.KV()

	if len(nodename) == 0 {
		log.Panic("You must provide a nodename")
	} else if ok, err := deployments.HasScalableCapability(kv, id, nodename); err != nil {
		log.Panic(err)
	} else if !ok {
		log.Panic("The given nodename must be scalable")
	}

	var delta uint32

	if value, ok := r.URL.Query()["remove"]; ok {
		if val, err := strconv.Atoi(value[0]); err != nil {
			log.Panic(err)
		} else if val > 0 {
			delta = uint32(val)
		} else {
			log.Panic("You need to provide a positive non zero value as remove parameter")
		}
	} else {
		log.Panic("You need to provide a remove parameter")
	}

	_, minInstances, err := deployments.GetMinNbInstancesForNode(kv, id, nodename)
	if err != nil {
		log.Panic(err)
	}
	_, currentNbInstance, err := deployments.GetNbInstancesForNode(kv, id, nodename)
	if err != nil {
		log.Panic(err)
	}

	if currentNbInstance-delta < minInstances {
		log.Debug("The delta is too low, the min instances number is choosen")
		delta = minInstances - currentNbInstance
	}

	depPath := path.Join(consulutil.DeploymentKVPrefix, id)
	instancesPath := path.Join(depPath, "topology", "instances")

	var req []string

	req, err = deployments.GetRequirementsKeysByNameForNode(kv, id, nodename, "network")
	if err != nil {
		log.Panic(err)
	}

	if tmp, err := deployments.GetRequirementsKeysByNameForNode(kv, id, nodename, "local_storage"); err != nil {
		log.Panic(err)
	} else {
		req = append(req, tmp...)
	}

	var reqNameArr []string
	for _, reqPath := range req {
		reqName, _, err := kv.Get(path.Join(reqPath, "node"), nil)
		if err != nil {
			log.Panic(err)
		}
		reqNameArr = append(reqNameArr, string(reqName.Value))
		for i := currentNbInstance - 1; i > currentNbInstance-1-delta; i-- {
			_, err := kv.DeleteTree(path.Join(instancesPath, string(reqName.Value), strconv.FormatUint(uint64(i), 10))+"/", nil)
			if err != nil {
				log.Panic(err)
			}
		}
	}

	newInstanceId := []string{}
	for i := currentNbInstance - 1; i > currentNbInstance-1-delta; i-- {
		newInstanceId = append(newInstanceId, strconv.Itoa(int(i)))
	}

	err = deployments.SetNbInstancesForNode(kv, id, nodename, currentNbInstance-delta)
	if err != nil {
		log.Panic(err)
	}

	data := make(map[string]string)

	data["node"] = nodename
	data["new_instances_ids"] = strings.Join(newInstanceId, ",")
	data["current_instances_number"] = strconv.Itoa(int(currentNbInstance - delta))
	data["req"] = strings.Join(reqNameArr, ",")

	destroy, lock, taskId, err := s.tasksCollector.RegisterTaskWithoutDestroyLock(id, tasks.ScaleDown, data)

	if err != nil {
		if tasks.IsAnotherLivingTaskAlreadyExistsError(err) {
			WriteError(w, r, NewBadRequestError(err))
			return
		}
		log.Panic(err)
	}

	destroy(lock, taskId, id)

	w.Header().Set("Location", fmt.Sprintf("/deployments/%s/tasks/%s", id, taskId))
	w.WriteHeader(http.StatusAccepted)
}

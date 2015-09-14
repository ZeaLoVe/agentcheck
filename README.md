# agentcheck

A plugin to agent alive check of Open-falcon 

这是一个临时方案，用来实现open-falcon的 agent检测。

主要方法是通过hbs上开一个 API，获取所有上报心跳的机器列表。然后通过这个列表中的endpoint去获取agent.alive的last数据，然后通过
对比last数据中的timestamp，相差在两个周期（60*2）的，则填充一个agent.alive = 0的指标，通过本机的agent提供的v1/push 的API将指标上报。这样做的好处是可以通过agent.alive！=1来报警，同时这个组件的部署也很灵活，可以通过变成一个单独的进程去跑，也可以当成插件。当成组件的话，推荐把log注释去掉。当成
插件的时候，最后会上报一个agent.not_alive.num的指标，endpoint就是插件的执行机器的hostname（如果hostname不是agent采集的endpoint）那么此处需要修改。这里有一个假设是，agent的汇报周期都是60，如果你设置的agent汇报周期有变，需要适当调整逻辑，此方案仅是临时过渡方案。

HBS：添加的API代码如下：

	
	type Endpoint struct {
		
		Endpoint string `json:"endpoint,omitempty"`
		
	}

	//get ,API of all endpoints ,use in agent alive check.

	http.HandleFunc("/all/endpoints", func(w http.ResponseWriter, r *http.Request) {
		
		var endpoints []Endpoint
		
		var endpoint Endpoint
		
		cache.HostMap.Lock()
		
		for host, _ := range cache.HostMap.M {
		
			endpoint.Endpoint = host
			
			endpoints = append(endpoints, endpoint)
			
		}
		
		cache.HostMap.Unlock()
	
		RenderJson(w, endpoints)
		
	})
	
	
cfg.json里配置的是三个API的地址。如果不配置，默认的IP都是127.0.0.1

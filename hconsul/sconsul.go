package hconsul

import (
	"fmt"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/hashicorp/consul/api"
)

func RegisterServiceInfo(cp xconfig.IConfigProvider) {
	// Read configuration
	consulAddr := cp.GetString("Consul.Addr")
	consulToken := cp.GetString("Consul.Token")
	serviceName := cp.GetString("Consul.Service.Name")
	serviceCheckTimeout := cp.GetString("Consul.Service.Check.Timeout")
	serviceCheckInterval := cp.GetString("Consul.Service.Check.Interval")
	serviceHost := cp.GetString("Consul.Service.Host")
	servicePort := cp.GetInt("Consul.Service.Port")
	serviceID := fmt.Sprintf("%v:%v", serviceHost, servicePort)

	// Service center client
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddr
	consulConfig.Token = consulToken
	consulClient, err := api.NewClient(consulConfig)
	xerr.FatalIfErr(err)
	consulAgent := consulClient.Agent()

	// Register service in service center
	err = consulAgent.ServiceRegister(&api.AgentServiceRegistration{
		ID:   serviceID,   // Service node name
		Name: serviceName, // Service name
		// Tags:    r.Tag,                                        // Tags, can be empty
		Address: serviceHost, // Service IP
		Port:    servicePort, // Service port
		Check: &api.AgentServiceCheck{ // Health check
			Interval:                       serviceCheckInterval, // Health check interval
			TCP:                            fmt.Sprintf("%s:%d", serviceHost, servicePort),
			DeregisterCriticalServiceAfter: serviceCheckTimeout, // Deregistration time, equivalent to expiration time
		},
	})
	xerr.FatalIfErr(err)
}

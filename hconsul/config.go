package hconsul

import "github.com/DreamvatLab/go/xconfig"

type ConsulConfig struct {
	Addr  string
	Token string
}

func GetConsulConfig(configProvider xconfig.IConfigProvider) (*ConsulConfig, error) {
	var consulConfig *ConsulConfig
	err := configProvider.GetStruct("Consul", &consulConfig)
	if err != nil {
		return nil, err
	}
	return consulConfig, nil
}

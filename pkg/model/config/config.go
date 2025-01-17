package config

import (
	"encoding/json"

	provider_config "github.com/nchc-ai/oauth-provider/pkg/config"
	"github.com/spf13/viper"
)

type Config struct {
	APIConfig     *APIConfig     `json:"api-server"`
	DBConfig      *DBConfig      `json:"database"`
	K8SConfig     *K8SConfig     `json:"kubernetes"`
	RFStackConfig *RFStackConfig `json:"rfstack"`
	RedisConfig   *RedisConfig   `json:"redis"`
}

// Snake-Case JSON Fields Ignored by UnmarshalKey(), so we write our unmarsh function
// https://github.com/spf13/viper/issues/125
func UnmarshConfig(v *viper.Viper) (*Config, error) {

	// convert each provider config json to ProviderConfig struct
	// we need two phase conversion, map[string]interface{} -> json -> struct
	// https://www.cnblogs.com/liang1101/p/6741262.html
	k8sconfig := K8SConfig{}
	err := v.UnmarshalKey("kubernetes", &k8sconfig)
	if err != nil {
		return nil, err
	}

	redisConfig := RedisConfig{}
	err = v.UnmarshalKey("redis", &redisConfig)
	if err != nil {
		return nil, err
	}

	dbconfig := DBConfig{}
	err = v.UnmarshalKey("database", &dbconfig)
	if err != nil {
		return nil, err
	}

	stackConfig := RFStackConfig{}
	err = v.UnmarshalKey("rfstack", &stackConfig)
	if err != nil {
		return nil, err
	}

	apiconfig := APIConfig{}
	err = v.UnmarshalKey("api-server", &apiconfig)

	if err != nil {
		return nil, err
	}

	providerConfigstr := v.GetStringMapString("api-server.provider")
	var vconf provider_config.ProviderConfig

	// map[string]string -> json
	jsonStr, err := json.Marshal(providerConfigstr)
	if err != nil {
		return nil, err
	}

	// json -> struct
	err = json.Unmarshal([]byte(jsonStr), &vconf)
	if err != nil {
		return nil, err
	}

	apiconfig.Provider = vconf

	config := Config{
		K8SConfig:     &k8sconfig,
		DBConfig:      &dbconfig,
		APIConfig:     &apiconfig,
		RFStackConfig: &stackConfig,
		RedisConfig:   &redisConfig,
	}
	return &config, nil
}

type APIConfig struct {
	IsOutsideCluster bool                           `json:"isOutsideCluster"`
	Port             int                            `json:"port"`
	EnableSecureAPI  bool                           `json:"enableSecureAPI"`
	Provider         provider_config.ProviderConfig `json:"provider"`
	NamespacePrefix  string                         `json:"namespacePrefix"`
	UidRange         string                         `json:"uidRange"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type K8SConfig struct {
	KUBECONFIG   string `json:"kubeconfig"`
	NodePortDNS  string `json:"nodeportDNS"`
	StorageClass string `json:"storageclass"`
}

type PConfig struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type RFStackConfig struct {
	Enable bool   `json:"enable"`
	Url    string `json:"url"`
}

type RedisConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

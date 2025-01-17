package main

import (
	"flag"

	log "github.com/golang/glog"
	_ "github.com/nchc-ai/backend-api/docs"
	"github.com/nchc-ai/backend-api/pkg/api"
	"github.com/nchc-ai/backend-api/pkg/consts"
	cm "github.com/nchc-ai/backend-api/pkg/model/config"
	"github.com/spf13/viper"
)

// @title AI Train API
// @version 0.2
// @description AI Train API.

// @host localhost:38080
// @BasePath /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {

	configPath := flag.String("conf", "", "The file path to a config file")
	flag.Parse()

	config, err := ReadConfig(*configPath)
	if err != nil {
		log.Fatalf("Unable to read configure file: %s", err.Error())
	}

	conf, err := cm.UnmarshConfig(config)
	if err != nil {
		log.Fatalf("Unable to unmarshal configure file: %s", err.Error())
	}

	consts.Init(conf.APIConfig.NamespacePrefix)

	server := api.NewAPIServer(conf)
	if server == nil {
		log.Fatalf("Create api server fail, Stop!!")
		return
	}

	log.Info("Start API Server")
	err = server.RunServer(conf.APIConfig.Port)
	if err != nil {
		log.Fatalf("start api server error: %s", err.Error())
	}

}

func ReadConfig(fileConfig string) (*viper.Viper, error) {
	viper := viper.New()
	viper.SetConfigType("json")

	if fileConfig == "" {
		viper.SetConfigName("api-config")
		viper.AddConfigPath("/etc/api-server")
	} else {
		viper.SetConfigFile(fileConfig)
	}

	// overwrite by file
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return viper, nil
}

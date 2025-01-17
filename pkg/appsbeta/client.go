package beta

import (
	"github.com/dghubble/sling"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/apps"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/config"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/nchc-ai/oauth-provider/pkg/provider"
	"github.com/nitishm/go-rejson/v4"
	"k8s.io/client-go/kubernetes"
)

type BetaClient struct {
	classroom apps.ClassroomInterface
	course    apps.CourseInterface
	dataset   apps.DatasetInterface
	health    apps.HealthInterface
	image     apps.ImageInterface
	job       apps.JobInterface
	proxy     apps.ProxyInterface
	user      apps.UserInterface
}

func NewClient(kclient *kubernetes.Clientset, crdclient *versioned.Clientset,
	config *config.Config, db *gorm.DB, provider provider.Provider, rh *rejson.Handler) *BetaClient {

	var rfstackbase *sling.Sling
	if config.RFStackConfig.Enable == true {
		log.Info("Enable VM course function, create rfStack client")
		rfstackbase = sling.New().Base(config.RFStackConfig.Url)
	} else {
		rfstackbase = nil
	}

	return &BetaClient{
		classroom: &Classroom{
			DB:              db,
			KClientSet:      kclient,
			CourseCrdClient: crdclient,
			Config:          config,
		},

		course: &Course{
			DB:              db,
			Redis:           rh,
			CourseCrdClient: crdclient,
			rfStackBase:     rfstackbase,
		},

		dataset: &Dataset{
			Config:     config,
			KClientSet: kclient,
		},

		health: &Health{
			KClientSet: kclient,
			DB:         db,
		},

		image: &Image{
			provider: provider,
			db:       db,
		},

		job: &Job{
			DB:              db,
			redis:           rh,
			CourseCrdClient: crdclient,
			config:          config,
			rfStackBase:     rfstackbase,
			StopChanMap:     make(map[string]chan string),
		},

		proxy: &Proxy{
			provider: provider,
			db:       db,
			config:   config,
		},

		user: &User{
			db: db,
		},
	}
}

func (c *BetaClient) Classroom() apps.ClassroomInterface {
	return c.classroom
}

func (c *BetaClient) Course() apps.CourseInterface {
	return c.course
}

func (c *BetaClient) Dataset() apps.DatasetInterface {
	return c.dataset
}

func (c *BetaClient) Health() apps.HealthInterface {
	return c.health
}

func (c *BetaClient) Image() apps.ImageInterface {
	return c.image
}

func (c *BetaClient) Job() apps.JobInterface {
	return c.job
}

func (c *BetaClient) Proxy() apps.ProxyInterface {
	return c.proxy
}

func (c *BetaClient) User() apps.UserInterface {
	return c.user
}

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	goredis "github.com/go-redis/redis/v8"
	beta "github.com/nchc-ai/backend-api/pkg/appsbeta"
	"github.com/nchc-ai/backend-api/pkg/model/db"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/nitishm/go-rejson/v4"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/nchc-ai/backend-api/pkg/model/config"
	github_provider "github.com/nchc-ai/github-oauth-provider/pkg/provider"
	go_provider "github.com/nchc-ai/go-oauth-provider/pkg/provider"
	google_provider "github.com/nchc-ai/google-oauth-provider/pkg/provider"
	provider_inerface "github.com/nchc-ai/oauth-provider/pkg/provider"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

type APIServer struct {
	router                    *gin.Engine
	clientSet                 *ClientSet
	db                        *gorm.DB
	redis                     *rejson.Handler
	isSecure                  bool
	authMiddleware            gin.HandlerFunc
	corsMiddleware            gin.HandlerFunc
	addProviderNameMiddleware gin.HandlerFunc
}

func NewAPIServer(config *config.Config) *APIServer {
	kclient, crdclient, err := NewKClients(config)
	if err != nil {
		log.Fatalf("Create kubernetes client fail, Stop...: %s", err.Error())
		return nil
	}
	log.Info("Create Kubernetes Client")

	dbclient, err := NewDBClient(config)
	if err != nil {
		log.Fatalf("Create database client fail, Stop...: %s", err.Error())
		return nil
	}
	log.Info("Create Database Client")

	redisAddr := fmt.Sprintf("%s:%d", config.RedisConfig.Host, config.RedisConfig.Port)
	rh := rejson.NewReJSONHandler()
	cli := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	rh.SetGoRedisClient(cli)

	vconf := config.APIConfig.Provider
	var providerProxy provider_inerface.Provider
	switch oauthProvider := vconf.Type; oauthProvider {
	case go_provider.GO_OAUTH:
		log.Infof("Create %s Provider Proxy", go_provider.GO_OAUTH)
		providerProxy = go_provider.NewGoAuthProvider(vconf)
	case github_provider.GITHUB_OAUTH:
		log.Infof("Create %s Provider Proxy", github_provider.GITHUB_OAUTH)
		providerProxy = github_provider.NewGitAuthProvider(vconf)
	case google_provider.GOOGLE_OAUTH:
		log.Infof("Create %s Provider Proxy", google_provider.GOOGLE_OAUTH)
		providerProxy = google_provider.NewGoogleAuthProvider(vconf)
	default:
		log.Warning(fmt.Sprintf("%s is a not supported provider type", oauthProvider))
	}

	server := &APIServer{
		db:        dbclient,
		redis:     rh,
		clientSet: NewClientset(kclient, crdclient, config, dbclient, providerProxy, rh),
		router:    gin.Default(),
		isSecure:  config.APIConfig.EnableSecureAPI,

		corsMiddleware: func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
			c.Next()
		},
		authMiddleware:            authMiddleware(providerProxy),
		addProviderNameMiddleware: addProviderNameMiddleware(providerProxy),
	}

	log.Info("Check pending jobRoute after api server restart")
	go server.resume(crdclient)

	return server
}

func (s *APIServer) RunServer(port int) error {

	defer s.db.Close()

	// add middleware
	s.router.Use(s.corsMiddleware)
	if s.isSecure {
		s.router.Use(s.addProviderNameMiddleware)
	}
	// add route
	s.addAPIRoute(s.isSecure)
	s.addSwaggerRoute()

	err := s.router.Run(":" + strconv.Itoa(port))
	if err != nil {
		return err
	}
	return nil
}

func (s *APIServer) addAPIRoute(isSecure bool) {
	s.courseRoute(isSecure)
	s.jobRoute(isSecure)
	s.classroomRoute(isSecure)
	s.datasetRoute(isSecure)
	s.healthRoute(isSecure)
	s.proxyRoute(isSecure)
	s.imageRoute(isSecure)
	s.userRoute(isSecure)
}

func (s *APIServer) courseRoute(isSecure bool) {
	course := s.router.Group("/api").Group("/beta").Group("/course")
	{
		course.GET("/level/:level", s.Beta().Course().ListLevelCourse)
		course.GET("/list", s.Beta().Course().ListAllCourse)
		course.GET("/namelist", s.Beta().Course().CourseNameList)
		course.POST("/search", s.Beta().Course().SearchCourse)
		course.OPTIONS("/level/:level", handleOption)
		course.OPTIONS("/create", handleOption)
		course.OPTIONS("/list", handleOption)
		course.OPTIONS("/delete/:id", handleOption)
		course.OPTIONS("/get/:id", handleOption)
		course.OPTIONS("/search", handleOption)
		course.OPTIONS("/update", handleOption)
		course.OPTIONS("/namelist", handleOption)

		if !isSecure {
			course.POST("/create", s.Beta().Course().Add)
			course.POST("/list", s.Beta().Course().ListUserCourse)
			course.DELETE("/delete/:id", s.Beta().Course().Delete)
			course.GET("/get/:id", s.Beta().Course().Get)
			course.PUT("/update", s.Beta().Course().Update)
		}
	}

	if isSecure {
		courseAuth := s.router.Group("/api").Group("/beta").Group("/course").Use(s.authMiddleware)
		{
			courseAuth.POST("/create", s.Beta().Course().Add)
			courseAuth.POST("/list", s.Beta().Course().ListUserCourse)
			courseAuth.DELETE("/delete/:id", s.Beta().Course().Delete)
			courseAuth.GET("/get/:id", s.Beta().Course().Get)
			courseAuth.PUT("/update", s.Beta().Course().Update)
		}
	}

}

func (s *APIServer) jobRoute(isSecure bool) {
	jobBeta := s.router.Group("/api").Group("/beta").Group("/job")
	{
		jobBeta.OPTIONS("/list", handleOption)
		jobBeta.OPTIONS("/delete/:id", handleOption)
		jobBeta.OPTIONS("/launch", handleOption)

		if !isSecure {
			jobBeta.POST("/list", s.Beta().Job().List)
			jobBeta.DELETE("/delete/:id", s.Beta().Job().Delete)
			jobBeta.POST("/launch", s.Beta().Job().Launch)
		}
	}

	if isSecure {
		jobBetaAuth := s.router.Group("/api").Group("/beta").Group("/job").Use(s.authMiddleware)
		{
			jobBetaAuth.POST("/list", s.Beta().Job().List)
			jobBetaAuth.DELETE("/delete/:id", s.Beta().Job().Delete)
			jobBetaAuth.POST("/launch", s.Beta().Job().Launch)
		}
	}
}

func (s *APIServer) classroomRoute(isSecure bool) {
	classroomBeta := s.router.Group("/api").Group("/beta").Group("/classroom")
	{
		classroomBeta.OPTIONS("/list", handleOption)
		classroomBeta.OPTIONS("/delete/:id", handleOption)
		classroomBeta.OPTIONS("/create", handleOption)
		classroomBeta.OPTIONS("/upload", handleOption)
		classroomBeta.OPTIONS("/get/:id", handleOption)
		classroomBeta.OPTIONS("/update", handleOption)

		if !isSecure {
			classroomBeta.POST("/list", s.Beta().Classroom().List)
			classroomBeta.GET("/list", s.Beta().Classroom().ListAll)
			classroomBeta.DELETE("/delete/:id", s.Beta().Classroom().Delete)
			classroomBeta.POST("/create", s.Beta().Classroom().Add)
			classroomBeta.POST("/upload", s.Beta().Classroom().UploadUserAccount)
			classroomBeta.GET("/get/:id", s.Beta().Classroom().Get)
			classroomBeta.PUT("/update", s.Beta().Classroom().Update)
		}
	}

	if isSecure {
		classroomBetaAuth := s.router.Group("/api").Group("/beta").Group("/classroom").Use(s.authMiddleware)
		{
			classroomBetaAuth.POST("/list", s.Beta().Classroom().List)
			classroomBetaAuth.GET("/list", s.Beta().Classroom().ListAll)
			classroomBetaAuth.DELETE("/delete/:id", s.Beta().Classroom().Delete)
			classroomBetaAuth.POST("/create", s.Beta().Classroom().Add)
			classroomBetaAuth.POST("/upload", s.Beta().Classroom().UploadUserAccount)
			classroomBetaAuth.GET("/get/:id", s.Beta().Classroom().Get)
			classroomBetaAuth.PUT("/update", s.Beta().Classroom().Update)
		}
	}
}

func (s *APIServer) datasetRoute(isSecure bool) {
	dataset := s.router.Group("/api").Group("/beta").Group("/datasets")
	{
		dataset.OPTIONS("/", handleOption)

		if !isSecure {
			dataset.GET("/", s.Beta().Dataset().List)
		}
	}

	if isSecure {
		datasetAuth := s.router.Group("/api").Group("/beta").Group("/datasets").Use(s.authMiddleware)
		{
			datasetAuth.GET("/", s.Beta().Dataset().List)
		}
	}
}

func (s *APIServer) healthRoute(isSecure bool) {
	health := s.router.Group("/api").Group("/beta").Group("/health")
	{
		health.GET("/kubernetes", s.Beta().Health().CheckK8s)
		health.POST("/database", s.Beta().Health().CheckDatabase)
		health.OPTIONS("/kubernetes", handleOption)
		health.OPTIONS("/database", handleOption)
		health.OPTIONS("/kubernetesAuth", handleOption)
		health.OPTIONS("/databaseAuth", handleOption)

		if !isSecure {
			health.GET("/kubernetesAuth", s.Beta().Health().CheckK8sAuth)
			health.POST("/databaseAuth", s.Beta().Health().CheckDatabaseAuth)
		}
	}

	if isSecure {
		healthAuth := s.router.Group("/api").Group("/beta").Group("/health").Use(s.authMiddleware)
		{
			healthAuth.GET("/kubernetesAuth", s.Beta().Health().CheckK8sAuth)
			healthAuth.POST("/databaseAuth", s.Beta().Health().CheckDatabaseAuth)
		}
	}
}

func (s *APIServer) proxyRoute(isSecure bool) {
	proxy := s.router.Group("/api").Group("/beta").Group("/proxy")
	{
		proxy.POST("/token", s.Beta().Proxy().GetToken)
		proxy.POST("/register", s.Beta().Proxy().RegisterUser)
		proxy.POST("/refresh", s.Beta().Proxy().RefreshToken)
		proxy.POST("/introspection", s.Beta().Proxy().Introspection)
		proxy.OPTIONS("/introspection", handleOption)
		proxy.OPTIONS("/logout", handleOption)
		proxy.OPTIONS("/register", handleOption)
		proxy.OPTIONS("/update", handleOption)
		proxy.OPTIONS("/changePW", handleOption)
		proxy.OPTIONS("/query", handleOption)
		proxy.OPTIONS("/token", handleOption)
		proxy.OPTIONS("/refresh", handleOption)

		if !isSecure {
			proxy.POST("/logout", s.Beta().Proxy().Logout)
			proxy.POST("/update", s.Beta().Proxy().UpdateUserBasicInfo)
			proxy.POST("/changePW", s.Beta().Proxy().ChangeUserPassword)
			proxy.GET("/query", s.Beta().Proxy().QueryUser)
		}
	}

	if isSecure {
		proxyAuth := s.router.Group("/api").Group("/beta").Group("/proxy").Use(s.authMiddleware)
		{
			proxyAuth.POST("/logout", s.Beta().Proxy().Logout)
			proxyAuth.POST("/update", s.Beta().Proxy().UpdateUserBasicInfo)
			proxyAuth.POST("/changePW", s.Beta().Proxy().ChangeUserPassword)
			proxyAuth.GET("/query", s.Beta().Proxy().QueryUser)
		}
	}
}

func (s *APIServer) imageRoute(isSecure bool) {
	image := s.router.Group("/api").Group("/beta").Group("/images")
	{
		image.OPTIONS("/", handleOption)
		image.OPTIONS("/commit", handleOption)

		if !isSecure {
			image.GET("/", s.Beta().Image().List)
			image.POST("/commit", s.Beta().Image().Commit)
		}
	}

	if isSecure {
		imageAuth := s.router.Group("/api").Group("/beta").Group("/images").Use(s.authMiddleware)
		{
			imageAuth.GET("/", s.Beta().Image().List)
			imageAuth.POST("/commit", s.Beta().Image().Commit)
		}
	}
}

func (s *APIServer) userRoute(isSecure bool) {
	user := s.router.Group("/api").Group("/beta").Group("/user")
	{
		user.OPTIONS("/role/:roleid", handleOption)
		if !isSecure {
			user.GET("/role/:roleid", s.Beta().User().RoleList)
		}
	}
	if isSecure {
		userAuth := s.router.Group("/api").Group("/beta").Group("/user").Use(s.authMiddleware)
		{
			userAuth.GET("/role/:roleid", s.Beta().User().RoleList)
		}
	}
}

func (s *APIServer) addSwaggerRoute() {
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *APIServer) resume(crdClient *versioned.Clientset) {
	job := db.Job{
		Status: beta.JobStatueReady,
	}
	resultJobs := []db.Job{}
	if err := s.db.Not(&job).Find(&resultJobs).Error; err != nil {
		log.Warningf("find Job in Pending state fail: %s", err.Error())
		return
	}

	b := s.clientSet.BetaClient.Job().(*beta.Job)
	for _, j := range resultJobs {
		log.Infof("start check Course CRD {%s}", j.ID)
		b.StopChanMap[j.ID] = make(chan string, 5)
		b.StopChanMap[j.ID] <- ""
		go beta.CheckCourseCRDStatus(s.db, s.redis, crdClient, *j.ClassroomID, j.ID, b.StopChanMap[j.ID])
	}
}

func (s *APIServer) Beta() *beta.BetaClient {
	return s.clientSet.BetaClient
}

// PRIVATE util func
func authMiddleware(p provider_inerface.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Error("Authorization header is missing")
			beta.RespondWithError(c, http.StatusUnauthorized, "Authorization header is missing")
			return
		}

		bearerToken := strings.Split(authHeader, " ")

		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			log.Errorf("Authorization header is not Bearer Token format or token is missing: %s", authHeader)
			beta.RespondWithError(c, http.StatusUnauthorized, "Authorization header is not Bearer Token format or token is missing")
			return
		}

		var validated bool
		var err error

		token := bearerToken[1]
		validated, err = p.Validate(token)

		if err != nil && err.Error() == "Access token expired" {
			log.Error("Access token expired")
			beta.RespondWithError(c, http.StatusForbidden, "登入過久超時，請重新登入")
			return
		}

		if err != nil && err.Error() == "Access token not found" {
			log.Error("Access token not found")
			beta.RespondWithError(c, http.StatusForbidden, "您已在別的設備登出，請再次登入")
			return
		}

		if err != nil {
			log.Errorf("verify token fail: %s", err.Error())
			beta.RespondWithError(c, http.StatusInternalServerError, "verify token fail: %s", err.Error())
			return
		}

		if !validated {
			beta.RespondWithError(c, http.StatusForbidden, "Invalid API token")
			return
		}
		c.Next()
	}
}

func addProviderNameMiddleware(p provider_inerface.Provider) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := fmt.Sprintf("%s:%s", p.Type(), p.Name())
		c.Set("Provider", provider)
		c.Next()
	}
}

func handleOption(c *gin.Context) {
	//	setup headers
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Access-Control-Allow-Origin, Access-Control-Allow-Credentials")
	c.Status(http.StatusOK)
}

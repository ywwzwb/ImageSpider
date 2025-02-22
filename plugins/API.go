package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"ywwzwb/imagespider/interfaces"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

type API struct {
	app       interfaces.IApplication
	router    *gin.Engine
	server    *http.Server
	dbService interfaces.IDBService
}

func newAPI() *API {
	API := API{}
	return &API
}

func init() {
	API := newAPI()
	interfaces.Plugins[API.ID()] = API
}

func (s *API) Name() string {
	return "API"
}
func (s *API) ID() string {
	return "API"
}
func (s *API) Load(app interfaces.IApplication) error {
	s.app = app

	dbService, err := app.GetService(s.ID(), DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	s.dbService = dbService.(interfaces.IDBService)
	s.router = gin.Default()
	s.router.Use(sloggin.New(slog.Default()))
	s.server = &http.Server{
		Addr:    ":" + strconv.FormatInt(int64(app.GetAppConfig().APIConfig.Port), 10),
		Handler: s.router,
	}
	go func() {
		// 服务连接
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to listen", "error", err)
		}
	}()
	s.router.GET("/:sourceid/tags", s.listAllTags)
	s.router.GET("/:sourceid/images", s.listImages)
	s.router.GET("/:sourceid/image/:id", s.getImage)
	s.router.Static("/image", s.app.GetAppConfig().ImageDir)
	return nil
}
func (s *API) Unload() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown server", "error", err)
	}
	slog.Error("Server stopped")
}
func (s *API) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	return nil, fmt.Errorf("unsupported service")
}
func (s *API) listAllTags(c *gin.Context) {
	sourceid := c.Param("sourceid")
	var offset int64 = 0
	var limit int64 = 50
	if v, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32); err == nil {
		offset = v
	}
	if v, err := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32); err == nil {
		limit = v
	}
	if tagList, err := s.dbService.ListNotGroupTags(sourceid, offset, limit); err == nil {
		c.JSON(http.StatusOK, tagList)
	} else {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
}
func (s *API) listImages(c *gin.Context) {
	sourceid := c.Param("sourceid")
	var offset int64 = 0
	var limit int64 = 50
	if v, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32); err == nil {
		offset = v
	}
	if v, err := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32); err == nil {
		limit = v
	}
	tags := c.QueryArray("tag")
	if imagList, err := s.dbService.ListDownloadedImageOfTags(sourceid, tags, offset, limit); err == nil {
		c.JSON(http.StatusOK, imagList)
	} else {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
}
func (s *API) getImage(c *gin.Context) {
	sourceid := c.Param("sourceid")
	metaID := c.Param("id")
	if imagList, err := s.dbService.GetImageMeta(sourceid, metaID); err == nil {
		c.JSON(http.StatusOK, imagList)
	} else if _, ok := err.(DBCommonError); ok {
		c.JSON(http.StatusNotFound, map[string]any{"error": err.Error()})
	} else {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
}

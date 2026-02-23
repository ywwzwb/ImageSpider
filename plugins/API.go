package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"ywwzwb/imagespider/embed"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"

	"github.com/gin-contrib/gzip"
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
	s.router.Use(gzip.Gzip(gzip.DefaultCompression))
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
	api := s.router.Group("api")
	api.GET("/sources", s.listSources)
	api.GET("/:sourceid/tags", s.listAllTags)
	api.GET("/:sourceid/images", s.listImages)
	api.GET("/:sourceid/image/:id", s.getImage)
	api.DELETE("/:sourceid/images", s.batchDeleteImages)
	api.POST("/:sourceid/images/redownload", s.batchRedownloadImages)
	api.POST("/:sourceid/images/status", s.batchUpdateImageStatus)
	api.POST("/:sourceid/tags/:tag/cover", s.setTagCover)
	s.router.Static("/image", s.app.GetAppConfig().ImageDir)
	// 自定义处理 /www 路径，解决静态文件服务不会自动加载 index.html 的问题
	// s.router.GET("/www", func(c *gin.Context) {
	// 	c.Header("Content-Type", "text/html; charset=utf-8")
	// 	data, err := embed.WebContent.ReadFile("index.html")
	// 	if err != nil {
	// 		c.String(http.StatusNotFound, "index.html not found")
	// 		return
	// 	}
	// 	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	// })
	s.router.StaticFS("/www/", http.FS(embed.WebContent))
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

	// Debug: log raw query parameters
	slog.Debug("listImages request", "query", c.Request.URL.RawQuery)

	tags := c.QueryArray("tag")
	slog.Debug("listImages tags", "tags", tags, "count", len(tags))

	// Parse integrity_status filter
	var status []models.ImageIntegrityStatus
	if statusStr := c.QueryArray("integrity_status"); len(statusStr) > 0 {
		status = make([]models.ImageIntegrityStatus, 0, len(statusStr))
		for _, s := range statusStr {
			if val, err := strconv.Atoi(s); err == nil {
				status = append(status, models.ImageIntegrityStatus(val))
			}
		}
	}
	slog.Debug("listImages status", "status", status, "count", len(status))

	if imagList, err := s.dbService.ListDownloadedImage(sourceid, tags, status, offset, limit); err == nil {
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
func (s *API) listSources(c *gin.Context) {
	sources := make([]string, 0, len(s.app.GetAppConfig().Spiders))
	for sourceID := range s.app.GetAppConfig().Spiders {
		sources = append(sources, sourceID)
	}
	c.JSON(http.StatusOK, map[string]any{"sources": sources})
}
func (s *API) batchDeleteImages(c *gin.Context) {
	sourceid := c.Param("sourceid")
	var request struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	deletedCount := 0
	for _, id := range request.IDs {
		if err := s.dbService.DeleteImageFile(sourceid, id); err != nil {
			slog.Error("Failed to delete image file", "id", id, "error", err)
			continue
		}
		if err := s.dbService.DeleteImageRecord(sourceid, id); err != nil {
			slog.Error("Failed to delete image record", "id", id, "error", err)
			continue
		}
		deletedCount++
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": deletedCount, "total": len(request.IDs)})
}
func (s *API) batchRedownloadImages(c *gin.Context) {
	sourceid := c.Param("sourceid")
	var request struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	redownloadCount := 0
	for _, id := range request.IDs {
		if err := s.dbService.DeleteImageFile(sourceid, id); err != nil {
			slog.Error("Failed to delete image file for redownload", "id", id, "error", err)
			continue
		}
		if err := s.dbService.UpdateImageIntegrityStatus(sourceid, id, models.ImageIntegrityUnknown); err != nil {
			slog.Error("Failed to update image status for redownload", "id", id, "error", err)
			continue
		}
		redownloadCount++
	}
	c.JSON(http.StatusOK, map[string]any{"redownloaded": redownloadCount, "total": len(request.IDs)})
}
func (s *API) setTagCover(c *gin.Context) {
	sourceid := c.Param("sourceid")
	tag := c.Param("tag")
	var request struct {
		ImageID string `json:"imageId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	if err := s.dbService.SetTagCover(sourceid, tag, request.ImageID); err != nil {
		slog.Error("Failed to set tag cover", "tag", tag, "imageId", request.ImageID, "error", err)
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]any{"success": true})
}
func (s *API) batchUpdateImageStatus(c *gin.Context) {
	sourceid := c.Param("sourceid")
	var request struct {
		IDs    []string `json:"ids" binding:"required"`
		Status int16    `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	updatedCount := 0
	status := models.ImageIntegrityStatus(request.Status)
	for _, id := range request.IDs {
		if err := s.dbService.UpdateImageIntegrityStatus(sourceid, id, status); err != nil {
			slog.Error("Failed to update image status", "id", id, "status", status, "error", err)
			continue
		}
		updatedCount++
	}
	c.JSON(http.StatusOK, map[string]any{"updated": updatedCount, "total": len(request.IDs)})
}

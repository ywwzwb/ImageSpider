package plugins

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"ywwzwb/imagespider/common"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
	"ywwzwb/imagespider/models/config"
	"ywwzwb/imagespider/util"

	"github.com/PuerkitoBio/goquery"
)

type spiderError int

const (
	SpiderErrorSuccess spiderError = iota
	SpiderErrorStop
	SpiderErrorError
)

type spiderState int

const (
	spiderStateInit spiderState = iota
	spiderStateRunning
	spiderStateError
	spiderStateFinished
	spiderStateEarlyStop
)

func (s spiderState) Equals(other common.State) bool {
	if otherState, ok := other.(spiderState); ok {
		return s == otherState
	}
	return false
}

type spiderEventType int

const (
	spiderEventTypeGetPage spiderEventType = iota
	spiderEventTypeError
	spiderEventTypeFinish
	spiderEventTypeEarlyStop
)

type spiderEvent struct {
	eventType spiderEventType
	page      int64
	error     error
}

func (e spiderEvent) Equals(other common.Event) bool {
	if oe, ok := other.(spiderEvent); ok {
		return e.eventType == oe.eventType
	}
	return false
}

type spiderContext struct {
	hasNewData   bool
	oldDataCount int
}

func (e spiderError) Error() string {
	switch e {
	case SpiderErrorSuccess:
		return "success"
	case SpiderErrorStop:
		return "stop spider"
	default:
		return "unknown spider error"
	}
}

type Spider struct {
	app              interfaces.IApplication
	config           config.SpiderList
	stopChain        chan bool
	stopFinishChain  chan bool
	dbService        interfaces.IDBService
	dataCheckService interfaces.IDataCheckerService
}

func newSpider() *Spider {
	spider := Spider{}
	return &spider
}

func init() {
	spider := newSpider()
	spider.stopChain = make(chan bool)
	spider.stopFinishChain = make(chan bool)
	interfaces.Plugins[newSpider().ID()] = spider
}

func (s *Spider) Name() string {
	return "spider"
}
func (s *Spider) ID() string {
	return "spider"
}
func (s *Spider) Load(app interfaces.IApplication) error {
	s.app = app
	s.config = app.GetAppConfig().Spiders
	if len(s.config) == 0 {
		return fmt.Errorf("no spiders")
	}
	dbService, err := app.GetService(s.ID(), DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	s.dbService = dbService.(interfaces.IDBService)

	rawImageDownloaderService, err := app.GetService(s.ID(), ImageDownloaderPluginID, interfaces.ImageDownloaderDownloaderServiceID)
	if err != nil {
		slog.Error("get image downloader service failed", "error", err)
		return err
	}
	imageDownloaderService := rawImageDownloaderService.(interfaces.IImageDownloaderService)

	dataCheckService, err := app.GetService(s.ID(), DataCheckerPluginID, interfaces.DataCheckerServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	s.dataCheckService = dataCheckService.(interfaces.IDataCheckerService)

	for _, spiderConfig := range s.config {
		imageDownloaderService.AddConfig(spiderConfig.ID, &spiderConfig.ImageDownloaderConfig)
		go s.runSpider(spiderConfig)
	}
	return nil
}
func (s *Spider) Unload() {
	for i := 0; i < len(s.config); i++ {
		s.stopChain <- true
		<-s.stopFinishChain
	}
}
func (s *Spider) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	return nil, fmt.Errorf("unsupported service")
}
func (s *Spider) runSpider(spiderConfig *config.SpiderConfig) {
	logger := slog.With("spider", spiderConfig.ID)
	logger.Info("start spider")
	s.dataCheckService.StartChecking(spiderConfig.ID)
	// 启动时添加一个 第 0 页到栈顶, 以便从头开始刷

	s.app.GetRuntimeConfig().AppendStack(spiderConfig.ID, 0)
	if err := s.dbService.InitSource(spiderConfig.ID); err != nil {
		logger.Error("init source failed", "error", err)
		<-s.stopChain
		goto finalize
	}
	for {
		// 抓取所有页面
		var page *int64
		for page = s.app.GetRuntimeConfig().StackTop(spiderConfig.ID); page != nil; page = s.app.GetRuntimeConfig().StackTop(spiderConfig.ID) {
			logger.Debug("page fetching", "start", page)
			err := s.fetchListFromPage(spiderConfig, *page)
			if err == SpiderErrorStop {
				// 结束了
				logger.Info("spider stopped")
				goto finalize
			} else if err == SpiderErrorSuccess {
				logger.Debug("page finish", "start", page)
			} else {
				logger.Debug("page error", "start", page)
			}
		}
		logger.Info("all pages finished, wait for next refresh")
		// 如果没有页面了, 添加一个第零页, 稍后从头开始刷
		s.app.GetRuntimeConfig().AppendStack(spiderConfig.ID, 0)
		select {
		case <-s.stopChain:
			logger.Info("stop spider")
			goto finalize
		case <-time.After(time.Duration(spiderConfig.MetaDownloaderConfig.RefreshInterval) * time.Second):
			// 刷新间隔到了, 从头开始刷
			logger.Info("refresh spider now")
		}
	}
finalize:
	s.stopFinishChain <- true
	logger.Info("stop spider finish")

}
func (s *Spider) fetchListFromPage(spiderConfig *config.SpiderConfig, starPage int64) spiderError {
	slog.Info("fetch list from page", "page", starPage)
	sm := common.NewStateMachine(spiderStateInit)
	c := &spiderContext{}
	sm.AddTransactions([]common.State{spiderStateInit, spiderStateRunning},
		spiderStateRunning,
		spiderEvent{eventType: spiderEventTypeGetPage}, func(event common.Event, context common.Context) bool {
			return true
		},
		func(event common.Event, context common.Context) {
			spiderEvent := event.(spiderEvent)
			slog.Debug("fetch list state run", "page", spiderEvent.page)
			s.fetchListStateRun(spiderEvent, context.(*spiderContext), sm, spiderConfig)
		})
	sm.AddTransaction(spiderStateRunning,
		spiderStateError,
		spiderEvent{eventType: spiderEventTypeError}, func(event common.Event, context common.Context) bool {
			return true
		},
		func(event common.Event, context common.Context) {
			spiderEvent := event.(spiderEvent)
			slog.Debug("fetch list state error", "error", spiderEvent.error)
		})
	sm.AddTransaction(spiderStateRunning,
		spiderStateFinished,
		spiderEvent{eventType: spiderEventTypeFinish}, func(event common.Event, context common.Context) bool {
			return true
		},
		func(event common.Event, context common.Context) {
			slog.Debug("fetch list state finish")
			s.app.GetRuntimeConfig().StackPop(spiderConfig.ID)
		})
	sm.AddTransaction(spiderStateRunning,
		spiderStateEarlyStop,
		spiderEvent{eventType: spiderEventTypeEarlyStop}, func(event common.Event, context common.Context) bool {
			return true
		},
		func(event common.Event, context common.Context) {
			slog.Debug("fetch list state early stop")
		})
	sm.Handle(spiderEvent{eventType: spiderEventTypeGetPage, page: starPage + 1}, c)
	slog.Info("fetch list finish", "page", starPage, "state", sm.CurrentState)
	switch sm.CurrentState {
	case spiderStateEarlyStop:
		return SpiderErrorStop
	case spiderStateFinished:
		return SpiderErrorSuccess
	default:
		return SpiderErrorError
	}
}
func (s *Spider) fetchListStateRun(event spiderEvent, context *spiderContext, sm *common.StateMachine, spiderConfig *config.SpiderConfig) {
	select {
	case <-s.stopChain:
		slog.Debug("fetch list state early stop")
		sm.Handle(spiderEvent{eventType: spiderEventTypeEarlyStop}, context)
		return
	default:
	}
	transport := &http.Transport{
		// 设置连接超时时间
		DialContext: (&net.Dialer{
			Timeout: time.Duration(spiderConfig.MetaDownloaderConfig.ConnectTimeout) * time.Second,
		}).DialContext,
	}
	httpClient := &http.Client{
		Transport: transport,
	}
	var resp *http.Response
	var err error
	url := strings.ReplaceAll(spiderConfig.ListParser.URLTemplate, "__PAGE__", fmt.Sprintf("%d", event.page))
	logger := slog.With("spider", spiderConfig.ID, "page", event.page, "url", url)
	logger.Info("start fetch page")
	for i := 0; i < int(spiderConfig.MetaDownloaderConfig.ErrorRetryMaxCount); i++ {
		var req *http.Request
		req, err = http.NewRequest("GET", url, nil)
		for k, v := range spiderConfig.ListParser.Headers {
			req.Header.Add(k, v)
		}
		if err != nil {
			logger.Error("create request failed", "error", err)
			sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
			return
		}
		resp, err = httpClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			logger.Error("request failed", "error", err, "response", resp)
			select {
			case <-s.stopChain:
				logger.Info("stop spider")
				sm.Handle(spiderEvent{eventType: spiderEventTypeEarlyStop}, context)
				return
			case <-time.After(time.Duration(spiderConfig.MetaDownloaderConfig.ErrorRetryInterval) * time.Second):
				continue
			}
		}
		break
	}
	if err != nil {
		logger.Error("fetch page failed", "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logger.Error("fetch page failed", "status", resp.StatusCode)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: fmt.Errorf("fetch page failed, status:%d", resp.StatusCode)}, context)
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("parse html failed", "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		// 保存错误到一个文件中, 方便事后检查
		func(resp *http.Response, savePath string) {
			// 创建文件
			file, err := os.Create(savePath)
			if err != nil {
				logger.Error("save last page failed, cause create error save file error", "error", err, "path", savePath)
				return
			}
			defer file.Close()
			// 将响应体复制到文件
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				logger.Error("save last page failed, cause copy content failed", "error", err, "path", savePath)
				return
			}
		}(resp, path.Join(s.app.GetAppConfig().WorkDir, "lastError.html"))
		return
	}
	if html, err := doc.Html(); err == nil {
		os.WriteFile(path.Join(s.app.GetAppConfig().WorkDir, "page.html"), []byte(html), 0644)
	} else {
		logger.Error("parse html failed", "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		return
	}
	idParser := util.NewParser(&spiderConfig.ListParser.IDList)
	idList, err := idParser.Parse(doc)
	if len(idList) == 0 || err != nil {
		logger.Error("get id failed", "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		return
	}
	pageNumParser := util.NewParser(&spiderConfig.ListParser.PageNum)
	pageList, err := pageNumParser.Parse(doc)
	if len(pageList) == 0 || err != nil {
		logger.Error("get page failed", "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		return
	}
	page, err := strconv.ParseInt(pageList[0], 10, 64)
	if err != nil {
		logger.Error("parse page failed", "page", pageList[0], "error", err)
		sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
		return
	}
	nextPageParser := util.NewParser(&spiderConfig.ListParser.NextPage)
	nextPageList, err := nextPageParser.Parse(doc)
	lastPage := false
	if len(nextPageList) == 0 || err != nil {
		lastPage = true
		logger.Info("last page")
	}
	for ididx, id := range idList {
		_, ok := s.dbService.GetMeta(id, spiderConfig.ID)
		if ok {
			// 已经刷到过的旧数据
			logger.Debug("already fetched", "id", id)
			if context.hasNewData {
				// 之前已经有新数据了, 已经到新数据的结尾了
				if context.oldDataCount >= spiderConfig.ListParser.SameIDtolerance {
					// 停止
					// 进入完成状态
					logger.Info("this task finish")
					sm.Handle(spiderEvent{eventType: spiderEventTypeFinish}, context)
					return
				} else {
					logger.Debug("found old data, try to continue")
					context.oldDataCount++
					continue
				}
			}
			// 之前没有新数据?
			if page == 1 {
				// 如果是第一页, 那就直接完成了(最新的一页没有任何新数据)
				logger.Info("this task finish cause first page has no new data", "id idx", ididx)
				sm.Handle(spiderEvent{eventType: spiderEventTypeFinish}, context)
				return
			}
			// 没有数据,但不是第一页, 尝试下一个数据继续检查
			continue
		}
		// 新数据
		// 获取元数据
		logger.Debug("new data", "id", id)
		if err := s.fetchMeta(httpClient, id, context, sm, spiderConfig); err != nil {
			if serr, ok := err.(spiderError); ok && serr == SpiderErrorStop {
				sm.Handle(spiderEvent{eventType: spiderEventTypeEarlyStop}, context)
				return
			}
			logger.Error("fetch meta failed", "error", err)
			sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
			return
		}
	}
	slog.Debug("page finished, goto next page", "page", page)
	s.app.GetRuntimeConfig().ReplaceStackTop(spiderConfig.ID, page)
	if lastPage {
		logger.Info("is last page finish now")
		sm.Handle(spiderEvent{eventType: spiderEventTypeFinish}, context)
		return
	} else {
		sm.Handle(spiderEvent{eventType: spiderEventTypeGetPage, page: page + 1}, context)
	}
}
func (s *Spider) fetchMeta(httpClient *http.Client, id string, context *spiderContext, sm *common.StateMachine, spiderConfig *config.SpiderConfig) error {
	select {
	case <-s.stopChain:
		slog.Debug("fetch list state early stop")
		return SpiderErrorStop
	default:
	}
	var resp *http.Response
	var err error
	url := strings.ReplaceAll(spiderConfig.MetaParser.URLTemplate, "__ID__", id)
	logger := slog.With("spider", spiderConfig.ID, "meta id", id, "url", url)
	logger.Info("start fetch meta")
	for i := 0; i < int(spiderConfig.MetaDownloaderConfig.ErrorRetryMaxCount); i++ {
		var req *http.Request
		req, err = http.NewRequest("GET", url, nil)
		for k, v := range spiderConfig.MetaParser.Headers {
			req.Header.Add(k, v)
		}
		if err != nil {
			logger.Error("create request failed", "error", err)
			sm.Handle(spiderEvent{eventType: spiderEventTypeError, error: err}, context)
			return err
		}
		resp, err = httpClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			logger.Error("request failed", "error", err, "response", resp)
			select {
			case <-s.stopChain:
				logger.Info("stop spider")
				return SpiderErrorStop
			case <-time.After(time.Duration(spiderConfig.MetaDownloaderConfig.ErrorRetryInterval) * time.Second):
				continue
			}
		}
		break
	}
	if err != nil {
		logger.Error("fetch meta failed", "error", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logger.Error("fetch meta failed", "status", resp.StatusCode)
		return fmt.Errorf("fetch meta failed, status:%d", resp.StatusCode)
	}
	var meta models.ImageMeta
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("parse html failed", "error", err)
		return err
	}
	meta.Tags = make([]string, 0)
	for _, tagParser := range spiderConfig.MetaParser.Tags {
		idParser := util.NewParser(&tagParser)
		tagList, err := idParser.Parse(doc)
		if len(tagList) == 0 || err != nil {
			continue
		}
		meta.Tags = append(meta.Tags, tagList...)
	}
	imageURLParser := util.NewParser(&spiderConfig.MetaParser.ImageURL)
	imageURLList, err := imageURLParser.Parse(doc)
	if len(imageURLList) == 0 || err != nil {
		logger.Error("get image failed", "error", err)
		return err
	}
	meta.ImageURL = imageURLList[0]
	postTimeParser := util.NewParser(&spiderConfig.MetaParser.PostTime)
	postTimeList, err := postTimeParser.Parse(doc)
	if len(postTimeList) == 0 || err != nil {
		logger.Error("get post time failed", "error", err)
		return err
	}
	postTime, err := time.Parse(spiderConfig.MetaParser.PostTime.Ext["format"], postTimeList[0])
	if err != nil {
		logger.Error("parse post time failed", "error", err, "postTimeStr", postTimeList[0])
		return err
	}
	meta.PostTime = postTime
	meta.SourceID = spiderConfig.ID
	meta.ID = id
	context.oldDataCount = 0
	context.hasNewData = true
	logger.Debug("save new meta", "meta", meta)
	s.dbService.InsertMeta(meta)
	return nil
}

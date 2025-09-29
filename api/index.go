package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"pansou/api"
	"pansou/config"
	"pansou/service"
)

var (
	once sync.Once
	app  *gin.Engine
)

// Handler 是 Vercel 的入口函数
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		// 初始化配置
		config.LoadConfig()

		// 创建搜索服务
		searchService := service.NewSearchService()
		api.SetSearchService(searchService)

		// 创建 Gin 应用
		gin.SetMode(gin.ReleaseMode)
		app = gin.New()

		// 添加中间件
		app.Use(gin.Recovery())
		app.Use(api.CORSMiddleware())

		// 设置路由
		app.GET("/api/search", api.SearchHandler)
		app.POST("/api/search", api.SearchHandler)

		// 静态文件处理
		app.Static("/static", "./static")
		app.StaticFile("/", "./static/index.html")
		app.StaticFile("/favicon.ico", "./static/favicon.ico")
		app.StaticFile("/robots.txt", "./static/robots.txt")
		app.StaticFile("/ads.txt", "./static/ads.txt")
		app.StaticFile("/sitemap.xml", "./static/sitemap.xml")

		// HTML页面
		app.StaticFile("/about", "./static/about.html")
		app.StaticFile("/contact", "./static/contact.html")
		app.StaticFile("/help", "./static/help.html")
		app.StaticFile("/privacy", "./static/privacy.html")
		app.StaticFile("/terms", "./static/terms.html")
		app.StaticFile("/search-tips", "./static/search-tips.html")
		app.StaticFile("/categories", "./static/categories.html")
		app.NoRoute(func(c *gin.Context) {
			c.File("./static/404.html")
		})
	})

	app.ServeHTTP(w, r)
}
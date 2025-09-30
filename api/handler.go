package handler

import (
	"fmt"
	"net/http"
	"sync"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/model"
	"pansou/service"
	"pansou/plugin"
	jsonutil "pansou/util/json"
	"pansou/util"
	"pansou/util/cache"

	// 导入所有插件以触发init函数自动注册
	_ "pansou/plugin/hunhepan"
	_ "pansou/plugin/jikepan"
	_ "pansou/plugin/panwiki"
	_ "pansou/plugin/pansearch"
	_ "pansou/plugin/panta"
	_ "pansou/plugin/qupansou"
	_ "pansou/plugin/susu"
	_ "pansou/plugin/thepiratebay"
	_ "pansou/plugin/wanou"
	_ "pansou/plugin/xuexizhinan"
	_ "pansou/plugin/panyq"
	_ "pansou/plugin/zhizhen"
	_ "pansou/plugin/labi"
	_ "pansou/plugin/muou"
	_ "pansou/plugin/ouge"
	_ "pansou/plugin/shandian"
	_ "pansou/plugin/duoduo"
	_ "pansou/plugin/huban"
	_ "pansou/plugin/cyg"
	_ "pansou/plugin/erxiao"
	_ "pansou/plugin/miaoso"
	_ "pansou/plugin/fox4k"
	_ "pansou/plugin/pianku"
	_ "pansou/plugin/clmao"
	_ "pansou/plugin/wuji"
	_ "pansou/plugin/cldi"
	_ "pansou/plugin/xiaozhang"
	_ "pansou/plugin/libvio"
	_ "pansou/plugin/leijing"
	_ "pansou/plugin/xb6v"
	_ "pansou/plugin/xys"
	_ "pansou/plugin/ddys"
	_ "pansou/plugin/hdmoli"
	_ "pansou/plugin/yuhuage"
	_ "pansou/plugin/u3c3"
	_ "pansou/plugin/javdb"
	_ "pansou/plugin/clxiong"
	_ "pansou/plugin/jutoushe"
	_ "pansou/plugin/sdso"
	_ "pansou/plugin/xiaoji"
	_ "pansou/plugin/xdyh"
	_ "pansou/plugin/haisou"
)

var (
	searchService *service.SearchService
	once          sync.Once
	app           *gin.Engine
)

// CORSMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Handler 是 Vercel 的入口函数
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		// 初始化配置
		config.Init()

		// 初始化HTTP客户端（插件需要）
		util.InitHTTPClient()

		// 初始化缓存写入管理器
		globalCacheWriteManager, err := cache.NewDelayedBatchWriteManager()
		if err != nil {
			fmt.Printf("⚠️ 缓存写入管理器创建失败: %v\n", err)
		} else {
			if err := globalCacheWriteManager.Initialize(); err != nil {
				fmt.Printf("⚠️ 缓存写入管理器初始化失败: %v\n", err)
			} else {
				// 将缓存写入管理器注入到service包
				service.SetGlobalCacheWriteManager(globalCacheWriteManager)

				// 延迟设置主缓存更新函数，确保service初始化完成
				go func() {
					// 等待一小段时间确保service包完全初始化
					time.Sleep(100 * time.Millisecond)
					if mainCache := service.GetEnhancedTwoLevelCache(); mainCache != nil {
						globalCacheWriteManager.SetMainCacheUpdater(func(key string, data []byte, ttl time.Duration) error {
							return mainCache.SetBothLevels(key, data, ttl)
						})
					}
				}()
			}
		}

		// 初始化异步插件系统
		plugin.InitAsyncPluginSystem()

		// 初始化插件管理器
		pluginManager := plugin.NewPluginManager()

		// 注册全局插件（根据配置过滤）
		if config.AppConfig.AsyncPluginEnabled {
			pluginManager.RegisterGlobalPluginsWithFilter(config.AppConfig.EnabledPlugins)
			fmt.Printf("✅ 已注册 %d 个搜索插件\n", len(pluginManager.GetPlugins()))
		} else {
			fmt.Println("⚠️ 异步插件已禁用")
		}

		// 创建搜索服务
		searchService = service.NewSearchService(pluginManager)

		// 创建 Gin 应用
		gin.SetMode(gin.ReleaseMode)
		app = gin.New()

		// 添加中间件
		app.Use(gin.Recovery())
		app.Use(corsMiddleware())

		// 设置路由
		app.GET("/api/search", searchHandler)
		app.POST("/api/search", searchHandler)

		// 根路径返回简单的HTML
		app.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(200, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>网盘搜索引擎</title>
</head>
<body>
    <h1>网盘搜索引擎 API</h1>
    <p>API端点: <code>GET/POST /api/search</code></p>
    <p>参数示例: <code>/api/search?kw=关键词</code></p>
</body>
</html>`)
		})

		// 404处理
		app.NoRoute(func(c *gin.Context) {
			c.JSON(404, gin.H{
				"error": "Not Found",
				"path":  c.Request.URL.Path,
			})
		})
	})

	app.ServeHTTP(w, r)
}

// searchHandler 搜索处理函数（内部使用）
func searchHandler(c *gin.Context) {
	var req model.SearchRequest
	var err error

	// 根据请求方法不同处理参数
	if c.Request.Method == http.MethodGet {
		// GET方式：从URL参数获取
		// 获取keyword，必填参数
		// 兼容两种参数名
keyword := c.Query("kw")
if keyword == "" {
    keyword = c.Query("keyword")  // 添加这行兼容前端
}
		
		// 处理channels参数，支持逗号分隔
		channelsStr := c.Query("channels")
		var channels []string
		// 只有当参数非空时才处理
		if channelsStr != "" && channelsStr != " " {
			parts := strings.Split(channelsStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					channels = append(channels, trimmed)
				}
			}
		}
		
		// 处理并发数
		concurrency := 0
		concStr := c.Query("conc")
		if concStr != "" && concStr != " " {
			concurrency = util.StringToInt(concStr)
		}
		
		// 处理强制刷新
		forceRefresh := false
		refreshStr := c.Query("refresh")
		if refreshStr != "" && refreshStr != " " && refreshStr == "true" {
			forceRefresh = true
		}
		
		// 处理结果类型和来源类型
		resultType := c.Query("res")
		if resultType == "" || resultType == " " {
			resultType = "merge" // 直接设置为默认值merge
		}
		
		sourceType := c.Query("src")
		if sourceType == "" || sourceType == " " {
			sourceType = "all" // 直接设置为默认值all
		}
		
		// 处理plugins参数，支持逗号分隔
		var plugins []string
		// 检查请求中是否存在plugins参数
		if c.Request.URL.Query().Has("plugins") {
			pluginsStr := c.Query("plugins")
			// 判断参数是否非空
			if pluginsStr != "" && pluginsStr != " " {
				parts := strings.Split(pluginsStr, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						plugins = append(plugins, trimmed)
					}
				}
			}
		} else {
			// 如果请求中不存在plugins参数，设置为nil
			plugins = nil
		}
		
		// 处理cloud_types参数，支持逗号分隔
		var cloudTypes []string
		// 检查请求中是否存在cloud_types参数
		if c.Request.URL.Query().Has("cloud_types") {
			cloudTypesStr := c.Query("cloud_types")
			// 判断参数是否非空
			if cloudTypesStr != "" && cloudTypesStr != " " {
				parts := strings.Split(cloudTypesStr, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						cloudTypes = append(cloudTypes, trimmed)
					}
				}
			}
		} else {
			// 如果请求中不存在cloud_types参数，设置为nil
			cloudTypes = nil
		}
		
		// 处理ext参数，JSON格式
		var ext map[string]interface{}
		extStr := c.Query("ext")
		if extStr != "" && extStr != " " {
			// 处理特殊情况：ext={}
			if extStr == "{}" {
				ext = make(map[string]interface{})
			} else {
				if err := jsonutil.Unmarshal([]byte(extStr), &ext); err != nil {
					c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的ext参数格式: "+err.Error()))
					return
				}
			}
		}
		// 确保ext不为nil
		if ext == nil {
			ext = make(map[string]interface{})
		}

		req = model.SearchRequest{
			Keyword:      keyword,
			Channels:     channels,
			Concurrency:  concurrency,
			ForceRefresh: forceRefresh,
			ResultType:   resultType,
			SourceType:   sourceType,
			Plugins:      plugins,
			CloudTypes:   cloudTypes, // 添加cloud_types到请求中
			Ext:          ext,
		}
	} else {
		// POST方式：从请求体获取
		data, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "读取请求数据失败: "+err.Error()))
			return
		}

		if err := jsonutil.Unmarshal(data, &req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(400, "无效的请求参数: "+err.Error()))
			return
		}
	}
	
	// 检查并设置默认值
	if len(req.Channels) == 0 {
		req.Channels = config.AppConfig.DefaultChannels
	}
	
	// 如果未指定结果类型，默认返回merge并转换为merged_by_type
	if req.ResultType == "" {
		req.ResultType = "merged_by_type"
	} else if req.ResultType == "merge" {
		// 将merge转换为merged_by_type，以兼容内部处理
		req.ResultType = "merged_by_type"
	}
	
	// 如果未指定数据来源类型，默认为全部
	if req.SourceType == "" {
		req.SourceType = "all"
	}
	
	// 参数互斥逻辑：当src=tg时忽略plugins参数，当src=plugin时忽略channels参数
	if req.SourceType == "tg" {
		req.Plugins = nil // 忽略plugins参数
	} else if req.SourceType == "plugin" {
		req.Channels = nil // 忽略channels参数
	} else if req.SourceType == "all" {
		// 对于all类型，如果plugins为空或不存在，统一设为nil
		if req.Plugins == nil || len(req.Plugins) == 0 {
			req.Plugins = nil
		}
	}
	
	// 启用调试输出
	fmt.Printf("🔧 [调试] 搜索参数: keyword=%s, channels=%v, concurrency=%d, refresh=%v, resultType=%s, sourceType=%s, plugins=%v, cloudTypes=%v\n",
		req.Keyword, req.Channels, req.Concurrency, req.ForceRefresh, req.ResultType, req.SourceType, req.Plugins, req.CloudTypes)

	// 执行搜索
	result, err := searchService.Search(req.Keyword, req.Channels, req.Concurrency, req.ForceRefresh, req.ResultType, req.SourceType, req.Plugins, req.CloudTypes, req.Ext)
	
	if err != nil {
		response := model.NewErrorResponse(500, "搜索失败: "+err.Error())
		jsonData, _ := jsonutil.Marshal(response)
		c.Data(http.StatusInternalServerError, "application/json", jsonData)
		return
	}

	// 返回结果
	response := model.NewSuccessResponse(result)
	jsonData, _ := jsonutil.Marshal(response)
	c.Data(http.StatusOK, "application/json", jsonData)
} 

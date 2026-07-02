# PbootCMS-Go 开发技术文档（整合版）

> 本文档供新人接手参考，整合自「留言系统前基线版」与代码实勘，已逐项核对 go.mod、config、session、controller、parser 等源码。
> 对应源码版本：PbootCMS 3.2.12 PHP → Go 移植版
> 最后更新：2026-07-01

---

## 一、项目概述

PbootCMS-Go 是将 PHP 版 PbootCMS 3.2.12 忠实移植为 Go 语言的企业网站管理系统。项目保留了原版的数据库表结构（`ay_` 前缀）、模板语法、URL 路由规则和后台 UI（Layui），用 Go 技术栈替换了 PHP 后端。

### 技术栈一览

| 层次 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 语言 | Go | 1.25.0 | 单二进制部署 |
| Web 框架 | Gin | v1.12.0 | 路由分发、中间件、请求处理 |
| ORM | GORM | v1.31.1 | AutoMigrate，`ay_` 前缀策略 |
| 数据库驱动 | glebarez/sqlite | v1.11.0 | **纯 Go 驱动，无需 CGO**，文件 `data/pbootcms.db` |
| 后台模板 | Pongo2 | v6.1.0 | Django 风格模板 + 自研 PbootCMS 语法转换器 |
| 前台模板 | 自研 TagParser | — | `{gboot:xxx}` 标签语法 + fsnotify 热重载 |
| 模板热更新 | fsnotify | v1.10.1 | 前台模板文件变更自动重载 |
| 邮件 | go-mail | v0.7.3 | SMTP 发信，配置存 `ay_config` |
| 前端 UI（后台） | Layui 2.5.4 + jQuery 1.12.4 | — | 与 PbootCMS 原版一致 |
| 前端 UI（前台） | Bootstrap 4 + Swiper 4 | — | 前台模板自带 |

### 核心设计原则

1. **数据库零改动**：所有表结构、字段名、表前缀 `ay_` 与 PbootCMS PHP 版完全一致，可直接迁移数据
2. **模板语法兼容**：后台保留 PbootCMS 原版 PHP 模板语法（`{foreach}`、`{if}`、`{$var->field}`），前台使用 `{gboot:xxx}` 标签
3. **URL 100% 兼容**：通过路由重写和 NoRoute 兜底，实现原版 URL 格式的完整支持

### 与原版的关键命名差异

前台模板标签前缀由 PHP 版的 `{pboot:xxx}` 改为 Go 版的 `{gboot:xxx}`（**g**o-boot），但语义与属性与原版一致。后台模板仍沿用 pongo2 的 `{{ }}` 语法。这是新人最容易混淆的一点：**前台走自研 `gboot` 解析器，后台走 pongo2**。

---

## 二、目录结构与 PHP 映射

```
pbootcms-go/
├── main.go                          # 程序入口，启动流程，路由注册
├── go.mod / go.sum                  # Go 模块定义
├── build.ps1                        # 编译脚本（PowerShell）
├── config/
│   ├── config.go                    # 配置结构体与加载逻辑（sync.Once 单例）
│   └── config.json                  # 配置文件（端口、数据库路径等）
├── core/
│   ├── db/db.go                     # GORM + glebarez/sqlite 数据库初始化
│   ├── basic/view.go                # 后台模板引擎（pongo2 + PHP 语法转换）
│   └── mediaplugin/plugin.go        # GORM 媒体缓存失效插件
├── apps/
│   ├── route/route.go               # 后台路由集中注册
│   ├── common/
│   │   ├── BaseController.go        # 基础控制器（JSON 响应、批量排序等）
│   │   ├── AdminController.go       # 后台控制器基类（占位，实际未被嵌入）
│   │   ├── HomeController.go        # 前台控制器基类（占位，实际未被嵌入）
│   │   ├── Render.go                # 后台模板渲染入口（AppThemeDir 定义）
│   │   ├── session.go               # 自实现内存 Session（PbootGo Cookie）
│   │   ├── notice.go                # 通知消息常量（硬约束：禁止硬编码）
│   │   ├── middleware/
│   │   │   ├── auth.go              # 后台认证中间件
│   │   │   └── path_rewrite.go      # PbootCMS URL 重写映射表
│   │   ├── parser/                  # 前台模板标签解析引擎
│   │   │   ├── tags.go              # TagParser 核心解析流水线
│   │   │   ├── engine.go            # TemplateStore 模板存储与热重载
│   │   │   ├── providers.go         # 所有标签 Provider 注册
│   │   │   ├── if_eval.go           # 条件表达式求值
│   │   │   └── convert.go           # 辅助解析函数
│   │   ├── mail/mailer.go           # SMTP 邮件发送
│   │   └── webhook/webhook.go       # 异步 Webhook 推送
│   ├── admin/
│   │   ├── controller/              # 后台控制器
│   │   │   ├── IndexController.go   # 登录/首页/上传/验证码
│   │   │   ├── content/             # 内容管理（15个控制器）
│   │   │   ├── system/              # 系统管理（10个控制器）
│   │   │   └── member/              # 会员管理（4个控制器）
│   │   ├── model/                   # 数据模型
│   │   │   ├── db.go                # 全局 DB 实例 + 29个类型别名
│   │   │   ├── content/             # 内容模型（15个）
│   │   │   ├── system/              # 系统模型（11个）
│   │   │   ├── member/              # 会员模型（4个）
│   │   │   └── seed/seed.go         # 种子数据初始化
│   │   ├── service/content/         # 业务服务层（MVC+S）
│   │   ├── helper/                  # 模板辅助函数
│   │   └── view/                    # 后台 HTML 模板
│   │       ├── common/              # 公共模板（head/foot/ueditor）
│   │       ├── content/             # 内容管理模板
│   │       ├── system/              # 系统管理模板
│   │       ├── member/              # 会员管理模板
│   │       └── index.html           # 登录页
│   └── home/
│       └── controller/front.go      # 前台控制器（FrontController）
├── template/default/                # 前台模板目录
│   ├── comm/                        # 公共模板（head/foot/page等）
│   ├── index.html, list.html, content.html, search.html ...
│   └── static/                      # 前台静态资源（bootstrap/css/js/swiper）
├── static/                          # 全局公共资源
│   ├── admin/                       # 后台 CSS/JS/图片/Layui/font-awesome
│   ├── upload/                      # 用户上传文件（gitignore）
│   ├── images/                      # 全局图片（logo/nopic）
│   └── backup/                      # 备份目录
├── data/pbootcms.db                 # SQLite 数据库文件
├── runtime/                         # 运行时缓存/调试产物
├── bin/                             # 编译产物
└── docs/                            # 文档
```

### PHP → Go 映射要点

| PHP 原版 | Go 版 | 变化 |
|----------|-------|------|
| `apps/home/controller/*`（13 个控制器） | `apps/home/controller/front.go` | 合并为单个 FrontController |
| `core/view/Parser.php` | `apps/common/parser/`（5 文件） | 标签前缀 `pboot:`→`gboot:` |
| `core/basic/View.php` | `core/basic/view.go` | PHP 语法→pongo2 转译 |
| `core/basic/Db.php`（PDO/SQLite） | `core/db/db.go`（GORM） | 手写 SQL→ORM |
| 无 Service 层 | `apps/admin/service/*` | 新增业务逻辑层（MVC+S） |
| `apps/common/route.php` | `apps/route/route.go` + `main.go` | Gin 路由 + NoRoute 兜底 |
| SQL 文件导入初始数据 | `apps/admin/seed/seed.go` | 代码化种子，幂等 |
| 后台静态资源在 `apps/admin/view/default/` | `static/admin/` | 有意集中，便于 CDN 部署 |

---

## 三、启动流程

`main.go` 的 `main()` 函数按以下固定顺序执行：

```
1. config.Load("config/config.json")     → 加载配置（sync.Once 单例）
2. model.InitDB(cfg)                     → 初始化数据库（GORM + glebarez/sqlite）
3. system.AutoMigrate()                  → 系统模型自动迁移（11个表）
4. content.AutoMigrate()                 → 内容模型自动迁移（15个表）
5. member.AutoMigrate()                  → 会员模型自动迁移（4个表）
6. seed.Init()                           → 种子数据初始化（幂等）
7. basic.InitViewEngine(...)             → 初始化后台模板引擎（pongo2）
8. parser.NewTemplateStore(...)          → 初始化前台模板引擎（加载+热重载）
9. gin.Default() + 中间件 + 路由注册     → 创建 Gin 引擎
10. r.Run(":8080")                       → 启动 HTTP 服务
```

> **依赖顺序**：`seed.Init()` 必须在 `AutoMigrate` 之后执行——种子要往 `ay_user`、`ay_menu` 等表写数据。`seed` 内部幂等调用 `content.EnsureContentExtTable()` 确保 `ay_content_ext` 动态表存在。

### 种子数据

`seed.Init()` → `SeedData()` 逻辑：

1. **每次启动幂等**调用 `content.EnsureContentExtTable()` 确保 `ay_content_ext` 基础表存在
2. 检查 `ay_user` 是否非空：非空则跳过首次种子，但仍调 `ensureMenuVersion()` 校验菜单版本
3. `ensureMenuVersion()`：以 `mcode='M1006'` 为版本标志，缺失则 `DELETE FROM ay_menu` 后重建
4. 首次种子依次执行：
   - 管理员账号：`admin` / `admin`（密码双重 MD5：`md5(md5("admin"))`）
   - 站点信息、公司信息
   - 36 条后台菜单（6个一级菜单 + 30个子菜单，与 PbootCMS 3.2.12 原版 1:1 对齐）
   - 超级管理员角色 `R100`
   - 2个会员组（G1普通/G2VIP）、2个内容模型（3D1文章/3D2单页）
   - 12项系统配置（close_site、url_rule_type=2、page_size=15、session_time=1800 等）

---

## 四、配置系统

配置分两层：**静态配置**在 `config/config.json`（数据库连接、端口），**动态配置**在数据库 `ay_config` 表（站点开关、分页大小、SMTP、webhook 等）。

### 静态配置 config.json

```json
{
  "app": {
    "debug": true,
    "port": 8080,
    "template_dir": "template/default",
    "admin_template_dir": "apps/admin/view",
    "static_dir": "static",
    "runtime_dir": "runtime",
    "url_suffix": ".html",
    "page_size": 15,
    "page_num": 5,
    "admin_path": "admin",
    "cache_enabled": false,
    "cache_time": 900
  },
  "database": {
    "type": "sqlite",
    "host": "127.0.0.1",
    "port": 3306,
    "user": "root",
    "passwd": "",
    "dbname": "data/pbootcms.db",
    "prefix": "ay_"
  }
}
```

`config/config.go` 用 `encoding/json` 解析到 `Config` 结构体（`sync.Once` 单例）。`App` 结构体含 12 个字段（Debug/Port/TemplateDir/AdminTemplateDir/StaticDir/RuntimeDir/URLSuffix/PageSize/PageNum/AdminPath/CacheEnabled/CacheTime），`Database` 结构体含 7 个字段。修改端口或库路径后需重启。

### 动态配置 ay_config

业务代码统一通过 `model.GetConfigValue(name, default)` 实时查询：

```go
smtpServer := model.GetConfigValue("smtp_server", "")
pageSize    := model.GetConfigValue("page_size", "15")
```

后台"全局配置 → 配置参数"页直接读写此表。`Render.go` 的 `loadConfigToData()` 将全部配置注入后台模板数据。

---

## 五、数据库层

### 连接配置

```go
// core/db/db.go
DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
    NamingStrategy: schema.NamingStrategy{
        TablePrefix:   "ay_",        // 保留 PbootCMS 原版表前缀
        SingularTable: true,          // 单数表名（user 而非 users）
    },
})
sqlDB.SetMaxOpenConns(1)              // SQLite 红线：单连接防锁死
sqlDB.SetMaxIdleConns(1)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
db.Use(&mediaplugin.MediaDirtyPlugin{})  // 注册媒体缓存失效插件
```

> **单连接的代价与原因**：SQLite 写操作加全局锁，多连接并发写会触发 `SQLITE_BUSY`。`SetMaxOpenConns(1)` 用串行化换取稳定性。高并发写场景需考虑迁移到 MySQL/PostgreSQL。
>
> **注意**：使用 `glebarez/sqlite` 纯 Go 驱动，无需安装 CGO/GCC，跨平台编译友好。

### 模型组织

29个 GORM 模型按三个子包组织，通过 `apps/admin/model/db.go` 的 type alias 统一导出：

```go
var DB = db.DB                        // 全局数据库实例
type Content = content.Content        // 类型别名，控制器只需 import model
type Message = content.Message
type AdminUser = system.AdminUser
// ... 共 29 个别名
```

绝大多数模型 struct 不需要写 `TableName()`，自动映射为 `ay_` + 蛇形单数名。少数显式覆盖（如 `AdminUser→ay_user`、`MemberComment→ay_comment`）。

### AutoMigrate 建表机制

三个模型子包各自导出 `AutoMigrate()`，在 `main.go` 启动时调用。GORM `AutoMigrate` 会自动建表、自动补列，**但不删列**。删除字段时仅删 `ay_extfield` 定义记录，物理列保留（熔断保护，避免误删数据）。

### 动态扩展表 ay_content_ext

这是全项目**唯一不用 struct 映射**的表。它在 `ay_extfield` 定义新字段时通过 `ALTER TABLE ay_content_ext ADD COLUMN` 动态加列，列名运行时才确定，因此用 `map[string]interface{}` 操作而非 struct。

```go
// apps/admin/model/content/ExtFieldModel.go
func EnsureContentExtTable() {
    db.DB.Exec(`CREATE TABLE IF NOT EXISTS ay_content_ext (
        extid INTEGER PRIMARY KEY AUTOINCREMENT, contentid INTEGER NOT NULL)`)
}
// 加列前用 PRAGMA table_info 检测是否已存在，避免重复 ALTER
```

前台 `{gboot:list}` 标签读取内容时，`ext_` 前缀字段会 LEFT JOIN `ay_content_ext` 一并返回，扩展字段筛选 `ctx.Filters` 也走这条路径。

### 媒体缓存插件 mediaplugin

PHP 原版在每个 Controller 的 `mod()` 里手动处理媒体库缓存失效，逻辑分散易遗漏。Go 版用 GORM Plugin 集中解决：注册 `Create/Update/Delete` 钩子，写操作完成后检查表名是否在白名单 `MediaReferencingTables`，命中则用 `atomic.Bool` 无锁标记 dirty，下次 `MediaController` 读取时自动重扫。

性能设计：dirty 标记用 `sync/atomic` 而非 `RWMutex`，读写都是 O(1) 无锁；表名短名缓存到 `sync.Map`，避免热路径上重复字符串分配。

### 已注册模型清单

**系统模型（11个）**：AdminUser（ay_user）、Menu、MenuAction、Role、RoleLevel、Syslog、Area、Config、Type、DictType、Database

**内容模型（15个）**：Company、Content、ContentSort、ExtField、Form、FormField、Label、Link、Message、Model、Single、Site、Slide、Tags、MediaMark

**会员模型（4个）**：Member、Comment（MemberComment 共用 ay_comment）、MemberGroup、MemberField

### 主业务表总结

- **内容核心**：`ay_content`（文章/列表内容）、`ay_content_sort`（栏目分类，树形 pcode/scode）、`ay_content_ext`（动态扩展字段）、`ay_single`（单页内容）
- **模型与字段**：`ay_model`（内容模型）、`ay_extfield`（扩展字段定义，驱动 ay_content_ext 加列）
- **站点/公司**：`ay_site`、`ay_company`
- **扩展内容**：`ay_slide`（按 gid 分组）、`ay_link`（按 gid 分组）、`ay_message`、`ay_tags`、`ay_label`、`ay_form`+`ay_form_field`（自定义表单定义）
- **自定义表单数据**：动态表 `ay_diy_*`（无 struct，前台提交时动态 INSERT）
- **会员**：`ay_member`、`ay_member_group`、`ay_member_field`、`ay_comment`
- **系统**：`ay_user`、`ay_menu`/`ay_menu_action`、`ay_role`/`ay_role_level`、`ay_config`、`ay_area`、`ay_type`/`ay_dict_type`、`ay_syslog`、`ay_database`

---

## 六、路由系统

### 前台路由

直接注册在 Gin Engine 上，无中间件：

| 方法 | 路径 | 控制器方法 | 说明 |
|------|------|-----------|------|
| GET | `/` | `fc.Index` | 首页 |
| GET | `/search` | `fc.Search` | 搜索页（支持分页） |
| GET | `/tags` | `fc.Tags` | 标签页 |
| GET/POST | `/message` | `fc.Message` | 留言页 |
| GET | `/api/visits` | `fc.Visits` | 访问量统计 |
| GET | `/api/checkcode` | `fc.CheckCode` | 验证码图片 |
| GET | `/sort/:scode` | `fc.SortByScode` | 栏目列表页 |
| GET | `/content/:id` | `fc.ContentByID` | 内容详情页 |
| NoRoute | `*` | `fc.ContentPage` | 动态内容页兜底 |

### 后台路由

统一前缀 `/admin`，挂载 `AdminAuth()` 中间件。在 `apps/route/route.go` 中集中注册，采用 PbootCMS 风格的 `/admin/{模块}/{action}` 与 `/admin/{模块}/{子模块}/{action}` 两级结构，部分配 `/*action` catch-all。

免登录白名单：`/admin/`、`/admin/index/index`、`/admin/index/login`、`/admin/index/checkCode`。

- **IndexController** — 登录、首页、上传、验证码
- **content 模块** — 内容/栏目/单页/幻灯片/友链/留言/标签/表单/模型/扩展字段/媒体库
- **system 模块** — 菜单/用户/角色/配置/数据库/区域/日志/类型/图片清理
- **member 模块** — 会员/会员组/会员字段/评论

### URL 重写机制

由于 PbootCMS 原版 PHP 模板生成的 URL 与 Go 路由树不完全匹配，`NoRoute` 实现三层兜底：

1. **大小写归一化**：`/admin/Content/index` → `/admin/content/index`
2. **PbootCMS 短路径重写**：`/admin/config` → `/admin/system/config`（约 25 条前缀映射规则）
3. **前台内容页兜底**：基于 `filename`/`urlname` 的伪静态 URL 解析

`path_rewrite.go` 的 `pbootToGoRouteMap` 映射表示例：

```
/admin/config       → /admin/system/config
/admin/contentsort  → /admin/content/sort
/admin/single       → /admin/content/single
/admin/slide        → /admin/content/slide
/admin/membergroup  → /admin/member/group
/admin/M156         → /admin/index/home   # 顶级菜单占位
```

### .html 后缀剥离

全局中间件：所有以 `.html` 结尾的请求剥离后缀后用 `r.HandleContext` 重新路由，使 `/content/7.html` 和 `/content/7` 访问同一页面。

---

## 七、后台管理系统

### 认证流程

**Session 机制**（`apps/common/session.go`）：
- **自实现内存 Session**（`map[string]SessionData` + `sync.RWMutex`），非持久化，重启丢失
- Cookie 名 `PbootGo`，有效期 24 小时（86400 秒）
- Cookie 值为随机生成的 base64 session ID，服务端按 session ID 查内存 map

**登录流程**（`IndexController.Login`）：
1. 验证码校验（答案存内存 `checkCodeStore` map，按 session ID 索引）
2. 登录锁定检查（`checkLoginBlack`：同 IP 失败 5 次锁 900 秒）
3. 密码双重 MD5：`md5(md5(明文))`
4. 查询 `ay_user` 表验证（`username AND password AND status=1`）
5. Session 写入 `admin_uid`/`username`/`realname`/`ucode`/`rcodes`/`levels` + 区域信息

**认证中间件**（`middleware/auth.go`）：
- 白名单路径直接放行：`/admin/`、`/admin/index/login`、`/admin/index/checkCode`
- 未登录：AJAX 返回 JSON（`{"code":0,"msg":"未登录"}`），普通请求 302 重定向到 `/admin/`
- 已登录：Session 信息注入 `gin.Context`

### 控制器继承体系

所有 admin 控制器直接嵌入 `common.BaseController`。`AdminController` 和 `HomeController` 虽然定义了，但**实际没有被任何控制器嵌入**，是占位基类。`FrontController` 不嵌入任何基类，而是持有 `*parser.TemplateStore`。

新人新增控制器时应嵌入 `common.BaseController`。

### BaseController 通用方法

| 方法 | 功能 |
|------|------|
| `BatchSort(c, modelPtr, sortCol, defaultSort)` | 通用批量排序（脏检查优化） |
| `JSONOK(c, data)` | 成功响应 `{"code":1,"data":...}` |
| `JSONOKMsg(c, msg)` | 成功响应带消息 |
| `JSONFail(c, msg)` | 失败响应 `{"code":0,"msg":...}` |
| `GetAdminUID(c)` / `GetAdminUsername(c)` | 从 Session 获取用户信息 |
| `IsLogin(c)` | 判断登录态 |
| `MarkMediaCacheDirty()` | 标记媒体缓存脏 |

### MVC+S 请求链路

Go 版在 PHP 的 MVC 基础上**新增了 Service 层**（MVC+S），把业务逻辑从 Controller 下沉到 Service。以"内容列表页"为例：

```
GET /admin/content/index?scode=1&page=2
  │
  ├─ middleware.AdminAuth()        # 校验 session admin_uid
  │
  ├─ ContentController.Index(c)    # controller：解析参数
  │     ├─ scode := c.Query("scode")
  │     └─ list, total := svc.GetContentList(scode, page, size)  # 调 service
  │
  ├─ ContentService.GetContentList()  # service：业务逻辑
  │     ├─ sorts := ContentSortModel.GetAll()
  │     ├─ list  := ContentModel.GetListByScode(scode, page, size)
  │     └─ return helper.AddSortName(list, sorts)  # 加工数据
  │
  ├─ ContentModel.GetListByScode()   # model：GORM 查询
  │     └─ db.Where("scode = ? AND status = 1", scode).Find(&list)
  │
  └─ Render(c, "content/index.html", data)  # 后台 pongo2 渲染
```

### Admin 控制器清单

约 30 个控制器，分三组，均嵌入 `common.BaseController`：

- **顶层**：`IndexController`（首页/登录/上传/验证码）、`UpgradeController`、`ImageExtController`（图片清理）
- **content/**：Content、ContentSort、Single、Media、Company、Site、Slide、Link、Message、Tags、Label、Form、Model、ExtField、DeleCache
- **member/**：Member、MemberGroup、MemberField、MemberComment
- **system/**：Menu、User、Role、Config、Area、Type、Syslog、Database

每个控制器的标准方法集是 `Index`（列表）/ `Add`（新增）/ `Mod`（编辑）/ `Del`（删除），配 `*CatchAll` 变体处理 PbootCMS 模板生成的带参数 URL。

### 配置读写机制

- **读**：`model.GetConfigValue(name, defaultVal)` — 查 `ay_config` 表
- **渲染注入**：`Render.go` 的 `loadConfigToData()` 将全部配置注入模板数据
- **写**：`ConfigController.Index` POST 时遍历预定义配置项数组逐个 upsert

### 通知消息常量

`apps/common/notice.go` 统一管理所有提示文案（繁体中文），项目硬约束：**禁止硬编码字符串，必须引用此处常量**。带变量的消息用函数，固定消息用常量。

### 权限模型

角色 `ay_role`（如 R100 超级管理员）关联权限 URL `ay_role_level`，用户 `ay_user.rcodes` 持有角色码。`AdminController.InitAdmin` 负责加载当前用户角色与可访问 URL。`NoAuthCheckControllers` 清单（Upgrade、ImageExt）跳过权限校验。

---

## 八、后台模板引擎

### PbootCMS 语法转换器

后台使用 pongo2 模板引擎，但 PbootCMS 原版模板使用 PHP 语法。`core/basic/view.go` 的 `convertPbootToPongo2()` 函数通过 20+ 个正则表达式实现语法转换：

| PbootCMS 语法 | 转换后 pongo2 语法 |
|---------------|-------------------|
| `{include file='common/head.html'}` | 内联展开（递归 5 层） |
| `{foreach $var(key,value)}` | `{% for valN in Var %}` |
| `{if($var->field==1)}` | `{% if Var.Field == 1 %}` |
| `{$var->field}` | `{{ Var.Field }}` |
| `{url./admin/path}` | 小写化路径 |
| `{$session.xxx}` | `{{ SessionXxx }}` |
| `{$get.xxx}` | `{{ GetXxx }}` |
| `{CMSNAME}` | `{{ CmsName }}` |
| `{php}...{/php}` | 清除 |

### 命名转换

`flattenData()` 将所有数据 key 从 snake_case 转为 PascalCase（含特殊缩写 IP/URL/API/DB/CMS/HTML 及复合词映射），使 PbootCMS 的 `{$config.page_size}` 能正确映射到 pongo2 的 `{{ Config.PageSize }}`。

### 自定义过滤器

`long2ip`、`safe`、`truncate`、`add`

### 模板渲染流程

`common.Render(c, tplPath, data)` 执行：
1. 注入 Session 信息（`session_xxx`）
2. 注入 CMS 常量（CmsName、CoreVersion、AppVersion、AppThemeDir 等）
3. 加载数据库配置到 `Config` map
4. 注入 GET 参数（`get_xxx`）
5. 构建菜单树（`MenuTree`、`MenuModels`）
6. snake_case → PascalCase 转换
7. 获取编译缓存的 pongo2 模板并执行

> **不可改动**：`Render.go` 中的 `AppThemeDir = "/static/admin"` 已写死，后台所有模板的 CSS/JS/layui 路径都强依赖它。把后台静态资源移回 `apps/admin/view/default/` 会破坏全部后台页面。

### pongo2 模板转换陷阱（必讀）

以下問題反覆出現，修改模板前務必確認：

#### 陷阱 1：config 變量有兩種語法，處理路徑不同

PbootCMS 模板中 config 變量有**單數**和**複數**兩種寫法，走不同的轉換路徑：

| 模板語法 | 場景 | 處理函數 | 轉換結果 |
|---------|------|---------|---------|
| `[$config.xxx]` | 條件判斷（無 `.value`） | `processPongo2If` → `pongo2DataKey` | `Config.Xxx` |
| `[$configs.xxx.value]` | 條件判斷（帶 `.value`） | `processConfigVars`（第 1 個正則） | `Config.Xxx` |
| `{$configs.xxx.value}` | 輸出值 | `processConfigVars`（第 2 個正則） | `{{ Config.Xxx }}` |

**關鍵**：兩條路徑最終都產出 `Config.Xxx`（PascalCase），但執行時機不同。`processConfigVars` 在管道中最先執行，確保複數形式在任何其他正則前被轉換。

#### 陷阱 2：pongo2DataKey 必須處理 config. 前綴

`pongo2DataKey` 函數處理三種前綴，每種的命名空間映射不同：

| 前綴 | 轉換結果 | 規則 |
|------|---------|------|
| `session.xxx` | `SessionXxx` | 拼前綴後 PascalCase |
| `get.xxx` | `GetXxx` | 拼前綴後 PascalCase |
| `config.xxx` | `Config.Xxx` | **單數命名空間**，轉 `Config.` 下的 PascalCase |
| 其他 | `XxxYyy` | 直接 PascalCase |

**陷阱**：如果 `pongo2DataKey` 遺漏 `config.` 前綴處理，`config.admin_check_code` 會被當作普通變量轉為 `Config.adminCheckCode`（camelCase），而 `loadConfigToData` 存儲的鍵是 `Config.AdminCheckCode`（PascalCase）。大小寫不匹配導致 pongo2 找不到值，條件永遠為 false。

#### 陷阱 3：PHP 的 `"0"` 是 falsy，pongo2 的 `"0"` 是 truthy

PbootCMS PHP 模板中 `{if([$config.xxx])}` 靠 PHP 的 falsy 機制判斷：`"0"` 為 falsy，`"1"` 為 truthy。但 pongo2 中**非空字符串都是 truthy**，`"0"` 也是 truthy。

**解決方案**：所有配置開關的條件判斷必須用顯式比較，不能用 truthy 判斷：

```html
<!-- ❌ 錯誤：PHP 可行，pongo2 中 "0" 也是 true -->
{if([$config.admin_check_code])}

<!-- ✅ 正確：顯式比較 -->
{if([$config.admin_check_code]!='0')}
```

#### 陷阱 4：processConfigVars 的命名空間必須是 Config（單數）

`processConfigVars` 早期版本生成 `Configs.Xxx`（複數），但 `loadConfigToData` 存儲為 `Config`（單數）。單複數不匹配導致配置頁面的所有 radio 條件失效。

**規則**：config 相關的 pongo2 變量命名空間固定為 `Config`（單數），不要用 `Configs`。

#### 陷阱 5：模板緩存不會自動失效

`GetAdminView` 使用內存緩存（`templateCache map`），修改 `.html` 模板文件後**必須重啟服務**才會生效。開發時可刪除 `runtime/pongo2_debug_*.html` 調試文件確認轉譯結果。

#### 調試方法

轉譯後的 pongo2 模板會輸出到 `runtime/pongo2_debug_<tplname>.html`，可直接查看轉譯結果是否符合預期。例如 `index.html` → `pongo2_debug_index.html`，`system/config.html` → `pongo2_debug_system_config.html`。

### 模板辅助函数 helper

`apps/admin/helper/template_helpers.go` 提供后台模板可调用的辅助函数：

- `StructToMap` — struct 转 PascalCase key 的 map（pongo2 兼容）
- `BuildSortSelectHTML` / `BuildSortTreeData` — 栏目树形下拉、treetable 排序
- `BuildPagebarHTML` — 后台分页 HTML
- `AddSortName` / `AddSonField` — 给内容列表补充栏目名、是否有子节点
- `ParseWildcardAction` — 解析 PbootCMS 模板生成的 `/scode/123/field/status/value/0` 形态 URL

### 后台模板组织

```
apps/admin/view/
├── common/          # 公共模板（head.html 含完整头部+侧边栏，foot.html，ueditor.html）
├── content/         # 内容管理模板（17个）
├── system/          # 系统管理模板（13个）
├── member/          # 会员管理模板（4个）
└── index.html       # 登录页
```

模板引用方式：`{include file='common/head.html'}`（注意是 `common/` 不是 `comm/`）

### 后台静态资源

集中放在 `static/admin/`，子目录（css/js/images/font-awesome/layui/extend）与 PHP 原版 `apps/admin/view/default/` 一一对应。`main.go` 用 `r.Static("/static", "static")` 统一挂载。

---

## 九、前台模板引擎与标签系统

### 双引擎架构

项目使用双模板引擎：后台用 pongo2 + 语法转换器，前台用自研 TagParser + TemplateStore。

### TemplateStore（模板存储与热重载）

- 启动时全量加载 `template/default/` 下所有 `.html`/`.htm` 文件到内存（跳过 static 目录）
- 使用 `fsnotify` 监听文件变更，实现模板热重载
- `Render(name)` 方法从内存取出模板 + 递归解析 `{include file=xxx}`（防环，最深 10 层）
- 读写使用 `sync.RWMutex` 保证线程安全

### TagParser 解析流水线

前台渲染分两个独立阶段，由 `FrontController` 串联：

**阶段 A**（`engine.go` 的 `TemplateStore.Render`）：从内存缓存取模板，递归解析 `{include}`，返回纯文本（含未解析的业务标签）。

**阶段 B**（`tags.go` 的 `TagParser.Render`）：处理所有 `gboot:` 标签，按以下固定顺序执行：

1. 提取 `{gboot:pre}...{/gboot:pre}` 保护块（暂存 JS/CSS 中的花括号）
2. 处理 include（no-op，已被阶段 A 解决）
3. 预解析配对标签参数中的单标签（如 `{gboot:list scode={sort:scode}}` → `{gboot:list scode=5}`）
4. 深度预解析嵌套 `{gboot:xxx}` 单标签（最多 10 次迭代）
5. 处理配对标签（list/nav/slide/link 等，17 种）
6. 处理单标签（site/sort/content/page 等）
7. 处理 if 条件标签（0/1/2/3 四级深度）
8. 恢复 pre 保护块

> **顺序为何刚性**：**single 必须在 if 之前**——否则 `{gboot:if({sort:scode}==1)}` 中的 `{sort:scode}` 无法解析。**pair 先于 single**——pair 循环体经 `ReplaceInnerTags` 用 `[xxx:field]` 占位，避免被 single 误吞。改解析顺序会引发连锁 bug。

### 标签语法规范

**单标签**：`{site:title}`、`{gboot:sitetitle}`、`{sort:name}`、`{content:title}`

**配对标签**：
```
{gboot:list scode=5 num=10}
    [list:title]
    [list:link]
{/gboot:list}
```
内部标签用方括号 `[prefix:field]`，每条记录有 `[prefix:i]`（1-based）和 `[prefix:n]`（0-based）序号。

**条件标签**：
```
{gboot:if(condition)}
    true 分支
{else}
    false 分支
{/gboot:if}
```
支持 4 层嵌套（`{gboot:if}`、`{gboot:1if}`、`{gboot:2if}`、`{gboot:3if}`），对应 else 为 `{else}`/`{1else}`/`{2else}`/`{3else}`。嵌套通过编号前缀实现：`3if` 最先解析（最内层）→ `2if` → `1if` → `if`（最外层）。

运算符：`==` `!=` `>` `<` `>=` `<=` `%`（两侧均非空判断）和 `&&` `||`。

### 已注册标签清单

**单标签**：site、gboot（兼容前缀）、company、sort、content、page、user、label、qrcode、position、pagetitle、pagekeywords、pagedescription、httpurl、pageurl、islogin、checkcode、msgaction、scaction、form、keyword、commentaction、lgpath、login、register、ucenter、loginstatus、registerstatus、commentstatus

**配对标签**：list、nav、sort_loop、content_loop、slide、link、loop、tags、pics、checkbox、message、formlist、search、selectall、select、comment（未实现）、commentsub（未实现）、mycomment（未实现）

### 完整标签目录

#### 站点 / 全局单值

| 标签 | 作用 |
|------|------|
| `{gboot:sitetitle}` / `sitesubtitle` / `sitekeywords` / `sitedescription` | 站点信息 |
| `{gboot:sitelogo}` / `siteicp` / `sitecopyright` / `sitestatistical` | LOGO/备案/版权/统计 |
| `{gboot:sitetplpath}` | 模板资源路径 `/template/{theme}/static` |
| `{gboot:pagetitle}` / `pagekeywords` / `pagedescription` | 页面级（Content>Sort>Site 优先级） |
| `{gboot:position separator=/ indextext=首页}` | 面包屑（递归父栏目链） |
| `{gboot:httpurl}` / `siteurl` / `pageurl` | 当前/站点 URL |
| `{gboot:islogin}` / `login` / `register` / `ucenter` | 会员状态与入口 |
| `{gboot:checkcode}` / `msgaction` / `scaction` / `commentaction` | 验证码/表单提交地址 |
| `{gboot:qrcode string=xxx}` | 二维码图片 |
| `{gboot:form fcode=X}` | 表单提交 URL `/message?fcode=X` |

#### 循环标签（pair）

| 标签 | 属性 | 循环体内标签 |
|------|------|-------------|
| `{gboot:list scode=X num=10 order=date page=1}` | 栏目/数量/排序/分页 | `[list:id/title/content/ico/date/link/visits/ext_*]` |
| `{gboot:content id=X scode=Y}` | 单条内容 | `[content:xxx]` |
| `{gboot:sort scode=1,2,3}` | 栏目列表 | `[sort:name/scode/link/rows]` |
| `{gboot:nav parent=X num=10}` | 导航 | `[nav:name/scode/link/rows]` |
| `{gboot:slide gid=1 num=5}` | 轮播图 | `[slide:src/pic/link/title]` |
| `{gboot:link gid=1 num=10}` | 友情链接 | `[link:logo/link/title]` |
| `{gboot:tags num=10}` | 标签云 | `[tags:tag/text/link]` |
| `{gboot:pics}` | 当前内容多图 | `[pics:src]` |
| `{gboot:message num=10}` | 最新留言 | `[message:contacts/mobile/content]` |
| `{gboot:search num=10 page=1 scode=X}` | 搜索结果 | 同 list |
| `{gboot:formlist fcode=X num=10}` | 自定义表单数据 | 动态表 `ay_diy_*` |
| `{gboot:loop start=1 end=10}` | 纯数字循环 | `[loop:n/i/index]` |
| `{gboot:select field=xxx}` | 扩展字段筛选 | `[select:link/value/current]` |
| `{gboot:comment}` / `commentsub` / `mycomment` | 评论 | 未实现，返回空 |

#### 公司信息

两类语法等价：`{gboot:companyname}` 或 `{company:name}`，均路由到同一数据源 `ctx.Company`。

### 内部标签参数修饰

| 参数 | 示例 | 说明 |
|------|------|------|
| `len=N` | `[list:title len=12]` | 按字符截断 |
| `lencn=N` | `[list:title lencn=30]` | 按中文宽度截断（中文按2字宽） |
| `drophtml=1` | `[list:content drophtml=1]` | 去除 HTML 标签 |
| `dropblank=1` | — | 去除空白 |
| `style=Y-m-d` | `[list:date style=Y-m-d]` | 日期格式化（PHP 格式转 Go） |
| `decode=1` | `[content:content decode=1]` | HTML 实体解码 |
| `substr=0,10` | `[content:title substr=0,10]` | 子字符串 |
| `more=...` | — | 追加文本 |

执行顺序：drophtml → dropblank → len/lencn → style → decode → substr。drophtml 必须在 len 之前以防截断 HTML 标签。

### Context 结构体

```go
type Context struct {
    Sort        *model.ContentSort      // 当前栏目
    Content     *model.Content          // 当前内容
    Site        *model.Site             // 站点配置
    Company     *model.Company          // 公司信息
    Page        map[string]interface{}  // 分页信息
    Member      *model.Member           // 当前会员
    Keyword     string                  // 搜索关键词
    CurrentPage int                     // 当前页码
    Filters     map[string]string       // ext_ 筛选参数
}
```

`buildContext()` 每次请求执行：初始化 Page/Filters → 收集 `ext_` URL 参数 → 从数据库加载 Site 和 Company。

### 分页机制

- 模板中 `page=1` 参数启用分页：`{gboot:list scode=5 num=15 page=1}`
- 页码通过 `?page=N` 传递
- 分页标签：`{page:numbar}`（数字分页条）、`{page:index}`、`{page:pre}`、`{page:next}`、`{page:last}`、`{page:current}`、`{page:count}`、`{page:rows}`
- 分页 basePath 保留所有 `ext_` 筛选参数
- 分页模板：`{include file=comm/page.html}`

### 条件求值机制

`{gboot:if}` 因正则无法可靠处理嵌套括号，采用**手写解析器**按括号深度匹配。求值流程（`EvalIfCondition`）：

1. 危险模式黑名单过滤（`eval/system/exec/$_GET` 等命中即 false）
2. 变量解析（`[content:xxx]`、`[sort:xxx]`、`{gboot:xxx}` 替换为实际值，未命中返回 `"0"`）
3. 逻辑表达式（`&&`/`||`）
4. 简单表达式（`==/!=/>=/<=/>/<`/`%`，`%` 语义为"两侧均非空"）
5. 数值比较先 int 后 float，最后字符串比较

### 前台模板文件

```
template/default/
├── comm/            # 公共模板
│   ├── head.html    # 头部导航（Bootstrap/Swiper 引入、导航栏）
│   ├── foot.html    # 底部（公司信息、JS 脚本）
│   ├── page.html    # 分页组件
│   ├── position.html # 面包屑
│   └── sortnav.html  # 子分类导航
├── index.html       # 首页
├── list.html        # 通用列表页
├── content.html     # 通用内容页
├── search.html      # 搜索页
├── message.html     # 留言页
├── about.html       # 关于（单页）
├── news.html / newslist.html
├── product.html / productlist.html
└── static/          # 前台静态资源
```

---

## 十、前台路由与模板映射

### URL 生成规则

**栏目链接**（sortToMap）：
1. `outlink`（外部链接）优先
2. `filename` → `/filename/`
3. `urlname` → `/urlname/`
4. fallback → `/sort/scode`

**内容链接**（contentToMap）：
1. `outlink` 优先
2. `filename` → `/filename`
3. fallback → `/content/{id}`

### ContentPage 兜底逻辑

NoRoute 处理基于 filename/urlname 的伪静态 URL，优先级：

1. 去除 `.html`/`.htm` 后缀
2. 优先查 `content.filename` → fallback `content.urlname`（status=1）→ 内容详情页
3. 否则查 `sort.filename` → fallback `sort.urlname` → 栏目列表页
4. 都不命中 → 404

### renderSortPage 模板选择

- 查 `ay_model` 表的 `type` 字段
- `type == 1`（单页模型）→ 使用 `sort.ContentTpl`（如 about.html）
- 其他（列表模型）→ 使用 `sort.ListTpl`（如 list.html）
- 模板为空 → 默认 `list.html`

---

## 十一、扩展子系统

### 邮件 mail

`apps/common/mail/mailer.go` 用 `go-mail` 实现 SMTP 发信，配置从 `ay_config` 读取（`smtp_server/username/password/port/ssl`）。三个入口：`SendMail`（通用）、`SendTestMail`（测试）、`SendNotifyMail`（留言/表单通知，读 `message_send_to` 收件人）。

### Webhook

`apps/common/webhook/webhook.go` 在留言/表单提交时**异步** POST JSON 到 `ay_config.webhook_url`（为空则跳过）。异步 `go func()` 不阻塞用户请求，10 秒超时。

### 上传

`apps/common/upload/` 处理后台文件上传，写入 `static/upload/image/{yyyyMM}/`、`file/`、`video/`。`static/upload/` 被 `.gitignore` 排除，需用户自行备份。

### 验证码

后台登录（`/admin/index/checkCode`，开关 `admin_check_code`）与前台留言（`/api/checkcode`，开关 `message_check_code`）各一个入口。

### 自定义表单

`{gboot:form fcode=X}` 生成提交地址 `/message?fcode=X`。`FrontController.Message` 按 `fcode` 区分：`fcode=1` 或空 → 写 `ay_message`；`fcode>=2` → 调 `handleFormSubmit` 动态 INSERT 到 `ay_diy_*` 表（表名由 `ay_form.table_name` 指定，无 struct）。提交后触发邮件与 webhook 通知。

### 媒体库

`MediaController` 扫描 `static/upload/` 构建媒体库，`ay_media_mark` 标记受保护文件。写操作经 `mediaplugin` 自动失效缓存。

---

## 十二、开发约束与规范

### 硬约束（必须遵守）

1. **数据库字段名**：使用 `mcode` 而非 `modelcode`；查询自定义字段时 SQL 必须包含 `COALESCE(status,1) = 1` 处理 NULL 值
2. **Form action URL**：必须使用小写 `/admin/content/mod`，大写会导致 POST 数据丢失
3. **模板条件判断**：字段类型必须用整数比较（`val1.Type==1`）而非字符串比较
4. **日期处理**：日期字符串必须用 `time.Parse("2006-01-02 15:04:05", v)` 解析后存入数据库
5. **通知消息**：必须使用 `apps/common/notice.go` 中的常量，禁止硬编码字符串
6. **编辑表单脏检查**：无变更时不写库、不发成功通知
7. **构建产物**：必须输出到 `bin/` 目录
8. **文档**：所有 `.md` 文档放在 `docs/` 文件夹
9. **根目录**：仅保留必要文件，临时文件不得滞留
10. **首页 Banner**：`.swiper-container`/`.swiper-wrapper`/`.swiper-slide` 最大高度 400px + `overflow: hidden`
11. **模板文件**：开头的 BOM 字符（`U+FEFF`）必须清除
12. **导航栏**：使用 `sticky-top` 实现粘性置顶
13. **Layui checkbox**：多选字段不要使用 `lay-skin="primary"`
14. **AppThemeDir**：禁止改动 `Render.go` 的 `AppThemeDir = "/static/admin"`（破坏全部后台页面）
15. **解析器管道顺序**：禁止调换 single/if/pair 处理顺序
16. **ay_content_ext**：禁止删除物理列（熔断保护，仅删定义记录）

### 工程约定

- URL 参数中的 `&mcode=3` 需截断 `&` 或 `?` 后缀提取有效 ID
- 多选框数据收集需同时检查 `fieldName[]` 和 `fieldName[0]`、`fieldName[1]` 格式
- 零值 `time.Time` 显示为空字符串
- Pongo2 模板使用控制器预格式化的 `DateStr`，不用不存在的 `date` 过滤器
- 所有 admin 控制器嵌入 `common.BaseController`
- 业务逻辑放 Service 层，Controller 只做参数与响应
- 新增表用 GORM `AutoMigrate`，不用手写 SQL（`ay_content_ext` 例外）
- 动态配置用 `model.GetConfigValue`，不硬编码
- 移动文件用 `git mv` 保留历史
- 对 `data/pbootcms.db` 禁止用 `git checkout --`（会还原 SQLite header）

### 经验教训

- Layui form 模块会自动将 `[]` 名称的 checkbox 重新索引为 `fieldName[0]`、`fieldName[1]` 格式
- 存储未解析的日期字符串到 `time.Time` 字段会显示零值日期（0001-01-01）
- Layui 表单中带 `lay-submit` 的排序按钮需设置 `lay-submit` 属性防止表单拦截

---

## 十三、开发指南

### 新增一个后台 CRUD 功能

以"公告管理"为例，按 MVC+S 四层依次创建：

1. **Model**：在 `apps/admin/model/content/` 新建 `NoticeModel.go`，定义 `Notice` struct（自动映射 `ay_notice`），加查询方法；在 `migrate.go` 的 `AutoMigrate` 注册 `&Notice{}`；在 `db.go` 加类型别名 `type Notice = content.Notice`
2. **Service**：在 `apps/admin/service/content/` 新建 `NoticeService.go`，封装 `GetList/Add/Mod/Del` 业务逻辑与数据加工
3. **Controller**：在 `apps/admin/controller/content/` 新建 `NoticeController.go`，嵌入 `common.BaseController`，持有 `svc svc.NoticeService`，实现 `Index/Add/Mod/Del`
4. **Route**：在 `apps/route/route.go` 的 `SetupAdminRoutes` 注册 `/admin/notice/index` 等路由
5. **View**：在 `apps/admin/view/default/content/` 新建 `notice.html`（原版 PHP 模板语法，由 view.go 转译）
6. **Menu**：在 `seed.go` 的 `seedMenus` 增加菜单项（`mcode/pcode/url`），或在后台"系统菜单"手动添加

### 新增一个前台模板标签

以新增 `{gboot:notice num=5}` 为例：

1. 在 `tags.go` 的正则定义区加 pair 标签正则，并在 `pairTagOrder` 注册顺序
2. 在 `providers.go` 的 `RegisterAllProviders` 注册 provider：`p.Register("notice", provideNotice)`
3. 实现 `provideNotice`：解析参数 `num`，查 `ay_notice`，用 `ReplaceInnerTags` 替换循环体 `[notice:xxx]`
4. 循环体字段在 provider 内构造 map（含 `n/i` 计数与业务字段）
5. 注意 pair 必须在 single 之前处理；若标签可嵌套 `{gboot:if}`，参考 `processInnerIfTags`

### 新增内容扩展字段

扩展字段走 `ay_extfield` + `ay_content_ext` 动态机制，**不需要改 struct 或 migrate**：

1. 后台"全局配置 → 模型字段"新增字段定义，写入 `ay_extfield`
2. `ExtFieldModel` 自动用 `ALTER TABLE ay_content_ext ADD COLUMN` 加列
3. 前台 `{gboot:list}` 自动 LEFT JOIN 返回 `ext_` 字段，`[list:ext_xxx]` 可直接用
4. 筛选通过 URL 参数 `?ext_xxx=value`，写入 `ctx.Filters`，list provider 自动 JOIN 过滤

---

## 十四、开发环境与构建

### 环境要求

- Go 1.25.0+
- GOPROXY：`https://goproxy.cn,direct`（国内镜像）
- 无需 CGO（使用纯 Go SQLite 驱动 glebarez/sqlite）

### 构建与启动

```powershell
# PowerShell
.\build.ps1            # go mod tidy + go build → bin/pbootcms-go.exe
.\build.ps1 -Run       # 构建后立即运行

# 或手动
go build -o bin/pbootcms-go.exe .
```

### 运行依赖目录

| 目录 | 必须性 | 说明 |
|------|--------|------|
| `data/pbootcms.db` | 必须 | 缺失时 AutoMigrate+seed 自动创建空库 |
| `config/config.json` | 必须 | 库路径与端口 |
| `template/` | 必须 | 前台模板 |
| `static/admin/` | 必须 | 后台静态资源 |
| `static/upload/` | 运行时创建 | 上传产物，不入库 |
| `runtime/` | 运行时创建 | 缓存 |

### 访问地址

- 前台：`http://localhost:8080/`
- 后台：`http://localhost:8080/admin`
- 默认账号：`admin` / `admin`

### 验证清单

```bash
# 前台
curl -I http://localhost:8080/            # 200
curl -I http://localhost:8080/aboutus     # 200
# 后台
curl -I http://localhost:8080/admin/      # 200 登录页
# 静态资源
curl -I http://localhost:8080/static/admin/css/comm.css            # 200
curl -I http://localhost:8080/template/default/static/bootstrap/css/bootstrap.min.css # 200
```

---

## 十五、已实现功能清单

### 后台管理

- 管理员登录/登出（密码双重 MD5 + 算术验证码 + 登录锁定）
- 后台首页 Dashboard
- 内容管理：文章/产品的增删改查、批量排序、扩展字段
- 栏目管理：树形结构、自定义 URL、模板选择
- 单页管理：关于我们等单页内容
- 站点信息、公司信息管理
- 幻灯片、友情链接管理
- 自定义标签、内链标签管理
- 内容模型、扩展字段管理
- 自定义表单管理
- 媒体库（图片管理）
- 系统配置（70+ 配置项）
- 菜单管理、角色权限、用户管理
- 数据库备份/优化
- 系统日志、区域管理
- 会员管理、会员组、会员字段、文章评论

### 前台展示

- 首页（轮播图、导航、产品展示）
- 列表页（分页、ext_ 筛选、子分类导航）
- 内容详情页（扩展字段、面包屑、上一篇/下一篇）
- 搜索页（带分页、关键词高亮）
- 标签页
- 留言页
- 单页（关于我们等）
- 模板热重载（fsnotify）
- .html 伪静态 URL
- 标题颜色渲染
- 访问量统计

---

## 十六、已知问题与路线

### 未完成功能

| 功能 | 状态 | 位置 |
|------|------|------|
| 评论 `{gboot:comment}` / `commentsub` / `mycomment` | 未实现，返回空 | `providers.go` |
| 会员前台登录/注册/中心 | 部分，标签返回固定值 | `providers.go` |
| HTML 静态缓存 `tpl_html_cache` | 未实现 | 配置项存在但无逻辑 |
| API 模块 | 未移植 | PHP `apps/api` |

### 技术债

- **密码哈希**：双重 MD5 不安全，应迁移 bcrypt（需处理存量密码）
- **SQLite 单连接**：高并发写受限，可考虑 MySQL/PostgreSQL 适配
- **Session 非持久化**：内存 Session 重启丢失，多实例部署需替换为 Redis
- **AdminController/HomeController 占位**：定义了但未嵌入，易误导新人
- **评论功能**：三个评论标签 provider 直接返回空，需补全

### 接手第一步建议

1. 跑通 `build.ps1`，用 `admin/admin` 登录后台，确认菜单与内容管理可用
2. 访问前台 `/`、`/aboutus`、`/article`，对照 `template/default/` 模板理解 `gboot` 标签
3. 读 `apps/home/controller/front.go` 的 `ContentPage`，理解 pretty URL 解析
4. 读 `apps/common/parser/tags.go` 的 `Render` 管道，再读 `providers.go` 的 `list` provider
5. 按第十三章新增一个简单 CRUD 功能练手

---

## 十七、关键文件索引

| 文件 | 职责 |
|------|------|
| `main.go` | 程序入口，启动流程，路由注册 |
| `config/config.go` | 配置结构体定义与加载（sync.Once 单例） |
| `core/db/db.go` | 数据库初始化（GORM + glebarez/sqlite） |
| `core/basic/view.go` | 后台模板引擎（pongo2 + PbootCMS 语法转换） |
| `core/mediaplugin/plugin.go` | GORM 媒体缓存失效插件 |
| `apps/route/route.go` | 后台路由集中注册 |
| `apps/common/BaseController.go` | 基础控制器（JSON 响应、批量排序） |
| `apps/common/Render.go` | 后台模板渲染入口（配置注入、菜单树、AppThemeDir） |
| `apps/common/session.go` | 自实现内存 Session（PbootGo Cookie） |
| `apps/common/notice.go` | 通知消息常量（硬约束） |
| `apps/common/middleware/auth.go` | 后台认证中间件 |
| `apps/common/middleware/path_rewrite.go` | URL 重写映射表 |
| `apps/common/parser/tags.go` | TagParser 核心解析流水线 |
| `apps/common/parser/engine.go` | TemplateStore 模板存储与热重载 |
| `apps/common/parser/providers.go` | 所有前台标签 Provider |
| `apps/common/parser/if_eval.go` | 条件表达式求值 |
| `apps/common/mail/mailer.go` | SMTP 邮件发送 |
| `apps/common/webhook/webhook.go` | 异步 Webhook 推送 |
| `apps/admin/helper/template_helpers.go` | 后台模板辅助函数 |
| `apps/home/controller/front.go` | 前台控制器 |
| `apps/admin/controller/IndexController.go` | 登录认证 |
| `apps/admin/model/db.go` | 全局 DB 实例 + 29 个类型别名 |
| `apps/admin/model/content/migrate.go` | AutoMigrate 注册 |
| `apps/admin/seed/seed.go` | 种子数据初始化 |

package models

import (
	"math"
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email        string `gorm:"size:100" json:"email"`
	PasswordHash string `gorm:"size:255" json:"-"`
	Avatar       string `gorm:"size:500" json:"avatar"`
	// 未开会员时修改过头像 → 加入会员后不再发放头像会员经验（仅影响经验，不影响积分）。
	AvatarChangedBeforeMember bool   `gorm:"default:false" json:"avatarChangedBeforeMember"`
	Bio                       string `gorm:"size:500" json:"bio"`
	Nickname                  string `gorm:"size:50" json:"nickname"`
	Phone                     string `gorm:"size:20" json:"phone"`
	// 遗留设备指纹（文件版 UUID）：软绑定/迁移参考值，非权威。存量账号平滑升级用；新客户端以 DeviceIDHw 为唯一权威绑定依据。
	DeviceID string `gorm:"size:200;index" json:"deviceId"`
	// 硬件指纹（sha256 前缀 hash）：权威绑定依据，非空即进入严格「一设备一账号」模式。
	DeviceIDHw string `gorm:"size:128;index" json:"deviceIdHw"`
	// 上次自助解绑时间，用于 72h 冷却；nil 表示从未解绑。
	DeviceUnbindAt *time.Time `json:"deviceUnbindAt,omitempty"`
	// 5 位数字登录码：可作为登录账号（替代主键 ID，避免暴露主键）。
	// 注册时生成且唯一；存量用户由启动迁移补齐。不在此处加 uniqueIndex，
	// 唯一性由应用层（注册 / 改名）保证，避免存量空串冲突导致迁移失败。
	LoginCode string `gorm:"size:20;index" json:"loginCode"`
	// 计算字段：用户名是否符合新规范（<5 显示长度 / 纯数字 / 含违禁词 视为不合规）。
	// 旧客户端注册的低位数用户名会标记为 false，客户端据此强制改名。
	UsernameValid    bool       `gorm:"-" json:"usernameValid"`
	MembershipLevel  string     `gorm:"size:20;default:'free'" json:"membershipLevel"`
	MembershipExpiry *time.Time `json:"membershipExpiry,omitempty"`
	// ===== 新会员体系字段（双组 VIP/SVIP + 经验等级，1.0.5 双端启用；旧 MembershipLevel 字符串保留给老客户端） =====
	// MembershipGroup：用户所属新体系组，空串=未加入新体系。
	MembershipGroup     string     `gorm:"size:10;default:''" json:"membershipGroup"`
	MembershipExp       int        `gorm:"default:0" json:"membershipExp"`              // 累计经验
	MembershipTier      int        `gorm:"default:1" json:"membershipTier"`             // 当前等级（新体系，1-based）
	MembershipSubType   string     `gorm:"size:10;default:''" json:"membershipSubType"` // 订阅类型：空/monthly/annual（决定年费加速倍率）
	MembershipExpiresAt *time.Time `json:"membershipExpiresAt,omitempty"`               // 新体系订阅到期时间
	ExpUpdatedAt        *time.Time `json:"expUpdatedAt,omitempty"`                      // 最近一次经验/订阅刷新时间
	// 每日免费下载次数单用户覆盖：NULL=跟随会员等级；>0=固定该次数（有限）；<=0=不限。
	// 用于在「会员等级」之外，给个别用户单独设定每日免费下载额度（如永久会员但限 10 次/天）。
	DailyFreeCountOverride *int    `gorm:"column:daily_free_count_override" json:"dailyFreeCountOverride,omitempty"`
	Points                 int     `gorm:"default:0" json:"points"`
	Balance                float64 `gorm:"default:0" json:"balance"`
	Role                   string  `gorm:"size:20;default:'user'" json:"role"`                   // user / admin
	P2PExcluded            bool    `gorm:"column:p2p_excluded;default:false" json:"p2pExcluded"` // 不参与 P2P 图片存储/分享
	Status                 string  `gorm:"size:20;default:'active'" json:"status"`
	SocialAccounts         string  `gorm:"type:text" json:"socialAccounts,omitempty"` // JSON
	// 分平台最后活跃时间：用户在任何一端（已登录）触发请求即刷新对应字段。
	// 用于后台判断用户「在线/离线」状态，按平台分别统计。
	LastActiveWeb    *time.Time     `json:"lastActiveWeb,omitempty"`
	LastActiveClient *time.Time     `json:"lastActiveClient,omitempty"`
	LastActiveApp    *time.Time     `json:"lastActiveApp,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// Category 文章分类目录
type Category struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"size:100;not null" json:"name"`
	Slug              string         `gorm:"size:100;uniqueIndex" json:"slug"`
	ParentID          *uint          `json:"parentId,omitempty"`
	Sort              int            `gorm:"default:0" json:"sort"`
	Description       string         `gorm:"size:500" json:"description"`
	Status            string         `gorm:"size:20;default:'active'" json:"status"`
	ThumbStyle        string         `gorm:"size:20;default:'horizontal'" json:"thumbStyle"` // horizontal/vertical/waterfall/list
	AspectRatio       string         `gorm:"size:10;default:'16:9'" json:"aspectRatio"`      // 16:9, 4:3, 3:2, 21:9, 1:1（仅 horizontal 有效）
	CardsPerRow       int            `gorm:"default:0" json:"cardsPerRow"`                   // 每行显示几个（电脑端），0=按前端默认
	CardsPerRowMobile int            `gorm:"default:0" json:"cardsPerRowMobile"`             // 每行显示几个（手机端），0=跟随电脑端
	Icon              string         `gorm:"size:50;default:''" json:"icon"`                 // emoji 或 Font Awesome 图标类名（如 fa-solid fa-house）
	ShowInTop         bool           `gorm:"default:true" json:"showInTop"`                  // 是否在顶部导航显示
	ShowInMobile      bool           `gorm:"default:true" json:"showInMobile"`               // 是否在手机端导航显示
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// MembershipLevel 会员等级（可后台动态配置）
type MembershipLevel struct {
	Key            string    `gorm:"size:30;primaryKey" json:"key"` // free / month / quarter / year / vip
	Name           string    `gorm:"size:100;not null" json:"name"`
	Price          float64   `gorm:"default:0" json:"price"`          // 售价
	Months         int       `gorm:"default:0" json:"months"`         // 周期月数（1=月费，3=季费，12=年费，9999=永久，0=按days）
	Days           int       `gorm:"default:0" json:"days"`           // 有效期天数，0=永久；months>0时优先按months换算
	Discount       float64   `gorm:"default:1" json:"discount"`       // 内容折扣 0-1
	DailyFreeCount int       `gorm:"default:0" json:"dailyFreeCount"` // 每天免费次数，0=不限
	Color          string    `gorm:"size:20" json:"color"`
	Icon           string    `gorm:"size:50" json:"icon"`
	Sort           int       `gorm:"default:0" json:"sort"`
	Permissions    string    `gorm:"type:text" json:"permissions,omitempty"` // JSON 权限列表
	Status         string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ===== 新会员体系（双组 VIP/SVIP + 经验等级，1.0.5 双端启用） =====
// 旧 MembershipLevel（扁平时长型）保留不动，继续服务老客户端；新体系与其并存过渡。

// MembershipGroup 会员组（vip / svip）。后台可自定义显示名（plus/pro）、最高等级、年费加速倍率、组图标。
// 组 key 固定为 vip / svip（两张种子记录），但除 key 外的字段均可后台修改。
type MembershipGroup struct {
	GroupKey         string    `gorm:"size:10;primaryKey" json:"groupKey"` // vip / svip
	GroupName        string    `gorm:"size:50" json:"groupName"`           // 后台自定义显示名，如 plus / pro / 超级会员
	MaxLevel         int       `gorm:"default:6" json:"maxLevel"`          // 该组最高等级（可分别设，如 SVIP=10）
	AnnualPlaceLevel int       `gorm:"default:2" json:"annualPlaceLevel"`  // 年费购买瞬间直达等级
	JoinPlaceLevel   int       `gorm:"default:5" json:"joinPlaceLevel"`    // 首次开通会员瞬间直达等级（用户需求：开通直升 VIP5）
	AnnualSpeed      float64   `gorm:"default:1.9" json:"annualSpeed"`     // 年费订阅/任务经验加速倍率（VIP 1.9 / SVIP 2.8）
	SubMonthlyExp    int       `gorm:"default:300" json:"subMonthlyExp"`   // 月费每月基础经验
	ExpCurveP        float64   `gorm:"default:1.4" json:"expCurveP"`       // 经验曲线指数 p（越大后期越难）
	MonthlyPrice     float64   `gorm:"default:0" json:"monthlyPrice"`      // 月费价（元，支持小数），1.0.5 客户端订阅收银用
	AnnualPrice      float64   `gorm:"default:0" json:"annualPrice"`       // 年费价（元，支持小数）
	QuarterlyPrice   float64   `gorm:"default:0" json:"quarterlyPrice"`    // 季费价（元，支持小数）
	IconJson         string    `gorm:"type:text" json:"iconJson"`          // 组图标：{"type":"preset"|"upload"|"url"|"svg","value":"..."}
	Status           string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// MembershipTier 某组下某等级：经验阈值 + 每日下载次数 + 图标 + 颜色。
// ExpThreshold 为累计经验阈值，升级判定 LevelForExp 直接读各 tier 的该字段（与曲线生成方式解耦）。
// 可由 SeedDefaultMembershipV2 的截图自定义曲线直接写入，也可后台手改任意一级。
type MembershipTier struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PBID          string    `gorm:"size:32;index" json:"pbId,omitempty"` // PocketBase 配置域拉取的记录 id（PB 管理的行非空）
	GroupKey      string    `gorm:"size:10;index" json:"groupKey"`       // vip / svip
	Level         int       `json:"level"`                               // 1..max_level
	Name          string    `gorm:"size:50" json:"name"`                 // 该等级展示名，如 "VIP3"
	ExpThreshold  int       `json:"expThreshold"`                        // 累计经验阈值（升到本级所需，L1=0）
	DownloadCount int       `json:"downloadCount"`                       // 每日下载次数，0=不限
	IconJson      string    `gorm:"type:text" json:"iconJson"`           // 等级图标（见上）
	Color         string    `gorm:"size:20" json:"color"`
	Sort          int       `json:"sort"`
	Status        string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// MembershipTaskExp 任务经验配置（签到/换头像/评论/点赞/收藏），全站单行。
type MembershipTaskExp struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CheckinExp       int       `gorm:"default:5" json:"checkinExp"`        // 每日签到经验
	AvatarExp        int       `gorm:"default:50" json:"avatarExp"`        // 首次换头像（一次性）
	CommentExp       int       `gorm:"default:2" json:"commentExp"`        // 每条评论经验
	CommentDailyCap  int       `gorm:"default:5" json:"commentDailyCap"`   // 每日评论经验次数上限
	LikeExp          int       `gorm:"default:1" json:"likeExp"`           // 每条点赞经验
	LikeDailyCap     int       `gorm:"default:10" json:"likeDailyCap"`     // 每日点赞经验次数上限
	FavoriteExp      int       `gorm:"default:1" json:"favoriteExp"`       // 每条收藏经验
	FavoriteDailyCap int       `gorm:"default:10" json:"favoriteDailyCap"` // 每日收藏经验次数上限
	UpdatedAt        time.Time `json:"updatedAt"`
}

// MembershipExpLog 经验流水（用于任务每日次数上限统计 + 审计）。
// Source：checkin / avatar / comment / like / favorite / subscription。
type MembershipExpLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	GroupKey  string    `gorm:"size:10;index" json:"groupKey"`
	Source    string    `gorm:"size:20;index" json:"source"`
	Exp       int       `json:"exp"`
	CreatedAt time.Time `json:"createdAt"`
}

// BuildCurve 生成从 L1 到 maxLevel 的累计经验阈值（升序）。
// C(L) = (12 × subMonthlyExp) × ((L-1)/(maxLevel-1))^p ，C(1)=0，C(maxLevel)=12×sub。
// 与 outputs/membership_calculator.html 同一公式，保持前后端一致。
func BuildCurve(maxLevel, subMonthlyExp int, p float64) []int {
	if maxLevel < 1 {
		maxLevel = 1
	}
	if maxLevel == 1 {
		return []int{0}
	}
	base := 12 * subMonthlyExp
	thresholds := make([]int, maxLevel)
	thresholds[0] = 0
	for L := 2; L <= maxLevel; L++ {
		ratio := float64(L-1) / float64(maxLevel-1)
		thresholds[L-1] = int(math.Round(float64(base) * math.Pow(ratio, p)))
	}
	return thresholds
}

// LevelForExp 根据累计经验返回当前等级（1-based）。thresholds 为 BuildCurve 结果（升序）。
func LevelForExp(exp int, thresholds []int) int {
	lv := 1
	for i, t := range thresholds {
		if exp >= t {
			lv = i + 1
		} else {
			break
		}
	}
	return lv
}

// Article 文章模型
type Article struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	Title         string  `gorm:"size:200;not null" json:"title"`
	Summary       string  `gorm:"size:500" json:"summary"`
	Content       string  `gorm:"type:longtext" json:"content"`
	Cover         string  `gorm:"size:500" json:"cover"`
	CoverLocal    bool    `gorm:"default:false" json:"coverLocal"` // 封面是否已本地化
	Tags          string  `gorm:"size:500" json:"tags"`            // comma-separated
	CategoryID    *uint   `gorm:"index" json:"categoryId,omitempty"`
	CategorySlug  string  `gorm:"-" json:"categorySlug,omitempty"` // 分类 slug（服务端注入，非 DB 字段）
	CategoryName  string  `gorm:"-" json:"categoryName,omitempty"` // 分类名称（服务端注入，非 DB 字段）
	MenuID        *uint   `gorm:"index" json:"menuId,omitempty"`   // 所属导航菜单ID（文章/资源选择挂载的菜单）
	AuthorID      uint    `json:"authorId"`
	AuthorName    string  `gorm:"size:50" json:"authorName"`
	IsPremium     bool    `gorm:"default:false" json:"isPremium"`
	MembersOnly   bool    `gorm:"default:false" json:"membersOnly"`     // 仅会员可访问/下载（下载/视频/漫画/小说模块）
	Price         float64 `gorm:"default:0" json:"price"`               // 基础价格（免费用户/非会员）
	OriginalPrice float64 `gorm:"default:0" json:"originalPrice"`       // 原价
	VipPrices     string  `gorm:"type:text" json:"vipPrices,omitempty"` // JSON: {"regular":9.9,"premium":4.9,"vip":0}
	// DownloadTitle 下载模块标题（覆盖默认「付费资源/免费资源」）。空 = 用前端自动判定。
	// 用于后台编辑文章时手动给下载模块命名（如「漫画全集」「游戏资源」），为空则前端按附件价格自动显示「付费资源/免费资源」。
	DownloadTitle string     `gorm:"size:100" json:"downloadTitle,omitempty"`
	Status        string     `gorm:"size:20;default:'published'" json:"status"`    // published / pending / draft
	ContentType   string     `gorm:"size:30;default:'article'" json:"contentType"` // article / video / comic / novel / download
	PayModule     string     `gorm:"size:30;default:''" json:"payModule"`          // 付费模块类型: 2=资源下载 3=视频 4=漫画 5=小说
	Attrs         string     `gorm:"size:200" json:"attrs"`                        // pc_app / mobile_app
	GamePlatform  string     `gorm:"size:20;default:''" json:"gamePlatform"`       // pc / mobile / both（仅资源下载类型）
	GameGenre     string     `gorm:"size:100;default:''" json:"gameGenre"`         // SLG/RPG/ADV/ACT/HTML 逗号分隔
	Views         int        `gorm:"default:0" json:"views"`
	LikeCount     int        `gorm:"default:0" json:"likeCount"` // 短剧/文章点赞数
	EpisodeCount  int        `gorm:"-" json:"episodeCount"`      // 视频集数（非 DB 字段，列表接口批量注入）
	VideoURL      string     `gorm:"size:500" json:"videoUrl,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	ScheduledAt   *time.Time `gorm:"index" json:"scheduledAt,omitempty"`                         // 定时发布时间（nil=立即发布）
	PublishedAt   *time.Time `gorm:"index" json:"publishedAt,omitempty"`                         // 实际发布时间：状态翻转为 published 时写入（用于「发布日期」展示）
	ReleaseStatus string     `gorm:"size:20;default:'published'" json:"releaseStatus,omitempty"` // 定时到期后的目标状态：published / draft / pending

	// ===== 本次新增：类型元数据（articles-editor 改造，2026-10-05）=====
	// 全部走 AutoMigrate 自动 ADD COLUMN，无需迁移文件。命名避坑见下。

	// ---- 通用 / 新闻 article ----
	Subtitle     string `gorm:"size:200;default:''" json:"subtitle"`          // 副标题/导语
	SourceType   string `gorm:"size:20;default:'original'" json:"sourceType"` // original / reprint / compile
	SourceName   string `gorm:"size:100;default:''" json:"sourceName"`        // 来源/转载出处
	SourceURL    string `gorm:"size:500;default:''" json:"sourceUrl"`         // 原文链接
	IsFeatured   bool   `gorm:"default:false" json:"isFeatured"`              // 头条/置顶
	AllowComment bool   `gorm:"default:true" json:"allowComment"`             // 允许评论 ★default:true 会在 Create 时吞掉前端的 false，见 handlers/article.go CreateArticle 的补偿逻辑

	// ---- 视频 / 漫画 / 小说 共用 ----
	SerialStatus string `gorm:"size:20;default:''" json:"serialStatus"` // ongoing / completed / paused
	Region       string `gorm:"size:30;default:''" json:"region"`       // 地区

	// ---- 下载 / 视频 共用 ----
	Language string `gorm:"size:30;default:''" json:"language"` // zh-CN / zh-TW / en / ja / ko / multi / other

	// ---- 下载 download ----
	ResourceVersion     string     `gorm:"size:50;default:''" json:"resourceVersion"`
	Requirement         string     `gorm:"size:1000;default:''" json:"requirement"`
	ResourceSizeDisplay string     `gorm:"size:50;default:''" json:"resourceSizeDisplay"`
	Publisher           string     `gorm:"size:100;default:''" json:"publisher"`
	IsOfficial          bool       `gorm:"default:false" json:"isOfficial"`
	ResourceUpdatedAt   *time.Time `json:"resourceUpdatedAt,omitempty"`

	// ---- 视频 video ----
	DurationMinutes int        `gorm:"default:0" json:"durationMinutes"` // ★分钟，区别于 VideoSource.Duration（秒）
	EpisodeTotal    int        `gorm:"default:0" json:"episodeTotal"`    // ★真实列，区别于 EpisodeCount(gorm:"-")
	ReleaseYear     int        `gorm:"default:0" json:"releaseYear"`
	ReleaseDate     *time.Time `json:"releaseDate,omitempty"` // P1
	Cast            string     `gorm:"size:500;default:''" json:"cast"`
	Director        string     `gorm:"size:100;default:''" json:"director"`
	QualityOptions  string     `gorm:"size:100;default:''" json:"qualityOptions"` // ★多选项，区别于 VideoSource.Quality（单条）

	// ---- 漫画 comic ----
	Illustrator       string `gorm:"size:100;default:''" json:"illustrator"`
	ChapterTotal      int    `gorm:"default:0" json:"chapterTotal"`
	TranslationStatus string `gorm:"size:20;default:''" json:"translationStatus"` // official_zh / fansub / raw / unknown
	FirstPlatform     string `gorm:"size:100;default:''" json:"firstPlatform"`

	// ---- 小说 novel ----
	WordCount int `gorm:"default:0" json:"wordCount"`

	// ---- 通用：正文格式（存量兼容）----
	ContentFormat string `gorm:"size:10;default:''" json:"contentFormat"` // '' / html / markdown

	// ExternalRef 采集去重引用：存储外部来源标识（如 "bfzy:<vod_id>"），空 = 非采集导入。
	// 采集器用它在 upsert 时定位已导入的文章，避免重复写入。
	ExternalRef string         `gorm:"size:120;index" json:"externalRef,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Chapter 章节（漫画/小说）
type Chapter struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ContentID    uint           `gorm:"index;not null" json:"contentId"`
	Title        string         `gorm:"size:200" json:"title"`
	Content      string         `gorm:"type:longtext" json:"content"`
	ChapterOrder int            `gorm:"not null" json:"chapterOrder"`
	IsFree       bool           `gorm:"default:false" json:"isFree"`
	Price        float64        `gorm:"default:0" json:"price"`
	Images       string         `gorm:"type:text" json:"images,omitempty"` // JSON array for comics
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Order 订单模型
type Order struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	OrderNo         string         `gorm:"uniqueIndex;size:50;not null" json:"orderNo"`
	UserID          uint           `gorm:"index;not null" json:"userId"`
	Type            string         `gorm:"size:30;not null" json:"type"` // membership / attachment / chapter / content / membership_v2
	TargetLevel     string         `gorm:"size:20" json:"targetLevel,omitempty"`
	TargetGroup     string         `gorm:"size:20" json:"targetGroup,omitempty"` // 新体系会员组：vip / svip
	SubType         string         `gorm:"size:20" json:"subType,omitempty"`     // 新体系订阅类型：monthly / quarterly / annual
	ArticleID       *uint          `json:"articleId,omitempty"`
	ContentID       *uint          `json:"contentId,omitempty"`
	ChapterID       *uint          `json:"chapterId,omitempty"`
	AttachmentID    *uint          `json:"attachmentId,omitempty"`
	PlatformOrderNo string         `gorm:"size:64" json:"platformOrderNo,omitempty"` // 第三方平台订单号（KikiPay 等）
	Amount          float64        `gorm:"not null" json:"amount"`
	Status          string         `gorm:"size:20;default:'pending'" json:"status"` // pending / paid / cancelled
	CouponCode      string         `gorm:"size:50" json:"couponCode,omitempty"`
	Source          string         `gorm:"size:10;default:'web'" json:"source"` // 下单来源：web（网站）/ client（桌面客户端）/ app（安卓 App），默认 web
	PayMethod       string         `gorm:"size:20;default:''" json:"payMethod"` // 支付方式：balance / alipay / wxpay
	PayUrl          string         `gorm:"size:512" json:"payUrl,omitempty"`    // 缓存 KikiPay 返回的收银台 URL（同单号同通道重试时直接返回，避免「商户单号重复」）
	ExpiryDate      *time.Time     `json:"expiryDate,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// Purchase 购买记录
type Purchase struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	Type      string    `gorm:"size:30;not null" json:"type"` // attachment / chapter / content
	ItemID    uint      `gorm:"not null" json:"itemId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Coupon 优惠券
type Coupon struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Code       string     `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Type       string     `gorm:"size:20;not null" json:"type"` // fixed / percent
	Value      float64    `gorm:"not null" json:"value"`
	MinAmount  float64    `gorm:"default:0" json:"minAmount"`
	MaxUses    int        `gorm:"default:0" json:"maxUses"` // 0 = unlimited
	UsedCount  int        `gorm:"default:0" json:"usedCount"`
	ValidFrom  *time.Time `json:"validFrom,omitempty"`
	ValidUntil *time.Time `json:"validUntil,omitempty"`
	Status     string     `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Message 站内消息
type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"userId"`
	Title     string         `gorm:"size:200" json:"title"`
	Content   string         `gorm:"type:text" json:"content"`
	Type      string         `gorm:"size:30;default:'system'" json:"type"` // system / order / content
	IsRead    bool           `gorm:"default:false" json:"isRead"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Badge 徽章
type Badge struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	Description string    `gorm:"size:200" json:"description"`
	Icon        string    `gorm:"size:50" json:"icon"`
	Condition   string    `gorm:"size:200" json:"condition"`
	CreatedAt   time.Time `json:"createdAt"`
}

// UserBadge 用户徽章关联
type UserBadge struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	UserID   uint      `gorm:"index;not null" json:"userId"`
	BadgeID  uint      `gorm:"index;not null" json:"badgeId"`
	EarnedAt time.Time `json:"earnedAt"`
}

// CheckIn 签到记录
type CheckIn struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"userId"`
	CheckDate    string    `gorm:"size:10;not null" json:"checkDate"` // YYYY-MM-DD
	PointsEarned int       `gorm:"default:0" json:"pointsEarned"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Setting 系统配置
type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Category  string    `gorm:"size:50;default:'general'" json:"category"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// EmailVerification 邮箱验证码
type EmailVerification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"index;size:100" json:"email"`
	Code      string    `gorm:"size:10" json:"code"`
	Used      bool      `gorm:"default:false" json:"used"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// ScanLoginToken 扫码登录令牌（网页 / Windows 桌面端扫码登录用）
//
// 流程：网页端 POST /auth/qr/create 拿到 token → 展示二维码（内容为该 token 的登录链接）
// → App 扫码后 POST /auth/qr/:token/scan 标记已扫 → App 用户确认
// POST /auth/qr/:token/confirm 签发 JWT，网页端轮询 /auth/qr/:token/status 拿到 token 完成登录。
type ScanLoginToken struct {
	Token       string    `gorm:"primaryKey;size:64" json:"token"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"` // pending / scanned / confirmed / expired
	Device      string    `gorm:"size:20" json:"device"`                   // web / windows
	ScannedBy   uint      `json:"scannedBy"`                               // 扫码确认的用户 ID
	ConfirmedBy uint      `json:"confirmedBy"`                             // 最终登录的用户 ID
	JWT         string    `gorm:"size:512" json:"-"`                       // 确认后签发的 JWT
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// NavigationMenu 自定义导航菜单
type NavigationMenu struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:50;not null" json:"name"`                   // 显示名称（如"全部"、"新闻"，可覆盖分类名）
	ContentType string `gorm:"size:30;default:''" json:"contentType"`          // 对应的文章类型（article/download/video/comic/novel，空表示全部）
	CategoryID  *uint  `gorm:"index" json:"categoryId,omitempty"`              // 关联的分类目录ID
	Path        string `gorm:"size:200;default:''" json:"path"`                // 自定义路径（优先级高于contentType）
	Icon        string `gorm:"size:50;default:''" json:"icon"`                 // emoji 或 Font Awesome 图标类名（如 fa-solid fa-house）
	ThumbStyle  string `gorm:"size:20;default:'horizontal'" json:"thumbStyle"` // 缩略图风格: horizontal=横板 / vertical=竖板 / list=列表(左图右文加摘要)
	Position    string `gorm:"size:10;default:'top'" json:"position"`          // top / left / mobile（顶部导航 / 左侧菜单 / 手机端菜单）
	SortOrder   int    `gorm:"default:0" json:"sortOrder"`                     // 排序
	// ParentID 两级菜单的父级（仅自定义菜单用；NULL=一级）。分类项无父子概念，恒为 NULL。
	ParentID  *uint     `gorm:"index" json:"parentId,omitempty"`
	Status    string    `gorm:"size:20;default:'active'" json:"status"` // active/inactive
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Install 安装记录
type Install struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Installed   bool      `gorm:"default:false" json:"installed"`
	Version     string    `gorm:"size:20" json:"version"`
	AdminName   string    `gorm:"size:50" json:"adminName"`
	InstalledAt time.Time `json:"installedAt"`
}

// ==================== Zibll 社区论坛模块 ====================

// BbsForum 论坛版块
type BbsForum struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Icon        string         `gorm:"size:50" json:"icon"`
	Sort        int            `gorm:"default:0" json:"sort"`
	TopicCount  int            `gorm:"default:0" json:"topicCount"`
	Status      string         `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BbsTopic 话题/帖子
type BbsTopic struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ForumID     uint           `gorm:"index;not null" json:"forumId"`
	UserID      uint           `gorm:"index;not null" json:"userId"`
	Username    string         `gorm:"size:50" json:"username"`
	Avatar      string         `gorm:"size:500" json:"avatar"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Content     string         `gorm:"type:longtext" json:"content"`
	Images      string         `gorm:"type:text" json:"images"`              // JSON
	Type        string         `gorm:"size:20;default:'normal'" json:"type"` // normal / question / share
	IsEssence   bool           `gorm:"default:false" json:"isEssence"`
	IsTop       bool           `gorm:"default:false" json:"isTop"`
	IsSolved    bool           `gorm:"default:false" json:"isSolved"`
	ViewCount   int            `gorm:"default:0" json:"viewCount"`
	ReplyCount  int            `gorm:"default:0" json:"replyCount"`
	LikeCount   int            `gorm:"default:0" json:"likeCount"`
	Tags        string         `gorm:"size:200" json:"tags"`
	Status      string         `gorm:"size:20;default:'published'" json:"status"`
	LastReplyAt *time.Time     `json:"lastReplyAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BbsReply 回复/评论
type BbsReply struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TopicID   uint           `gorm:"index;not null" json:"topicId"`
	UserID    uint           `gorm:"index;not null" json:"userId"`
	Username  string         `gorm:"size:50" json:"username"`
	Avatar    string         `gorm:"size:500" json:"avatar"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	ParentID  *uint          `json:"parentId,omitempty"` // 回复的回复
	LikeCount int            `gorm:"default:0" json:"likeCount"`
	Status    string         `gorm:"size:20;default:'published'" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BbsLike 论坛点赞记录
type BbsLike struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	TargetID   uint      `gorm:"index;not null" json:"targetId"`
	TargetType string    `gorm:"size:20;not null" json:"targetType"` // topic / reply
	CreatedAt  time.Time `json:"createdAt"`
}

// ArticleLike 文章点赞（短剧/文章浮窗用，独立于 BBS 话题/回复的 BbsLike）
type ArticleLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	ArticleID uint      `gorm:"index;not null" json:"articleId"`
	CreatedAt time.Time `json:"createdAt"`
}

// ==================== 收藏模块 ====================

// Favorite 收藏
type Favorite struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	TargetID   uint      `gorm:"index;not null" json:"targetId"`
	TargetType string    `gorm:"size:20;not null" json:"targetType"` // article / topic / content
	Title      string    `gorm:"size:200" json:"title"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ==================== 私信模块 ====================

// PrivateMessage 用户间私信
type PrivateMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	FromUserID     uint      `gorm:"index;not null" json:"fromUserId"`
	FromName       string    `gorm:"size:50" json:"fromName"`
	FromAvatar     string    `gorm:"size:500" json:"fromAvatar"`
	ToUserID       uint      `gorm:"index;not null" json:"toUserId"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	Image          string    `gorm:"size:500;default:''" json:"image"` // 客服聊天图片（R2/CDN URL）
	IsRead         bool      `gorm:"default:false" json:"isRead"`
	MsgType        string    `gorm:"size:20;default:'text'" json:"msgType"`          // text / greeting / article（客服自动回复卡片类型）
	Payload        string    `gorm:"type:text;default:''" json:"payload"`            // 结构化附加数据：article 卡片时为 {"article":{...}}
	IsAuto         bool      `gorm:"default:false" json:"isAuto"`                    // 是否由客服自动回复引擎生成
	ConversationID uint      `gorm:"index;not null;default:0" json:"conversationId"` // 所属「通话」分组；0 表示历史未分组消息
	CreatedAt      time.Time `json:"createdAt"`
}

// ChatConversation 用户与官方客服的一次「通话」（可主动结束；结束后归档为历史）。
// 用户在客服聊天中「结束本次通话」即把当前 active 会话置为 ended；再次进入若无 active 会话则新建并种入开场语。
// 历史未分组消息（conversation_id=0）在首次打开时会被归入一个新建的会话，保证存量数据可用。
type ChatConversation struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"userId"`
	AdminID    uint       `gorm:"index;not null" json:"adminId"`              // 兜底/默认接待坐席（首个 role=admin 用户）
	AssigneeID uint       `gorm:"index;not null;default:0" json:"assigneeId"` // 当前接待坐席（接管/转接后变更；0 表示沿用 AdminID）
	Status     string     `gorm:"size:20;default:'active'" json:"status"`     // active / pending / ended / closed
	QueueAt    *time.Time `json:"queueAt,omitempty"`                          // 进入待接入队列时间（首响时长统计）
	AssignAt   *time.Time `json:"assignAt,omitempty"`                         // 首次分配/接入时间
	TakeoverBy uint       `gorm:"default:0" json:"takeoverBy"`                // 最近一次接管者（Agent.UserID）
	TakeoverAt *time.Time `json:"takeoverAt,omitempty"`                       // 最近一次接管时间
	Priority   int        `gorm:"default:0" json:"priority"`                  // 队列优先级
	Preview    string     `gorm:"size:200;default:''" json:"preview"`         // 最近一条消息预览
	CreatedAt  time.Time  `json:"createdAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
}

// ChatAutoReply 客服自动回复规则（模糊/精确关键词 + 文章卡片匹配）。
// 后台可热改（增删改），无需重部署。
type ChatAutoReply struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`           // 规则名称（后台便于辨识）
	MatchType    string    `gorm:"size:20;not null" json:"matchType"`       // exact / contains / fuzzy / article_intent
	Keyword      string    `gorm:"size:200;not null" json:"keyword"`        // 触发关键词；article_intent 时为意图词「|」分隔
	ReplyType    string    `gorm:"size:20;default:'text'" json:"replyType"` // text / article
	ReplyContent string    `gorm:"type:text" json:"replyContent"`           // 文本回复内容 或 文章卡片引导语
	Priority     int       `gorm:"default:0" json:"priority"`               // 越大越优先
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ==================== 多端统一配置（替代原三套 JSON Blob） ====================

// ClientParam 按客户端类型一行的参数配置（导航/菜单/更新/红包/PCDN 等）。
type ClientParam struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ClientType string    `gorm:"uniqueIndex;size:30;not null" json:"clientType"` // web / desktop / android
	Config     string    `gorm:"type:text" json:"config"`                        // 全部参数 JSON
	Version    int       `gorm:"default:1" json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// FeatureFlag 按客户端类型 + key 一行的功能开关（可灰度）。
type FeatureFlag struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PBID        string    `gorm:"size:32;index" json:"pbId,omitempty"` // PocketBase 配置域拉取的记录 id（PB 管理的行非空）
	ClientType  string    `gorm:"index;size:30;not null" json:"clientType"`
	Key         string    `gorm:"index;size:50;not null" json:"key"`
	Enabled     bool      `gorm:"default:false" json:"enabled"`
	Description string    `gorm:"size:200" json:"description"`
	Rollout     int       `gorm:"default:100" json:"rollout"` // 灰度 0-100
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ClientStrategy 强制更新/红包时间窗/PCDN 等策略。
type ClientStrategy struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	ClientType string     `gorm:"index;size:30;not null" json:"clientType"`
	Type       string     `gorm:"index;size:50;not null" json:"type"` // force_update / red_packet / pcdn / github_update ...
	Name       string     `gorm:"size:100" json:"name"`
	Enabled    bool       `gorm:"default:false" json:"enabled"`
	Payload    string     `gorm:"type:text" json:"payload"`
	Priority   int        `gorm:"default:0" json:"priority"`
	StartAt    *time.Time `json:"startAt,omitempty"`
	EndAt      *time.Time `json:"endAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// Agent 客服坐席（关联 User）。
type Agent struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex;not null" json:"userId"`
	Nickname       string    `gorm:"size:50" json:"nickname"`
	OnlineStatus   string    `gorm:"size:20;default:'offline'" json:"onlineStatus"` // offline / online / busy
	MaxConcurrency int       `gorm:"default:5" json:"maxConcurrency"`
	CurrentLoad    int       `gorm:"default:0" json:"currentLoad"`
	Skills         string    `gorm:"type:text" json:"skills"` // JSON 技能标签数组
	AutoAccept     bool      `gorm:"default:true" json:"autoAccept"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ChatRoutingRule 客服消息路由规则。
type ChatRoutingRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100" json:"name"`
	MatchType   string    `gorm:"size:20" json:"matchType"` // all / keyword / skill
	MatchValue  string    `gorm:"size:200" json:"matchValue"`
	Strategy    string    `gorm:"size:20" json:"strategy"` // round_robin / least_loaded / assign_agent
	TargetAgent uint      `gorm:"default:0" json:"targetAgent"`
	Priority    int       `gorm:"default:0" json:"priority"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ==================== VIP 卡密模块 ====================

// VipCard VIP卡密
type VipCard struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Code       string     `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Level      string     `gorm:"size:20;not null" json:"level"` // regular / premium / vip
	Days       int        `gorm:"not null" json:"days"`          // 天数
	RedeemedBy *uint      `json:"redeemedBy,omitempty"`
	RedeemedAt *time.Time `json:"redeemedAt,omitempty"`
	Batch      string     `gorm:"size:50" json:"batch"`                   // 批次
	Status     string     `gorm:"size:20;default:'active'" json:"status"` // active / used / disabled
	CreatedAt  time.Time  `json:"createdAt"`
}

// ==================== 积分商城模块 ====================

// PointsProduct 积分商品
type PointsProduct struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Cover       string         `gorm:"size:500" json:"cover"`
	PointsPrice int            `gorm:"not null" json:"pointsPrice"`  // 积分数
	Type        string         `gorm:"size:30;not null" json:"type"` // virtual / physical / membership
	Stock       int            `gorm:"default:-1" json:"stock"`      // -1 无限
	SoldCount   int            `gorm:"default:0" json:"soldCount"`
	RewardData  string         `gorm:"type:text" json:"rewardData"` // JSON: 兑换后的奖励数据
	Status      string         `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// PointsOrder 积分兑换订单
type PointsOrder struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderNo     string    `gorm:"uniqueIndex;size:50;not null" json:"orderNo"`
	UserID      uint      `gorm:"index;not null" json:"userId"`
	ProductID   uint      `gorm:"index;not null" json:"productId"`
	ProductName string    `gorm:"size:200" json:"productName"`
	PointsCost  int       `gorm:"not null" json:"pointsCost"`
	Status      string    `gorm:"size:20;default:'completed'" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ==================== 邀请推广模块 ====================

// InviteCode 邀请码
type InviteCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"uniqueIndex;size:20;not null" json:"code"`
	OwnerID   uint      `gorm:"index;not null" json:"ownerId"`
	UsedCount int       `gorm:"default:0" json:"usedCount"`
	MaxUses   int       `gorm:"default:0" json:"maxUses"` // 0 = 无限
	Status    string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// InviteRecord 推广记录
type InviteRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	InviterID  uint      `gorm:"index;not null" json:"inviterId"`
	InviteeID  uint      `gorm:"index;not null" json:"inviteeId"`
	Code       string    `gorm:"size:20" json:"code"`
	Reward     int       `gorm:"default:0" json:"reward"`                     // 奖励数量
	RewardType string    `gorm:"size:20;default:'balance'" json:"rewardType"` // balance=余额 / points=积分
	Status     string    `gorm:"size:20;default:'granted'" json:"status"`     // pending=待设备验证 / granted=已发放 / invalid=设备已注册取消
	DeviceID   string    `gorm:"size:200" json:"deviceId"`                    // 被邀请人注册所用设备
	CreatedAt  time.Time `json:"createdAt"`
}

// ImageCacheClaim 客户端图库认领记录（用于跨用户去重：同一篇文章尽量只被一个客户端缓存）
type ImageCacheClaim struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ArticleID uint      `gorm:"uniqueIndex:uniq_article_device;index;not null" json:"articleId"`
	DeviceID  string    `gorm:"uniqueIndex:uniq_article_device;size:64;not null" json:"deviceId"`
	CreatedAt time.Time `json:"createdAt"`
}

// ImagePeer P2P 图片持有索引：记录某个图块(hash)被哪些客户端设备持有
// 服务器只存索引，不存图片字节；客户端之间经服务器中转互相传图
type ImagePeer struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Hash         string    `gorm:"uniqueIndex:uniq_hash_device;size:64;not null" json:"hash"`
	DeviceID     string    `gorm:"uniqueIndex:uniq_hash_device;size:64;not null" json:"deviceId"`
	UserID       uint      `gorm:"index;not null" json:"userId"`
	ArticleID    uint      `gorm:"index;not null" json:"articleId"`
	ImageURL     string    `gorm:"size:500;not null" json:"imageUrl"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// ==================== P2P 资源（百度直链 / 文件分块） ====================

// ResourcePeer P2P 文件分块持有索引：记录某个资源(resourceKey)的某分块(chunkIndex)
// 被哪些设备持有。服务器只存索引，字节经服务器中转(store-and-forward)。
type ResourcePeer struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ResourceKey  string    `gorm:"uniqueIndex:uniq_res_dev;size:128;not null" json:"resourceKey"`
	DeviceID     string    `gorm:"uniqueIndex:uniq_res_dev;size:128;not null" json:"deviceId"`
	UserID       uint      `gorm:"index;not null" json:"userId"`
	ChunkIndex   int       `gorm:"not null" json:"chunkIndex"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// BaiduResource 资源元数据：把分享链接+fsId 映射到一个稳定 resourceKey，
// 后端据此解析百度直链并分块缓存。避免每次都重新访问分享页。
type BaiduResource struct {
	ResourceKey string    `gorm:"primaryKey;size:128" json:"resourceKey"`
	ShareURL    string    `gorm:"type:text;not null" json:"shareUrl"`
	FsID        int64     `gorm:"index" json:"fsId"`
	Pwd         string    `gorm:"size:32" json:"pwd"`
	FileName    string    `gorm:"size:255" json:"fileName"`
	FileSize    int64     `json:"fileSize"`
	ChunkSize   int64     `json:"chunkSize"`
	ExpireAt    time.Time `gorm:"index" json:"expireAt"` // 直链缓存过期后资源元数据也需刷新
	CreatedAt   time.Time `json:"createdAt"`
}

// BaiduDlinkCache 百度直链缓存：fsId → dlink（带 TTL），省去每次重新解析/转存，
// 节省 SVIP 账号配额。dlink 通常几小时内有效，这里 TTL 取 2h。
type BaiduDlinkCache struct {
	FsID     int64     `gorm:"primaryKey" json:"fsId"`
	Dlink    string    `gorm:"type:text;not null" json:"dlink"`
	FileName string    `gorm:"size:255" json:"fileName"`
	FileSize int64     `json:"fileSize"`
	ExpireAt time.Time `gorm:"index" json:"expireAt"`
}

// BaiduMigration 百度网盘迁移记录：把文章附件的原分享文件转存到系统账号后，
// 记录系统账号内的文件 fs_id 与路径。下载时优先走系统账号直链（dlink），
// 从而在原分享被上传者删除/撤销后仍能下载。
type BaiduMigration struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AttachmentID uint      `gorm:"index" json:"attachmentId"`
	ArticleID    uint      `gorm:"index" json:"articleId"`
	PanPath      string    `gorm:"size:255" json:"panPath"`    // 系统账号内路径，如 /gitmoe_files/9981
	FileFsIDs    string    `gorm:"type:text" json:"fileFsIds"` // JSON 数组：系统账号内文件 fs_id 列表
	FileNames    string    `gorm:"type:text" json:"fileNames"` // JSON 数组：对应文件名
	FileCount    int       `json:"fileCount"`
	Status       string    `gorm:"size:32" json:"status"` // pending|transferred|verified|failed
	DlinkOK      bool      `json:"dlinkOk"`
	Err          string    `gorm:"type:text" json:"err"`
	MigratedAt   time.Time `json:"migratedAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ==================== 任务模块 ====================

// DailyTask 每日任务定义
type DailyTask struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PBID         string    `gorm:"size:32;index" json:"pbId,omitempty"` // PocketBase 配置域拉取的记录 id（PB 管理的行非空）
	Name         string    `gorm:"size:100;not null" json:"name"`
	Description  string    `gorm:"size:300" json:"description"`
	Icon         string    `gorm:"size:50" json:"icon"`
	Action       string    `gorm:"size:50;not null" json:"action"` // checkin / read_article / post_topic / reply / share
	PointsReward int       `gorm:"not null" json:"pointsReward"`
	MaxDaily     int       `gorm:"default:1" json:"maxDaily"` // 每日最大完成次数
	Sort         int       `gorm:"default:0" json:"sort"`
	Status       string    `gorm:"size:20;default:'active'" json:"status"`
	Once         bool      `gorm:"default:false" json:"once"`       // 全站仅一次（如首次改头像），不按日重置
	ServerOnly   bool      `gorm:"default:false" json:"serverOnly"` // 仅服务端发放（如充值到账），客户端不可手动领取
	CreatedAt    time.Time `json:"createdAt"`
}

// Barrage 用户发送的弹幕（区别于 paym ent 弹幕墙），发送即发放积分。
type Barrage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	Username  string    `gorm:"size:50" json:"username"`
	Avatar    string    `gorm:"size:500" json:"avatar"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Color     string    `gorm:"size:20;default:''" json:"color"`  // 弹幕颜色（前端传）
	ArticleID *uint     `gorm:"index" json:"articleId,omitempty"` // 关联文章（可选）
	Status    string    `gorm:"size:20;default:'published'" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserTask 用户任务完成记录
type UserTask struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	TaskID     uint      `gorm:"index;not null" json:"taskId"`
	CompleteAt time.Time `json:"completeAt"`
	Date       string    `gorm:"size:10" json:"date"` // YYYY-MM-DD
}

// ==================== 评论模块（文章评论） ====================

// Comment 文章评论
type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ArticleID uint           `gorm:"index;not null" json:"articleId"`
	UserID    uint           `gorm:"index;not null" json:"userId"`
	Username  string         `gorm:"size:50" json:"username"`
	Avatar    string         `gorm:"size:500" json:"avatar"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Image     string         `gorm:"size:500;default:''" json:"image"` // 评论附图（R2/CDN URL）
	ParentID  *uint          `json:"parentId,omitempty"`
	LikeCount int            `gorm:"default:0" json:"likeCount"`
	Status    string         `gorm:"size:20;default:'published'" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ==================== 单页（独立页面）模块 ====================

// Page 单页文档（如"关于我们""用户协议"），供后台 CRUD、前台按 slug 渲染。
type Page struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:200;not null" json:"title"`
	Slug      string         `gorm:"size:100;uniqueIndex" json:"slug"`
	Content   string         `gorm:"type:text" json:"content"`
	Status    string         `gorm:"size:20;default:'draft'" json:"status"` // draft/published
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ==================== 打赏模块 ====================

// Reward 打赏记录
type Reward struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	FromUserID uint      `gorm:"index;not null" json:"fromUserId"`
	FromName   string    `gorm:"size:50" json:"fromName"`
	ToUserID   uint      `gorm:"index;not null" json:"toUserId"`
	TargetID   uint      `json:"targetId"`                  // 文章/话题ID
	TargetType string    `gorm:"size:20" json:"targetType"` // article / topic
	Amount     float64   `gorm:"not null" json:"amount"`
	Message    string    `gorm:"size:200" json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ==================== 文章付费下载模块 ====================

// ArticleAttachment 文章付费附件
type ArticleAttachment struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ArticleID uint   `gorm:"index;not null" json:"articleId"`
	Title     string `gorm:"size:200" json:"title"` // nullable: 允许为空（后台不填则前端展示「付费资源/免费资源」）
	// 链接类型: file=本地上传, netdisk=网盘链接, external=外部直链
	LinkType string `gorm:"size:20;default:'file'" json:"linkType"`
	// 本地文件字段
	FileURL  string `gorm:"size:500" json:"fileUrl"`
	FileName string `gorm:"size:200" json:"fileName"`
	FileSize int64  `gorm:"default:0" json:"fileSize"`
	FileType string `gorm:"size:50" json:"fileType"` // zip/pdf/rar/doc/other
	// 网盘链接字段
	LinkURL      string `gorm:"size:1000" json:"linkUrl,omitempty"`    // 网盘/外部链接地址
	LinkCode     string `gorm:"size:50" json:"linkCode,omitempty"`     // 提取码
	LinkUnzip    string `gorm:"size:50" json:"linkUnzip,omitempty"`    // 解压码
	LinkPlatform string `gorm:"size:30" json:"linkPlatform,omitempty"` // baidu/quark/ali/115/other
	// 付费设置
	Price         float64        `gorm:"default:0" json:"price"`                 // 单独售价, 0=免费
	OriginalPrice float64        `gorm:"default:0" json:"originalPrice"`         // 原价
	IsMemberFree  bool           `gorm:"default:false" json:"isMemberFree"`      // 会员免费下载
	LevelPrices   string         `gorm:"type:text" json:"levelPrices,omitempty"` // JSON: {"month":0,"year":9.9}
	FreeLevels    string         `gorm:"size:500" json:"freeLevels,omitempty"`   // 逗号分隔的免费等级key
	DownloadCount int            `gorm:"default:0" json:"downloadCount"`
	Sort          int            `gorm:"default:0" json:"sort"`
	Status        string         `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// AttachmentDownload 附件下载记录
// AttachmentID 为 NULL 时表示「按文章价购买」的文章级解锁记录，作用于同 article 下所有 att。
type AttachmentDownload struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	UserID       uint    `gorm:"index;not null" json:"userId"`
	AttachmentID *uint   `gorm:"index" json:"attachmentId"` // nullable: NULL = 文章级解锁（按 article.price 购）
	ArticleID    uint    `gorm:"index" json:"articleId"`
	Amount       float64 `gorm:"default:0" json:"amount"`           // 付费金额, 0=会员免费
	IsFreeQuota  bool    `gorm:"default:false" json:"isFreeQuota"`  // 是否计入会员每日免费次数
	IsPointsPaid bool    `gorm:"default:false" json:"isPointsPaid"` // 是否用积分抵扣下载（替代免费额度/余额）
	// ExpiresAt 下载/解锁有效期（跨端统一：网站 / 客户端 / App 共用同一口径）
	//   - 免费额度（is_free_quota=1）：当天 24:00 失效；
	//     且用户消耗下一次免费额度时，上一次未满 24 小时的免费记录立即失效；
	//   - 付费购买（amount>0）：保留 7 天；
	//   - 占位记录（amount=0 且 is_free_quota=0）：跟随其所属文章级解锁记录的到期时间。
	ExpiresAt *time.Time `gorm:"index" json:"expiresAt,omitempty"`
	ViewedAt  *time.Time `gorm:"index" json:"viewedAt,omitempty"` // 查看链接时间
	CreatedAt time.Time  `json:"createdAt"`
}

// ==================== 视频模块 ====================

// VideoSource 视频源（支持m3u8/mp4/x.com等，按分集组织）
type VideoSource struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ArticleID   uint           `gorm:"index;not null" json:"articleId"`
	Episode     int            `gorm:"default:1" json:"episode"` // 第几集
	Title       string         `gorm:"size:200" json:"title"`
	SourceType  string         `gorm:"size:30;not null" json:"sourceType"`   // m3u8 / mp4 / x_com / youtube / bilibili
	SourceURL   string         `gorm:"size:1000;not null" json:"sourceUrl"`  // 原始地址
	EncryptKey  string         `gorm:"size:128" json:"encryptKey,omitempty"` // HLS加密密钥（16进制AES-128）
	EncryptIV   string         `gorm:"size:64" json:"encryptIv,omitempty"`   // 加密IV
	Encrypted   bool           `gorm:"default:false" json:"encrypted"`       // 是否已加密
	Duration    int            `gorm:"default:0" json:"duration"`            // 时长(秒)
	Thumbnail   string         `gorm:"size:500" json:"thumbnail"`
	Quality     string         `gorm:"size:20;default:'hd'" json:"quality"` // sd/hd/fhd/4k
	FileSize    int64          `gorm:"default:0" json:"fileSize"`
	CachedPath  string         `gorm:"size:500" json:"cachedPath,omitempty"`            // 本地缓存路径
	CacheStatus string         `gorm:"size:20;default:''" json:"cacheStatus,omitempty"` // '' / caching / cached / failed
	Sort        int            `gorm:"default:0" json:"sort"`
	Status      string         `gorm:"size:20;default:'active'" json:"status"`
	PlayUrl     string         `gorm:"-" json:"playUrl,omitempty"` // 经解析/播放接口包裹后的地址（不入库，读取时计算）
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VideoWatchRecord 视频观看记录
type VideoWatchRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	ArticleID uint      `gorm:"index;not null" json:"articleId"`
	SourceID  uint      `gorm:"index" json:"sourceId"`
	Progress  int       `gorm:"default:0" json:"progress"` // 观看进度(秒)
	WatchedAt time.Time `json:"watchedAt"`
}

// ==================== Banner 轮播管理 ====================

// Banner 首页轮播图
type Banner struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"size:200" json:"title"`
	ImageURL     string         `gorm:"size:500" json:"imageUrl"`
	VideoURL     string         `gorm:"size:500" json:"videoUrl,omitempty"` // PC端视频URL（手机端fallback图片）
	LinkURL      string         `gorm:"size:500" json:"linkUrl"`
	Width        int            `gorm:"default:0" json:"width"`        // 电脑端显示宽度(px)，0=自适应全宽
	Height       int            `gorm:"default:0" json:"height"`       // 电脑端显示高度(px)，0=默认高度
	MobileWidth  int            `gorm:"default:0" json:"mobileWidth"`  // 手机端显示宽度(px)，0=自适应全宽
	MobileHeight int            `gorm:"default:0" json:"mobileHeight"` // 手机端显示高度(px)，0=跟随电脑端
	Sort         int            `gorm:"default:0" json:"sort"`
	Status       string         `gorm:"size:20;default:'active'" json:"status"` // active / inactive
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// ArticleSEO 文章SEO元数据
type ArticleSEO struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ArticleID    uint      `gorm:"uniqueIndex;not null" json:"articleId"`
	MetaTitle    string    `gorm:"size:200" json:"metaTitle"`
	MetaDesc     string    `gorm:"size:500" json:"metaDesc"`
	MetaKeywords string    `gorm:"size:300" json:"metaKeywords"`
	Canonical    string    `gorm:"size:500" json:"canonical"`
	OgImage      string    `gorm:"size:500" json:"ogImage"`
	IsRewritten  bool      `gorm:"default:false" json:"isRewritten"`           // 是否已AI伪原创
	RewriteText  string    `gorm:"type:longtext" json:"rewriteText,omitempty"` // AI重写后的文本
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CrawlerLog 爬虫访问记录
type CrawlerLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CrawlerKey  string    `gorm:"index" json:"crawlerKey"`  // baidu / google / bing ...
	CrawlerName string    `gorm:"index" json:"crawlerName"` // 百度 / Google / 微软(Bing)
	Spider      string    `gorm:"index" json:"spider"`      // BaiduSpider / Googlebot ...
	URL         string    `gorm:"type:text" json:"url"`
	Status      int       `gorm:"index" json:"status"` // 200 / 404 ...
	Referer     string    `gorm:"type:text" json:"referer"`
	UserAgent   string    `gorm:"type:text" json:"userAgent"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
}

// ==================== 首页自定义模块 ====================

// HomeModule 首页自定义内容模块
type HomeModule struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PBID         string         `gorm:"size:32;index" json:"pbId,omitempty"` // PocketBase 配置域拉取的记录 id（PB 管理的行非空）
	Title        string         `gorm:"size:100;not null" json:"title"`
	Type         string         `gorm:"size:30;not null" json:"type"`               // video / article / comic / novel / game / custom
	CategoryID   *uint          `gorm:"index" json:"categoryId"`                    // 关联分类目录ID（首页模块按分类取内容，优于 Type）
	ContentIDs   string         `gorm:"type:text" json:"contentIds"`                // JSON array of article IDs
	DisplayStyle string         `gorm:"size:20;default:'grid'" json:"displayStyle"` // grid / carousel / list
	ItemCount    int            `gorm:"default:6" json:"itemCount"`
	CardsPerRow  int            `gorm:"default:0" json:"cardsPerRow"` // 每行几个（0=跟随分类/默认）
	Sort         int            `gorm:"default:0" json:"sort"`
	Status       string         `gorm:"size:20;default:'active'" json:"status"`  // active / inactive
	Device       string         `gorm:"size:10;default:'desktop'" json:"device"` // desktop / mobile
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// ==================== 漫画/视频付费配置 ====================

// ContentAccessRule 内容访问规则（漫画/视频的VIP分级访问）
type ContentAccessRule struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ContentType     string    `gorm:"size:30;not null" json:"contentType"`      // comic / video
	ArticleID       uint      `gorm:"index;not null" json:"articleId"`          // 关联的内容条目
	MembershipLevel string    `gorm:"size:20; not null" json:"membershipLevel"` // free / regular / premium / vip
	FreeChapters    int       `gorm:"default:0" json:"freeChapters"`            // 免费章节数, -1=无限制
	PricePerChapter float64   `gorm:"default:0" json:"pricePerChapter"`         // 单章价格（超出免费数后）
	IsUnlimited     bool      `gorm:"default:false" json:"isUnlimited"`         // 该等级是否无限免费
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ==================== 红包雨模块 ====================

// RedPacketRecord 红包雨领取记录（用于每日每人限领一次 + 查历史）
type RedPacketRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	Date       string    `gorm:"size:10;index" json:"date"` // YYYY-MM-DD（每日限领）
	RewardType string    `gorm:"size:20" json:"rewardType"` // balance / points
	Amount     float64   `gorm:"default:0" json:"amount"`   // 金额（balance 类型）
	Points     int       `gorm:"default:0" json:"points"`   // 积分（points 类型）
	CreatedAt  time.Time `json:"createdAt"`
}

// ==================== 邮件群发模块 ====================

// EmailRecipient 邮件群发收件人
// status: pending / sent / failed
// isLast: 唯一 true，表示这封是 kampagn 的最后一封（gmshe@qq.com），必须等所有非 last 发完才发
type EmailRecipient struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Email     string     `gorm:"size:100;uniqueIndex" json:"email"`
	Status    string     `gorm:"size:20;default:'pending'" json:"status"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	ErrorMsg  string     `gorm:"size:500" json:"errorMsg,omitempty"`
	IsLast    bool       `gorm:"default:false" json:"isLast"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// EmailCampaign 邮件群发活动元数据（单活动，字段扁平化便于管理）
// Notification 后台站内通知群发记录
type Notification struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:200;not null" json:"title"`
	Content     string    `gorm:"type:text" json:"content"`
	Target      string    `gorm:"size:20;default:'all'" json:"target"` // all / free / levels
	Levels      string    `gorm:"size:200" json:"levels"`              // target=levels 时的等级 key，逗号分隔
	Attachments string    `gorm:"type:text" json:"attachments"`        // JSON 数组 [{name,url,size}]
	SentCount   int       `gorm:"default:0" json:"sentCount"`
	Status      string    `gorm:"size:20;default:'done'" json:"status"` // done / failed
	CreatedAt   time.Time `json:"createdAt"`
}

type EmailCampaign struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:100" json:"name"`
	Subject    string     `gorm:"size:200" json:"subject"`
	BodyHTML   string     `gorm:"type:text" json:"bodyHtml"`
	ImagePath  string     `gorm:"size:500" json:"imagePath"`
	DailyLimit int        `gorm:"default:400" json:"dailyLimit"`
	Status     string     `gorm:"size:20;default:'paused'" json:"status"` // running / paused / finished
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

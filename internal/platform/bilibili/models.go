package bilibili

// PreUpInfo 预上传返回信息
type PreUpInfo struct {
	OK              int64       `json:"OK"`
	Auth            string      `json:"auth"`
	BizId           int64       `json:"biz_id"`
	ChunkRetry      int64       `json:"chunk_retry"`
	ChunkRetryDelay int64       `json:"chunk_retry_delay"`
	ChunkSize       int64       `json:"chunk_size"`
	Endpoint        string      `json:"endpoint"`
	Endpoints       []string    `json:"endpoints"`
	ExposeParams    interface{} `json:"expose_params"`
	PutQuery        string      `json:"put_query"`
	Threads         int64       `json:"threads"`
	Timeout         int64       `json:"timeout"`
	Uip             string      `json:"uip"`
	UposUri         string      `json:"upos_uri"`
}

// UpInfo 上传返回信息
type UpInfo struct {
	Location string `json:"location"`
	Etag     string `json:"etag"`
	OK       int64  `json:"OK"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	UploadId string `json:"upload_id"`
}

// ReqJson 分片确认请求
type ReqJson struct {
	Parts []Part `json:"parts"`
}

// Part 分片信息
type Part struct {
	PartNumber int64  `json:"partNumber"`
	ETag       string `json:"eTag"`
}

// AddReqJson 提交视频请求
type AddReqJson struct {
	Copyright        int64    `json:"copyright"` // 1:原创 2:转载
	Cover            string   `json:"cover"`     // 封面url
	Title            string   `json:"title"`     // 视频标题
	Tid              int64    `json:"tid"`       // 分区id
	Tag              string   `json:"tag"`       // 标签 , 分割
	DescFormatId     int64    `json:"desc_format_id"`
	Desc             string   `json:"desc"`             // 简介
	Source           string   `json:"source,omitempty"` // 来源
	Dynamic          string   `json:"dynamic"`
	Interactive      int64    `json:"interactive"`
	Videos           []Video  `json:"videos"`
	ActReserveCreate int64    `json:"act_reserve_create"`
	NoDisturbance    int64    `json:"no_disturbance"`
	NoReprint        int64    `json:"no_reprint"`
	Subtitle         Subtitle `json:"subtitle"`
	Dolby            int64    `json:"dolby"`
	LosslessMusic    int64    `json:"lossless_music"`
	Csrf             string   `json:"csrf"`
}

// Video 视频信息
type Video struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Desc     string `json:"desc"`
}

// Subtitle 字幕信息
type Subtitle struct {
	Open int64  `json:"open"`
	Lan  string `json:"lan"`
}

// CoverInfo 封面上传返回信息
type CoverInfo struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Ttl     int64  `json:"ttl"`
	Data    struct {
		Url string `json:"url"`
	} `json:"data"`
}

// UserInfo 用户信息
type UserInfo struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    struct {
		IsLogin bool   `json:"isLogin"`
		Uname   string `json:"uname"`
		Mid     int64  `json:"mid"`
	} `json:"data"`
}

package g

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"runtime"
)

type ConfigStruct struct {
	Host                       string `json:"host"`
	Port                       int    `json:"port"`
	DB                         string `json:"db"`
	DBNAME                     string `json:"dbname"`
	CookieSecret               string `json:"cookie_secret"`
	SmtpUsername               string `json:"smtp_username"`
	SmtpPassword               string `json:"smtp_password"`
	SmtpHost                   string `json:"smtp_host"`
	SmtpAddr                   string `json:"smtp_addr"`
	FromEmail                  string `json:"from_email"`
	Superusers                 string `json:"superusers"`
	TimeZoneOffset             int64  `json:"time_zone_offset"`
	AnalyticsFile              string `json:"analytics_file"`
	StaticFileVersion          int    `json:"static_file_version"`
	QiniuAccessKey             string `json:"qiniu_access_key"`
	QiniuSecretKey             string `json:"qiniu_secret_key"`
	QiniuBucket                string `json:"qiniu_bucket"`
	QiniuDomain                string `json:"qiniu_domain"`
	CookieSecure               bool   `json:"cookie_secure"`
}

var (
	Config        ConfigStruct
	analyticsCode template.HTML // 网站统计分析代码
	shareCode     template.HTML // 分享代码
	goVersion     = runtime.Version()
)

func parseJsonFile(path string, v interface{}) {
	file, err := os.Open(path)
	if err != nil {
		logger.Fatal("配置文件读取失败:", err)
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	err = dec.Decode(v)
	if err != nil {
		logger.Fatal("配置文件解析失败:", err)
	}
}

func getDefaultCode(path string) (code template.HTML) {
	if path != "" {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			logger.Fatal("文件 " + path + " 没有找到")
		}
		code = template.HTML(string(content))
	}
	return
}


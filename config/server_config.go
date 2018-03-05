// Package config provides Config struct for proxy.
package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// Config 保存代理服务器的配置
type Config struct {
	// 代理服务器工作端口,eg:":8080"
	Listen string `json:"listen"`

	// web管理端口
	WebListen string `json:"weblisten"`

	// 反向代理标志
	Reverse bool `json:"reverse"`

	// 反向代理目标地址,eg:"127.0.0.1:8090"
	ProxyPass string `json:"proxy_pass"`

	// 认证标志
	Auth bool `json:"auth"`
	
	// 认证失败时将请求发送到 failover (明文发送)
	Failover string `json:"failover"`

	// 缓存标志
	Cache bool `json:"cache"`

	// 缓存定期刷新时间，单位分钟
	CacheTimeout int64 `json:"cache_timeout"`

	// 日志信息，1输出Debug信息，0输出普通监控信息
	Log int `json:"log"`

	// 网站屏蔽列表
	GFWList []string `json:"gfwlist"`

	// 管理员账号
	Admin map[string]string `json:"admin"`
	// 普通用户账户
	User map[string]string `json:"user"`
}

// GetConfig gets config from json file.
// GetConfig 从指定json文件读取config配置
func (c *Config) GetConfig(filename string) error {
	c.Admin = make(map[string]string)
	c.User = make(map[string]string)

	configFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer configFile.Close()

	br := bufio.NewReader(configFile)
	err = json.NewDecoder(br).Decode(c)
	if err != nil {
		return err
	}
	return nil
}

// WriteToFile writes config into json file.
// WriteToFile 将config配置写入特定json文件
func (c *Config) WriteToFile(filename string) error {
	configFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer configFile.Close()

	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	cjson := string(b)
	cspilts := strings.Split(cjson, ",")
	cjson = strings.Join(cspilts, ",\n")

	configFile.Write([]byte(cjson))

	return nil
}

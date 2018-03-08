// Package config provides Config struct for proxy.
package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Client 客户端的配置
type Client struct {
	// 代理服务器工作端口,eg:":8080"
	Proxy string `json:"listen"`

	// 代理服务器域名
	Hostname string `json:"hostname"`

	// web管理端口
	WebListen string `json:"weblisten"`

	// 代理用户账户
	Username string `json:"username"`
	Password string `json:"password"`

	// 管理员账号
	Admin map[string]string `json:"admin"`

	// HTTP 代理监听地址
	HTTPListen string `json:"http"`

	// Socks 代理监听地址
	SocksListen string `json:"socks"`

	// Redirect 监听地址
	RedirListen string `json:"redirect"`

	// TProxy 监听地址
	TProxyListen string `json:"tproxy"`

	// Bogus DNS 监听地址
	DNSListen string `json:"dns"`

	// Bogus DNS 伪 IP 前缀
	DNSPrefix string `json:"dnsprefix"`

	// Bogus DNS TTL
	DNSTTL uint `json:"dnsttl"`

	// 忽略 TLS 证书检查
	InsecureSkipVerify bool `json:"insecure"`

	// 连接保持时间
	IdleTime time.Duration `json:"idletime"`

	// 最大空闲连接数
	MaxIdleConnections int `json:"maxconn"`

	// 日志信息，1输出Debug信息，0输出普通监控信息
	LogLevel int `json:"loglevel"`
}

// GetConfig gets config from json file.
// GetConfig 从指定json文件读取config配置
func (c *Client) GetConfig(filename string) error {
	c.Admin = make(map[string]string)

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
func (c *Client) WriteToFile(filename string) error {
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

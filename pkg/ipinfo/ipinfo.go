// Package ipinfo 提供IP地址信息查询功能
package ipinfo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IPInfo 结构体存储IP地址的详细信息
type IPInfo struct {
	Country  string `json:"country"`  // 国家
	Region   string `json:"region"`   // 地区/省份
	City     string `json:"city"`     // 城市
	ISP      string `json:"isp"`      // 网络服务提供商
	Location string `json:"location"` // 地理位置
}

// Client 结构体封装了IP信息查询客户端
type Client struct {
	apiURL        string        // API接口地址
	language      string        // 返回信息的语言
	timeout       int          // 请求超时时间（秒）
	retryCount    int          // 重试次数
	retryInterval int          // 重试间隔（秒）
	httpClient    *http.Client // HTTP客户端
}

// NewClient 创建并初始化一个新的IP信息查询客户端
// 参数:
//   - apiURL: API接口地址
//   - language: 返回信息的语言
//   - timeout: 请求超时时间（秒）
//   - retryCount: 重试次数
//   - retryInterval: 重试间隔（秒）
// 返回:
//   - *Client: 初始化后的客户端实例
func NewClient(apiURL, language string, timeout, retryCount, retryInterval int) *Client {
	return &Client{
		apiURL:        apiURL,
		language:      language,
		timeout:       timeout,
		retryCount:    retryCount,
		retryInterval: retryInterval,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// GetIPInfo 获取指定IP地址的详细信息
// 参数:
//   - ip: 要查询的IP地址
// 返回:
//   - *IPInfo: IP地址的详细信息
//   - error: 查询过程中的错误信息
func (c *Client) GetIPInfo(ip string) (*IPInfo, error) {
	url := fmt.Sprintf("%s/%s?lang=%s", c.apiURL, ip, c.language)
	
	var lastErr error
	for i := 0; i <= c.retryCount; i++ {
		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = err
			if i < c.retryCount {
				time.Sleep(time.Duration(c.retryInterval) * time.Second)
				continue
			}
			return nil, fmt.Errorf("获取IP信息失败: %v", err)
		}
		defer resp.Body.Close()

		var ipInfo IPInfo
		if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
			lastErr = err
			if i < c.retryCount {
				time.Sleep(time.Duration(c.retryInterval) * time.Second)
				continue
			}
			return nil, fmt.Errorf("解析IP信息失败: %v", err)
		}

		return &ipInfo, nil
	}

	return nil, fmt.Errorf("获取IP信息失败: %v", lastErr)
}

// FormatIPInfo 格式化IP地址信息为可读字符串
// 参数:
//   - ip: 要格式化的IP地址
// 返回:
//   - string: 格式化后的IP信息字符串
func (c *Client) FormatIPInfo(ip string) string {
	ipInfo, err := c.GetIPInfo(ip)
	if err != nil {
		return fmt.Sprintf("IP: %s (无法获取属地信息)", ip)
	}

	return fmt.Sprintf("IP: %s\n属地: %s %s %s\nISP: %s", 
		ip, ipInfo.Country, ipInfo.Region, ipInfo.City, ipInfo.ISP)
} 
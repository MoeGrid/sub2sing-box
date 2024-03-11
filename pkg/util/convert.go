package util

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	. "sub2sing-box/internal"
	"sub2sing-box/internal/model"
	"sub2sing-box/pkg/parser"
)

func MergeTemplate(proxies []model.Proxy, template string) (string, error) {
	config, err := ReadTemplate(template)
	proxyTags := make([]string, 0)
	if err != nil {
		return "", err
	}
	for _, p := range proxies {
		proxyTags = append(proxyTags, p.Tag)
	}
	ps, err := json.Marshal(&proxies)
	if err != nil {
		return "", err
	}
	var newOutbounds []model.Outbound
	err = json.Unmarshal(ps, &newOutbounds)
	if err != nil {
		return "", err
	}
	for i, outbound := range config.Outbounds {
		if outbound.Type == "urltest" || outbound.Type == "selector" {
			var parsedOutbound []string = make([]string, 0)
			for _, o := range outbound.Outbounds {
				if o == "<all-proxy-tags>" {
					parsedOutbound = append(parsedOutbound, proxyTags...)
				} else {
					parsedOutbound = append(parsedOutbound, o)
				}
			}
			config.Outbounds[i].Outbounds = parsedOutbound
		}
	}
	config.Outbounds = append(config.Outbounds, newOutbounds...)
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ConvertCProxyToSProxy(proxy string) (model.Proxy, error) {
	for prefix, parseFunc := range parser.ParserMap {
		if strings.HasPrefix(proxy, prefix) {
			proxy, err := parseFunc(proxy)
			if err != nil {
				return model.Proxy{}, err
			}
			return proxy, nil
		}
	}
	return model.Proxy{}, errors.New("Unknown proxy format")
}

func ConvertCProxyToJson(proxy string) (string, error) {
	sProxy, err := ConvertCProxyToSProxy(proxy)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(&sProxy)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FetchSubscription(url string, maxRetryTimes int) (string, error) {
	retryTime := 0
	var err error
	for retryTime < maxRetryTimes {
		resp, err := http.Get(url)
		if err != nil {
			retryTime++
			continue
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			retryTime++
			continue
		}
		return string(data), err
	}
	return "", err
}

func ConvertSubscriptionsToSProxy(urls []string) ([]model.Proxy, error) {
	proxyList := make([]model.Proxy, 0)
	for _, url := range urls {
		data, err := FetchSubscription(url, 3)
		if err != nil {
			return nil, err
		}
		proxy, err := DecodeBase64(data)
		if err != nil {
			return nil, err
		}
		proxies := strings.Split(proxy, "\n")
		for _, p := range proxies {
			for prefix, parseFunc := range parser.ParserMap {
				if strings.HasPrefix(p, prefix) {
					proxy, err := parseFunc(p)
					if err != nil {
						return nil, err
					}
					proxyList = append(proxyList, proxy)
				}
			}
		}
	}
	return proxyList, nil
}

func ConvertSubscriptionsToJson(urls []string) (string, error) {
	proxyList, err := ConvertSubscriptionsToSProxy(urls)
	if err != nil {
		return "", err
	}
	result, err := json.Marshal(proxyList)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func ReadTemplate(path string) (model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Config{}, err
	}
	var res model.Config
	err = json.Unmarshal(data, &res)
	if err != nil {
		return model.Config{}, err
	}
	return res, nil
}

func DeleteProxy(proxies []model.Proxy, regex string) ([]model.Proxy, error) {
	reg, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	var newProxies []model.Proxy
	for _, p := range proxies {
		if !reg.MatchString(p.Tag) {
			newProxies = append(newProxies, p)
		}
	}
	return newProxies, nil
}

func RenameProxy(proxies []model.Proxy, regex string, replaceText string) ([]model.Proxy, error) {
	reg, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	for i, p := range proxies {
		if reg.MatchString(p.Tag) {
			proxies[i].Tag = reg.ReplaceAllString(p.Tag, replaceText)
		}
	}
	return proxies, nil
}

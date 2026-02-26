package helper

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/client"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
	"github.com/dlclark/regexp2"
)

func ChannelHttpClient(channel *model.Channel) (*http.Client, error) {
	if channel == nil {
		return nil, errors.New("channel is nil")
	}
	var (
		httpClient *http.Client
		err        error
	)
	if !channel.Proxy {
		httpClient, err = client.GetHTTPClientSystemProxy(false)
	} else if channel.ChannelProxy == nil || strings.TrimSpace(*channel.ChannelProxy) == "" {
		httpClient, err = client.GetHTTPClientSystemProxy(true)
	} else {
		httpClient, err = client.GetHTTPClientCustomProxy(strings.TrimSpace(*channel.ChannelProxy))
	}
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		return nil, errors.New("http client is nil")
	}
	return httpClient, nil
}

func ChannelBaseUrlDelayUpdate(channel *model.Channel, ctx context.Context) {
	if channel == nil {
		return
	}
	httpClient, err := ChannelHttpClient(channel)
	if err != nil {
		log.Warnf("failed to get http client (channel=%d): %v", channel.ID, err)
		return
	}

	newBaseUrls := make([]model.BaseUrl, 0, len(channel.BaseUrls))
	for _, baseUrl := range channel.BaseUrls {
		url := strings.TrimSpace(baseUrl.URL)
		if url == "" {
			log.Warnf("skip empty base url (channel=%d)", channel.ID)
			continue
		}
		delay, err := GetUrlDelay(httpClient, url, ctx)
		if err != nil {
			log.Warnf("failed to get url delay (channel=%d, url=%s): %v", channel.ID, url, err)
			continue
		}
		newBaseUrls = append(newBaseUrls, model.BaseUrl{
			URL:   url,
			Delay: delay,
		})
	}
	if len(newBaseUrls) > 0 {
		op.ChannelBaseUrlUpdate(channel.ID, newBaseUrls)
	}
}

func ChannelAutoGroup(channel *model.Channel, ctx context.Context) {
	if channel == nil {
		return
	}
	if channel.AutoGroup == model.AutoGroupTypeNone {
		return
	}
	groups, err := op.GroupList(ctx)
	if err != nil {
		log.Warnf("get group list failed: %v", err)
		return
	}

	channelModelNames := xstrings.SplitTrimCompact(",", channel.Model, channel.CustomModel)
	if len(channelModelNames) == 0 {
		return
	}

	for _, group := range groups {
		matchedModelNames := make([]string, 0, len(channelModelNames))

		switch channel.AutoGroup {
		case model.AutoGroupTypeExact:
			for _, modelName := range channelModelNames {
				if strings.EqualFold(modelName, group.Name) {
					matchedModelNames = append(matchedModelNames, modelName)
				}
			}

		case model.AutoGroupTypeFuzzy:
			groupNameLower := strings.ToLower(strings.TrimSpace(group.Name))
			if groupNameLower == "" {
				continue
			}
			for _, modelName := range channelModelNames {
				if strings.Contains(strings.ToLower(modelName), groupNameLower) {
					matchedModelNames = append(matchedModelNames, modelName)
				}
			}

		case model.AutoGroupTypeRegex:
			if group.MatchRegex == "" {
				for _, modelName := range channelModelNames {
					if strings.EqualFold(modelName, group.Name) {
						matchedModelNames = append(matchedModelNames, modelName)
					}
				}
				break
			}

			re, err := regexp2.Compile(group.MatchRegex, regexp2.ECMAScript)
			if err != nil {
				log.Warnf("compile regex failed (channel=%d group=%d regex=%q): %v", channel.ID, group.ID, group.MatchRegex, err)
				continue
			}
			for _, modelName := range channelModelNames {
				matched, err := re.MatchString(modelName)
				if err != nil {
					log.Warnf("match regex failed (channel=%d group=%d regex=%q model=%q): %v", channel.ID, group.ID, group.MatchRegex, modelName, err)
					continue
				}
				if matched {
					matchedModelNames = append(matchedModelNames, modelName)
				}
			}
		}

		if len(matchedModelNames) > 0 {
			items := make([]model.GroupIDAndLLMName, 0, len(matchedModelNames))
			for _, modelName := range matchedModelNames {
				items = append(items, model.GroupIDAndLLMName{
					ChannelID: channel.ID,
					ModelName: modelName,
				})
			}
			if err := op.GroupItemBatchAdd(group.ID, items, ctx); err != nil {
				log.Warnf("group item batch add failed (channel=%d group=%d): %v", channel.ID, group.ID, err)
			}
		}
	}
}

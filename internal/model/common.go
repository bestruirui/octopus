package model

type FetchModelRequest struct {
	Url   string      `json:"url"`
	Key   string      `json:"key"`
	Type  ChannelType `json:"type"`
	Proxy bool        `json:"proxy"`
}

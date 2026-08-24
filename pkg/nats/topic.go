package nats

import (
	"strings"
)

const (
	Separator      = "."
	AllWildcard    = ">"
	SingleWildcard = "*"
	DirectionUp    = "up"
	DirectionDown  = "down"
)

type TopicBuilder struct {
	parts []string
}

func NewTopicBuilder() *TopicBuilder {
	return &TopicBuilder{parts: make([]string, 0)}
}

func (b *TopicBuilder) Domains(domainID string) *TopicBuilder {
	b.parts = append(b.parts, "domains", domainID)
	return b
}

func (b *TopicBuilder) DomainsAll() *TopicBuilder {
	b.parts = append(b.parts, "domains", AllWildcard)
	return b
}

func (b *TopicBuilder) Devices() *TopicBuilder {
	b.parts = append(b.parts, "devices")
	return b
}

func (b *TopicBuilder) Region(region string) *TopicBuilder {
	b.parts = append(b.parts, region)
	return b
}

func (b *TopicBuilder) RegionAll() *TopicBuilder {
	b.parts = append(b.parts, AllWildcard)
	return b
}

func (b *TopicBuilder) DeviceType(dt string) *TopicBuilder {
	b.parts = append(b.parts, dt)
	return b
}

func (b *TopicBuilder) DeviceTypeAll() *TopicBuilder {
	b.parts = append(b.parts, AllWildcard)
	return b
}

func (b *TopicBuilder) DeviceID(id string) *TopicBuilder {
	b.parts = append(b.parts, id)
	return b
}

func (b *TopicBuilder) DeviceIDAll() *TopicBuilder {
	b.parts = append(b.parts, AllWildcard)
	return b
}

func (b *TopicBuilder) Direction(dir string) *TopicBuilder {
	b.parts = append(b.parts, dir)
	return b
}

func (b *TopicBuilder) Build() string {
	return strings.Join(b.parts, Separator)
}

func DeviceReportTopic(domain, region, deviceType, deviceID string) string {
	return NewTopicBuilder().Domains(domain).Devices().Region(region).DeviceType(deviceType).DeviceID(deviceID).Direction(DirectionUp).Build()
}

func DeviceCommandTopic(domain, region, deviceType, deviceID string) string {
	return NewTopicBuilder().Domains(domain).Devices().Region(region).DeviceType(deviceType).DeviceID(deviceID).Direction(DirectionDown).Build()
}

func DomainAllReportTopic(domain string) string {
	return NewTopicBuilder().Domains(domain).Devices().RegionAll().DeviceTypeAll().DeviceIDAll().Direction(DirectionUp).Build()
}

func AllSensorReportTopic() string {
	return NewTopicBuilder().DomainsAll().Devices().RegionAll().DeviceType("sensor").DeviceIDAll().Direction(DirectionUp).Build()
}

func SystemEventsTopic() string {
	return "system.events"
}

func AlertEventTopic(domainID string) string {
	return "system.alerts." + domainID
}

func MatchWildcard(pattern, topic string) bool {
	patternParts := strings.Split(pattern, Separator)
	topicParts := strings.Split(topic, Separator)
	return matchParts(patternParts, topicParts, 0, 0)
}

func matchParts(pattern, topic []string, pi, ti int) bool {
	for pi < len(pattern) {
		if pattern[pi] == AllWildcard {
			// Non-greedy: try matching remaining pattern against all suffixes
			for remaining := ti; remaining <= len(topic); remaining++ {
				if matchParts(pattern, topic, pi+1, remaining) {
					return true
				}
			}
			return false
		}
		if ti >= len(topic) {
			return false
		}
		if pattern[pi] != SingleWildcard && pattern[pi] != topic[ti] {
			return false
		}
		pi++
		ti++
	}
	return ti == len(topic)
}

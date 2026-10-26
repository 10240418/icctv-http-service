package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"icctv-http-service/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var mediaMTXChannelPathPattern = regexp.MustCompile(`^channel([1-9][0-9]*)$`)

const maxChannelID = 1<<53 - 1

// Device versions expose either native MediaMTX pagination or normalized fields.
// Keep the admin API response stable and reject missing totals instead of
// treating an unrecognized pagination format as an empty catalogue.
func decodeMediaMTXPaths(reader io.Reader, page int) (*MediaMTXPathsListResponse, error) {
	var wire struct {
		Items      []MediaMTXPathItem `json:"items"`
		ItemsTotal *int               `json:"itemsTotal"`
		ItemCount  *int               `json:"itemCount"`
	}
	if err := json.NewDecoder(reader).Decode(&wire); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	total := wire.ItemsTotal
	if total == nil {
		total = wire.ItemCount
	}
	if total == nil || *total < 0 {
		return nil, fmt.Errorf("MediaMTX path response is missing a valid total count")
	}
	return &MediaMTXPathsListResponse{Items: wire.Items, ItemsPage: page, ItemsTotal: *total}, nil
}

func numericChannel(path string) int {
	matches := mediaMTXChannelPathPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return 0
	}
	channel, err := strconv.Atoi(matches[1])
	if err != nil || int64(channel) > maxChannelID {
		return 0
	}
	return channel
}

func concreteMediaMTXPath(path string) bool {
	return path != "" && path != "all" && path != "all_others" && !strings.HasPrefix(path, "~")
}

// Explicit building selections can use every known channel. Keep this separate
// from defaultBuildingChannels, which determines permissions for new bindings.
func catalogueChannels(device models.OrangePi) []int {
	channels := append(append([]int{}, device.AllChannels...), device.UserChannels...)
	for key := range device.ChannelPaths {
		if channel := numericChannel(key); channel > 0 {
			channels = append(channels, channel)
		}
	}
	return normalizeChannels(channels)
}

// Discovery adds options, not permissions. Stable IDs are never reused, including
// paths removed from MediaMTX, so a stale building rule cannot grant a new stream.
func (s *OrangePiService) registerMediaMTXPaths(ctx context.Context, id int64, items []MediaMTXPathItem) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device models.OrangePi
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&device, id).Error; err != nil {
			return err
		}
		paths := make(map[string]string, len(device.ChannelPaths)+len(items))
		byPath := make(map[string]int, len(paths))
		reserved := make(map[int]bool)
		highest := 0
		reserve := func(channel int) {
			if channel > 0 {
				reserved[channel] = true
				if channel > highest {
					highest = channel
				}
			}
		}
		for key, path := range device.ChannelPaths {
			paths[key] = path
			channel := numericChannel(key)
			reserve(channel)
			byPath[path] = channel
		}
		// Legacy rules already refer to channelN; keep that meaning even if that
		// path is currently absent or a custom path is discovered first.
		var links []models.OrangePiBuilding
		if err := tx.Where("orange_pi_id = ?", id).Find(&links).Error; err != nil {
			return err
		}
		legacy := append(append([]int{}, device.AllChannels...), device.UserChannels...)
		for _, link := range links {
			legacy = append(legacy, link.AllowedChannels...)
		}
		for _, channel := range normalizeChannels(legacy) {
			reserve(channel)
			key := fmt.Sprintf("channel%d", channel)
			if _, exists := paths[key]; !exists {
				paths[key] = key
				byPath[key] = channel
			}
		}
		names := make([]string, 0, len(items))
		seen := make(map[string]bool)
		for _, item := range items {
			if concreteMediaMTXPath(item.Name) && !seen[item.Name] {
				names = append(names, item.Name)
				seen[item.Name] = true
			}
		}
		sort.Strings(names)
		// Reserve canonical numeric paths before allocating IDs for named paths.
		for _, name := range names {
			if _, exists := byPath[name]; exists {
				continue
			}
			channel := numericChannel(name)
			if channel > 0 && !reserved[channel] {
				reserve(channel)
				paths[name] = name
				byPath[name] = channel
			}
		}
		for _, name := range names {
			if _, exists := byPath[name]; exists {
				continue
			}
			if int64(highest) >= maxChannelID {
				return fmt.Errorf("channel ID space exhausted")
			}
			highest++
			paths[fmt.Sprintf("channel%d", highest)] = name
			byPath[name] = highest
		}
		device.ChannelPaths = paths
		// Update just the catalogue to avoid clobbering concurrent permission edits.
		return tx.Model(&device).Select("ChannelPaths").Updates(&device).Error
	})
}

// SyncMediaMTXChannels reads every page before committing the catalogue. It never
// sends a write request to MediaMTX or modifies user/building visibility.
func (s *OrangePiService) SyncMediaMTXChannels(ctx context.Context, id int64, token string) (*models.OrangePi, error) {
	const pageSize = 50
	items := make([]MediaMTXPathItem, 0)
	seen := make(map[string]bool)
	total := -1
	for page := 0; ; page++ {
		result, err := s.fetchMediaMTXPaths(ctx, id, token, page, pageSize)
		if err != nil {
			return nil, fmt.Errorf("read MediaMTX paths page %d: %w", page, err)
		}
		if total < 0 {
			total = result.ItemsTotal
		}
		if total < 0 || result.ItemsTotal != total {
			return nil, fmt.Errorf("MediaMTX path count changed during sync; retry")
		}
		for _, item := range result.Items {
			if seen[item.Name] {
				return nil, fmt.Errorf("duplicate MediaMTX path across sync pages; retry")
			}
			seen[item.Name] = true
		}
		items = append(items, result.Items...)
		if (page+1)*pageSize >= result.ItemsTotal {
			if len(items) != total {
				return nil, fmt.Errorf("incomplete MediaMTX path catalogue: got %d, expected %d", len(items), total)
			}
			break
		}
		if len(result.Items) == 0 {
			return nil, fmt.Errorf("incomplete MediaMTX paths page %d", page)
		}
	}
	if err := s.registerMediaMTXPaths(ctx, id, items); err != nil {
		return nil, err
	}
	var device models.OrangePi
	if err := s.db.WithContext(ctx).Preload("Buildings").First(&device, id).Error; err != nil {
		return nil, err
	}
	hydrateOrangePi(&device)
	return &device, nil
}

package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// KernelConfigService manages kernel .config artifacts in storage.
// It generates config fragments at distribution create/update time
// and stores them as distribution artifacts.
type KernelConfigService struct {
	storage storage.Backend
}

// NewKernelConfigService creates a new kernel config service.
func NewKernelConfigService(store storage.Backend) *KernelConfigService {
	return &KernelConfigService{storage: store}
}

// KernelConfigArtifactPath returns the canonical storage key for a
// distribution's kernel config artifact.
func KernelConfigArtifactPath(ownerID, distributionID string) string {
	return fmt.Sprintf("distribution/%s/%s/kernel/.config", ownerID, distributionID)
}

// GenerateAndStore generates a kernel config fragment based on the
// distribution's config and stores it as an artifact. For defconfig
// and options modes, this produces a config fragment (not a full .config).
// For custom mode, it re-stores the user's uploaded config at the canonical path.
func (s *KernelConfigService) GenerateAndStore(ctx context.Context, dist *db.Distribution) error {
	if s.storage == nil {
		return fmt.Errorf("storage backend not configured")
	}
	if dist.Config == nil {
		return nil
	}

	mode := dist.Config.Core.Kernel.ConfigMode
	if mode == "" {
		mode = db.KernelConfigModeDefconfig
	}

	var content []byte
	var err error

	switch mode {
	case db.KernelConfigModeDefconfig:
		content = s.generateDefconfigFragment(dist)
	case db.KernelConfigModeOptions:
		content = s.generateOptionsFragment(dist)
	case db.KernelConfigModeCustom:
		content, err = s.fetchCustomConfig(ctx, dist)
		if err != nil {
			return fmt.Errorf("failed to fetch custom config: %w", err)
		}
	default:
		return fmt.Errorf("unknown kernel config mode: %s", mode)
	}

	key := KernelConfigArtifactPath(dist.OwnerID, dist.ID)
	reader := bytes.NewReader(content)

	if err := s.storage.Upload(ctx, key, reader, int64(len(content)), "text/plain"); err != nil {
		return fmt.Errorf("failed to store kernel config artifact: %w", err)
	}

	log.Info("Stored kernel config artifact",
		"distribution_id", dist.ID,
		"mode", mode,
		"key", key,
		"size", len(content))

	return nil
}

// StoreCustomConfig stores a user-provided kernel config at the canonical path
// with an LDF_CONFIG_MODE=custom header prepended.
func (s *KernelConfigService) StoreCustomConfig(ctx context.Context, dist *db.Distribution, configData []byte) error {
	if s.storage == nil {
		return fmt.Errorf("storage backend not configured")
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# LDF Kernel Configuration\n"+
		"# Mode: custom (user-provided)\n"+
		"# Distribution: %s\n"+
		"#\n"+
		"LDF_CONFIG_MODE=custom\n"+
		"LDF_TARGET_ARCH=%s\n\n",
		dist.ID, dist.Config.Target.Type))
	buf.Write(configData)

	content := buf.Bytes()
	key := KernelConfigArtifactPath(dist.OwnerID, dist.ID)
	reader := bytes.NewReader(content)

	if err := s.storage.Upload(ctx, key, reader, int64(len(content)), "text/plain"); err != nil {
		return fmt.Errorf("failed to store custom kernel config: %w", err)
	}

	log.Info("Stored custom kernel config artifact",
		"distribution_id", dist.ID,
		"key", key,
		"size", len(content))

	return nil
}

// generateDefconfigFragment creates a config fragment for defconfig mode.
// Contains LDF metadata + recommended kernel options.
func (s *KernelConfigService) generateDefconfigFragment(dist *db.Distribution) []byte {
	arch := db.TargetArch(dist.Config.Target.Type)

	options := GetRecommendedKernelOptions(dist.Config, arch)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# LDF Kernel Configuration Fragment\n"+
		"# Mode: defconfig\n"+
		"# Distribution: %s\n"+
		"# Kernel: %s\n"+
		"#\n"+
		"# At build time: make <defconfig>, merge this fragment, make olddefconfig\n"+
		"#\n"+
		"LDF_CONFIG_MODE=defconfig\n"+
		"LDF_TARGET_ARCH=%s\n\n",
		dist.ID, dist.Config.Core.Kernel.Version, arch))

	writeOptions(&buf, options)
	return buf.Bytes()
}

// generateOptionsFragment creates a config fragment for options mode.
// Contains LDF metadata + recommended options + user-specified options.
func (s *KernelConfigService) generateOptionsFragment(dist *db.Distribution) []byte {
	arch := db.TargetArch(dist.Config.Target.Type)

	// Start with recommended options
	options := GetRecommendedKernelOptions(dist.Config, arch)

	// Merge user-specified options (override recommended)
	for key, value := range dist.Config.Core.Kernel.ConfigOptions {
		if !strings.HasPrefix(key, "CONFIG_") {
			key = "CONFIG_" + key
		}
		options[key] = value
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# LDF Kernel Configuration Fragment\n"+
		"# Mode: options (defconfig + custom options)\n"+
		"# Distribution: %s\n"+
		"# Kernel: %s\n"+
		"#\n"+
		"# At build time: make <defconfig>, merge this fragment, make olddefconfig\n"+
		"#\n"+
		"LDF_CONFIG_MODE=options\n"+
		"LDF_TARGET_ARCH=%s\n\n",
		dist.ID, dist.Config.Core.Kernel.Version, arch))

	writeOptions(&buf, options)
	return buf.Bytes()
}

// fetchCustomConfig downloads the user's custom config from its original storage
// path and wraps it with LDF metadata headers.
func (s *KernelConfigService) fetchCustomConfig(ctx context.Context, dist *db.Distribution) ([]byte, error) {
	customPath := dist.Config.Core.Kernel.CustomConfigPath
	if customPath == "" {
		return nil, fmt.Errorf("custom config path is empty")
	}

	reader, _, err := s.storage.Download(ctx, customPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download custom config from %s: %w", customPath, err)
	}
	defer reader.Close()

	userData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom config: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# LDF Kernel Configuration\n"+
		"# Mode: custom (user-provided)\n"+
		"# Distribution: %s\n"+
		"# Source: %s\n"+
		"#\n"+
		"LDF_CONFIG_MODE=custom\n"+
		"LDF_TARGET_ARCH=%s\n\n",
		dist.ID, customPath, dist.Config.Target.Type))
	buf.Write(userData)

	return buf.Bytes(), nil
}

// writeOptions writes sorted CONFIG_ options to a buffer in kconfig format.
func writeOptions(buf *bytes.Buffer, options map[string]string) {
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := options[key]
		switch value {
		case "y", "m":
			buf.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		case "n":
			buf.WriteString(fmt.Sprintf("# %s is not set\n", key))
		default:
			if strings.HasPrefix(value, "\"") {
				buf.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			} else {
				buf.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
			}
		}
	}
}

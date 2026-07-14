package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/database"
	"gravitylink/backend/internal/model"
)

func main() {
	var domainsPath string
	var linksPath string
	var dryRun bool
	flag.StringVar(&domainsPath, "domains", "", "path to domains CSV")
	flag.StringVar(&linksPath, "links", "", "path to links CSV")
	flag.BoolVar(&dryRun, "dry-run", false, "parse and validate only")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if domainsPath == "" && linksPath == "" {
		logger.Error("nothing to migrate", "usage", "migrate-legacy -domains domains.csv -links links.csv")
		os.Exit(1)
	}

	cfg := config.Load()
	db, err := database.ConnectMySQL(cfg.MySQLDSN)
	if err != nil {
		logger.Error("connect mysql failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if domainsPath != "" {
		domains, err := readDomains(domainsPath)
		if err != nil {
			logger.Error("read domains failed", "error", err)
			os.Exit(1)
		}
		logger.Info("domains parsed", "count", len(domains), "dry_run", dryRun)
		if !dryRun {
			if err := db.WithContext(ctx).CreateInBatches(domains, 100).Error; err != nil {
				logger.Error("insert domains failed", "error", err)
				os.Exit(1)
			}
		}
	}

	if linksPath != "" {
		links, err := readLinks(linksPath)
		if err != nil {
			logger.Error("read links failed", "error", err)
			os.Exit(1)
		}
		logger.Info("links parsed", "count", len(links), "dry_run", dryRun)
		if !dryRun {
			if err := db.WithContext(ctx).CreateInBatches(links, 100).Error; err != nil {
				logger.Error("insert links failed", "error", err)
				os.Exit(1)
			}
		}
	}

	logger.Info("migration finished")
}

func readDomains(path string) ([]model.Domain, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	domains := make([]model.Domain, 0, len(rows))
	for index, row := range rows {
		host := strings.TrimSpace(row["host"])
		domainType := defaultString(row["type"], model.DomainTypeEntry)
		scheme := defaultString(row["scheme"], "https")
		if host == "" {
			return nil, fmt.Errorf("domains row %d: host is required", index+2)
		}
		remark := optionalCSVString(row["remark"])
		domains = append(domains, model.Domain{
			Host:      host,
			Type:      domainType,
			Scheme:    scheme,
			Remark:    remark,
			Status:    defaultString(row["status"], model.StatusActive),
			CreatedBy: parseUint(row["created_by"], 1),
		})
	}
	return domains, nil
}

func readLinks(path string) ([]model.Link, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	links := make([]model.Link, 0, len(rows))
	for index, row := range rows {
		code := strings.TrimSpace(row["code"])
		if code == "" {
			return nil, fmt.Errorf("links row %d: code is required", index+2)
		}

		linkType := defaultString(row["type"], model.LinkTypeShort)
		targetURL := optionalCSVString(row["target_url"])
		title := optionalCSVString(row["title"])
		expireAt, err := parseOptionalTime(row["expire_at"])
		if err != nil {
			return nil, fmt.Errorf("links row %d: %w", index+2, err)
		}

		links = append(links, model.Link{
			Code:            code,
			Type:            linkType,
			EntryDomainID:   parseUint(row["entry_domain_id"], 1),
			TransitDomainID: optionalCSVUint(row["transit_domain_id"]),
			LandingDomainID: optionalCSVUint(row["landing_domain_id"]),
			TargetURL:       targetURL,
			LandingPageID:   optionalCSVUint(row["landing_page_id"]),
			Title:           title,
			ExpireAt:        expireAt,
			Status:          defaultString(row["status"], model.LinkStatusActive),
			CreatedBy:       parseUint(row["created_by"], 1),
		})
	}
	return links, nil
}

func readCSV(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("empty csv: %s", path)
	}

	headers := records[0]
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(headers))
		for index, header := range headers {
			if index < len(record) {
				row[strings.TrimSpace(header)] = strings.TrimSpace(record[index])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func optionalCSVString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func optionalCSVUint(value string) *uint64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseUint(value string, fallback uint64) uint64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid time %q", value)
}

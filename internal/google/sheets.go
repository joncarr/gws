package google

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	gsheets "google.golang.org/api/sheets/v4"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/config"
)

type SheetInfo struct {
	SpreadsheetID  string `json:"spreadsheet_id"`
	SpreadsheetURL string `json:"spreadsheet_url"`
	Title          string `json:"title"`
}

type SheetService interface {
	CreateSheet(ctx context.Context, profile config.Profile, title string, rows [][]string) (SheetInfo, error)
	ReadRows(ctx context.Context, profile config.Profile, spreadsheetID string, readRange string) ([][]string, error)
}

type AdminSheetsExporter struct{}

func (AdminSheetsExporter) CreateSheet(ctx context.Context, profile config.Profile, title string, rows [][]string) (SheetInfo, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return SheetInfo{}, err
	}
	svc, err := gsheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return SheetInfo{}, fmt.Errorf("create Google Sheets client: %w", err)
	}
	spreadsheet, err := svc.Spreadsheets.Create(&gsheets.Spreadsheet{
		Properties: &gsheets.SpreadsheetProperties{Title: title},
	}).Context(ctx).Do()
	if err != nil {
		return SheetInfo{}, fmt.Errorf("create spreadsheet: %w", err)
	}
	if len(rows) > 0 {
		values := make([][]any, 0, len(rows))
		for _, row := range rows {
			record := make([]any, 0, len(row))
			for _, cell := range row {
				record = append(record, cell)
			}
			values = append(values, record)
		}
		_, err = svc.Spreadsheets.Values.Update(
			spreadsheet.SpreadsheetId,
			"A1",
			&gsheets.ValueRange{Values: values},
		).Context(ctx).ValueInputOption("RAW").Do()
		if err != nil {
			return SheetInfo{}, fmt.Errorf("write spreadsheet values: %w", err)
		}
	}
	return SheetInfo{
		SpreadsheetID:  spreadsheet.SpreadsheetId,
		SpreadsheetURL: spreadsheet.SpreadsheetUrl,
		Title:          title,
	}, nil
}

func (AdminSheetsExporter) ReadRows(ctx context.Context, profile config.Profile, spreadsheetID string, readRange string) ([][]string, error) {
	httpClient, err := auth.HTTPClient(ctx, profile)
	if err != nil {
		return nil, err
	}
	svc, err := gsheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Google Sheets client: %w", err)
	}
	if readRange == "" {
		readRange = "A:Z"
	}
	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("read spreadsheet values: %w", err)
	}
	rows := make([][]string, 0, len(resp.Values))
	for _, row := range resp.Values {
		record := make([]string, 0, len(row))
		for _, cell := range row {
			record = append(record, fmt.Sprint(cell))
		}
		rows = append(rows, record)
	}
	return rows, nil
}

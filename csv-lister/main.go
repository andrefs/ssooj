package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type csvRow struct {
	Key         string
	Filename    string
	Store       string
	Date        string
	Total       string
	Size        int64
	LastMod     time.Time
	DownloadURL string
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, _ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	bucket := os.Getenv("PROCESSED_BUCKET")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return serverError(err)
	}

	client := s3.NewFromConfig(cfg)
	presigner := s3.NewPresignClient(client)

	var rows []csvRow
	var token *string
	for {
		list, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            &bucket,
			Prefix:            strPtr("csv/"),
			ContinuationToken: token,
		})
		if err != nil {
			return serverError(err)
		}

		for _, obj := range list.Contents {
			if obj.Key == nil {
				continue
			}
			row, err := describeCSV(ctx, client, presigner, bucket, obj)
			if err != nil {
				continue
			}
			rows = append(rows, row)
		}

		if list.IsTruncated != nil && *list.IsTruncated && list.NextContinuationToken != nil {
			token = list.NextContinuationToken
			continue
		}
		break
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].LastMod.After(rows[j].LastMod)
	})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		Body:       renderHTML(rows),
	}, nil
}

func describeCSV(ctx context.Context, client *s3.Client, presigner *s3.PresignClient, bucket string, obj types.Object) (csvRow, error) {
	row := csvRow{
		Key:      *obj.Key,
		Filename: strings.TrimSuffix(strings.TrimPrefix(*obj.Key, "csv/"), ".csv"),
	}
	if obj.Size != nil {
		row.Size = *obj.Size
	}
	if obj.LastModified != nil {
		row.LastMod = *obj.LastModified
	}

	res, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    obj.Key,
		Range:  strPtr("bytes=0-4096"),
	})
	if err == nil {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(res.Body); err == nil {
			if fields, ok := firstCSVRow(buf.String()); ok {
				row.Store = fields["store"]
				row.Date = fields["date"]
				row.Total = fields["total"]
			}
		}
		res.Body.Close()
	}

	presigned, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    obj.Key,
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return row, err
	}
	row.DownloadURL = presigned.URL

	return row, nil
}

func firstCSVRow(chunk string) (map[string]string, bool) {
	r := csv.NewReader(strings.NewReader(chunk))
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, false
	}
	record, err := r.Read()
	if err != nil {
		return nil, false
	}

	fields := make(map[string]string, len(header))
	for i, name := range header {
		name = strings.TrimSpace(name)
		if name == "" || i >= len(record) {
			continue
		}
		fields[name] = record[i]
	}
	return fields, true
}

func renderHTML(rows []csvRow) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>ssooj - Receipt CSVs</title>`)
	b.WriteString(`<style>body{font-family:system-ui,-apple-system,sans-serif;background:#f0f2f5;margin:0;padding:40px 20px;color:#1a1a2e}`)
	b.WriteString(`h1{font-size:22px;margin:0 0 20px}table{border-collapse:collapse;width:100%;max-width:960px;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,.06)}`)
	b.WriteString(`th,td{text-align:left;padding:10px 14px;border-bottom:1px solid #eee;font-size:14px}th{background:#f9fafb;color:#6b7280;font-weight:600}tr:last-child td{border-bottom:none}`)
	b.WriteString(`a{color:#2563eb;text-decoration:none}a:hover{text-decoration:underline}.empty{color:#6b7280}</style></head><body>`)
	b.WriteString(`<h1>Receipt CSVs</h1>`)

	if len(rows) == 0 {
		b.WriteString(`<p class="empty">No CSVs found.</p>`)
	} else {
		b.WriteString(`<table><thead><tr><th>File</th><th>Store</th><th>Date</th><th>Total</th><th>Size</th><th>Last modified</th><th></th></tr></thead><tbody>`)
		for _, r := range rows {
			b.WriteString("<tr>")
			b.WriteString("<td>" + html.EscapeString(r.Filename) + "</td>")
			b.WriteString("<td>" + html.EscapeString(r.Store) + "</td>")
			b.WriteString("<td>" + html.EscapeString(r.Date) + "</td>")
			b.WriteString("<td>" + html.EscapeString(r.Total) + "</td>")
			b.WriteString("<td>" + formatSize(r.Size) + "</td>")
			b.WriteString("<td>" + r.LastMod.Format("2006-01-02 15:04") + "</td>")
			b.WriteString(`<td><a href="` + html.EscapeString(r.DownloadURL) + `">Download</a></td>`)
			b.WriteString("</tr>")
		}
		b.WriteString("</tbody></table>")
	}

	b.WriteString("</body></html>")
	return b.String()
}

func formatSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return strconv.FormatInt(n, 10) + " B"
	}
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       err.Error(),
	}, nil
}

func strPtr(s string) *string { return &s }

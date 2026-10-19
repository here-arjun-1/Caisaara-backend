package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type puzzleRecord struct {
	ID              string
	FEN             string
	Moves           string
	Rating          int
	RatingDeviation int
	Popularity      int
	NbPlays         int
	Themes          []string
	GameURL         string
	OpeningTags     string
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		filePath      string
		dbURL         string
		minPopularity int
		minPlays      int
		minRating     int
		maxRating     int
		batchSize     int
		limit         int
	)

	flag.StringVar(&filePath, "file", "", "path to lichess_db_puzzle.csv or .csv.gz")
	flag.StringVar(&dbURL, "db", "", "database URL (defaults to dbURL or DATABASE_URL in .env)")
	flag.IntVar(&minPopularity, "min-popularity", 80, "minimum popularity score (-100 to 100)")
	flag.IntVar(&minPlays, "min-plays", 100, "minimum plays count")
	flag.IntVar(&minRating, "min-rating", 0, "minimum puzzle rating")
	flag.IntVar(&maxRating, "max-rating", 3500, "maximum puzzle rating")
	flag.IntVar(&batchSize, "batch-size", 1000, "batch size for insertion (1 to 5000)")
	flag.IntVar(&limit, "limit", 0, "maximum puzzles to seed (0 = unlimited)")
	flag.Parse()

	if filePath == "" {
		flag.Usage()
		return errors.New("-file argument is required")
	}

	_ = godotenv.Load()

	if dbURL == "" {
		dbURL = os.Getenv("dbURL")
	}
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return errors.New("database URL is required via -db flag or dbURL/DATABASE_URL environment variable")
	}

	if batchSize < 1 || batchSize > 5000 {
		batchSize = 1000
	}

	pool, err := database.ConnectDB(dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var r io.Reader = file
	if strings.HasSuffix(filePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("open gzip reader: %w", err)
		}
		defer gzReader.Close()
		r = gzReader
	}

	csvReader := csv.NewReader(bufio.NewReader(r))
	csvReader.FieldsPerRecord = -1
	csvReader.ReuseRecord = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	batch := make([]puzzleRecord, 0, batchSize)
	totalLines := 0
	insertedCount := 0
	startTime := time.Now()

	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			continue
		}

		totalLines++

		if len(record) < 8 {
			continue
		}

		puzzleID := strings.TrimSpace(record[0])
		if puzzleID == "" || strings.EqualFold(puzzleID, "puzzleid") {
			continue
		}

		fen := strings.TrimSpace(record[1])
		moves := strings.TrimSpace(record[2])

		rating, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			continue
		}

		rd, _ := strconv.Atoi(strings.TrimSpace(record[4]))
		popularity, _ := strconv.Atoi(strings.TrimSpace(record[5]))
		nbPlays, _ := strconv.Atoi(strings.TrimSpace(record[6]))

		if popularity < minPopularity || nbPlays < minPlays {
			continue
		}
		if rating < minRating || (maxRating > 0 && rating > maxRating) {
			continue
		}

		rawThemes := strings.TrimSpace(record[7])
		var themes []string
		if rawThemes != "" {
			themes = strings.Fields(rawThemes)
		} else {
			themes = []string{}
		}

		var gameURL string
		if len(record) > 8 {
			gameURL = strings.TrimSpace(record[8])
		}

		var openingTags string
		if len(record) > 9 {
			openingTags = strings.TrimSpace(record[9])
		}

		batch = append(batch, puzzleRecord{
			ID:              puzzleID,
			FEN:             fen,
			Moves:           moves,
			Rating:          rating,
			RatingDeviation: rd,
			Popularity:      popularity,
			NbPlays:         nbPlays,
			Themes:          themes,
			GameURL:         gameURL,
			OpeningTags:     openingTags,
		})

		if len(batch) >= batchSize {
			if err := insertBatch(ctx, pool, batch); err != nil {
				return fmt.Errorf("insert batch: %w", err)
			}
			insertedCount += len(batch)
			batch = batch[:0]

			if limit > 0 && insertedCount >= limit {
				break
			}
		}

		if totalLines%50000 == 0 {
			slog.Info("seeding in progress",
				"lines_read", totalLines,
				"puzzles_seeded", insertedCount,
				"elapsed", time.Since(startTime).Truncate(time.Second).String(),
			)
		}
	}

	if len(batch) > 0 && (limit <= 0 || insertedCount < limit) {
		if err := insertBatch(ctx, pool, batch); err != nil {
			return fmt.Errorf("insert final batch: %w", err)
		}
		insertedCount += len(batch)
	}

	slog.Info("seeding completed",
		"total_lines_read", totalLines,
		"total_puzzles_seeded", insertedCount,
		"duration", time.Since(startTime).Truncate(time.Second).String(),
	)

	return nil
}

func insertBatch(ctx context.Context, pool *pgxpool.Pool, batch []puzzleRecord) error {
	if len(batch) == 0 {
		return nil
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO puzzles (id, fen, moves, rating, rating_deviation, popularity, nb_plays, themes, game_url, opening_tags) VALUES ")

	args := make([]any, 0, len(batch)*10)
	for i, p := range batch {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		offset := i * 10
		fmt.Fprintf(&queryBuilder, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5,
			offset+6, offset+7, offset+8, offset+9, offset+10)

		args = append(args, p.ID, p.FEN, p.Moves, p.Rating, p.RatingDeviation, p.Popularity, p.NbPlays, p.Themes, p.GameURL, p.OpeningTags)
	}

	queryBuilder.WriteString(" ON CONFLICT (id) DO NOTHING")

	_, err := pool.Exec(ctx, queryBuilder.String(), args...)
	return err
}

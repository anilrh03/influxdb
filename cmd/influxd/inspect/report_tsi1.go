package inspect

import (
	"errors"
	"io"
	"os"
	"runtime"

	"github.com/influxdata/influxdb"

	"github.com/influxdata/influxdb/logger"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxdb/tsdb/tsi1"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

// Command represents the program execution for "influxd reporttsi".
var tsiFlags = struct {
	// Standard input/output, overridden for testing.
	Stderr io.Writer
	Stdout io.Writer

	Path   string
	org    string
	bucket string

	seriesFilePath string // optional. Defaults to dbPath/_series
	sfile          *tsdb.SeriesFile

	topN          int
	byMeasurement bool
	byTagKey      bool

	// How many goroutines to dedicate to calculating cardinality.
	concurrency int
}{}

// NewReportTsiCommand returns a new instance of Command with default setting applied.
func NewReportTsiCommand() *cobra.Command {
	reportTsiCommand := &cobra.Command{
		Use:   "report-tsi",
		Short: "Reports the cardinality of tsi files short",
		Long:  `Reports the cardinality of tsi files long.`,
		RunE:  RunReportTsi,
	}
	reportTsiCommand.Flags().StringVar(&tsiFlags.Path, "path", os.Getenv("HOME")+"/.influxdbv2/engine", "Path to data engine. Defaults $HOME/.influxdbv2/engine")
	reportTsiCommand.Flags().StringVar(&tsiFlags.seriesFilePath, "series-file", "", "Optional path to series file. Defaults /path/to/db-path/_series")
	reportTsiCommand.Flags().BoolVar(&tsiFlags.byMeasurement, "measurements", true, "Segment cardinality by measurements")
	// TODO(edd): Not yet implemented.
	// fs.BoolVar(&cmd.byTagKey, "tag-key", false, "Segment cardinality by tag keys (overrides `measurements`")
	reportTsiCommand.Flags().IntVar(&tsiFlags.topN, "top", 0, "Limit results to top n")
	reportTsiCommand.Flags().IntVar(&tsiFlags.concurrency, "c", runtime.GOMAXPROCS(0), "Set worker concurrency. Defaults to GOMAXPROCS setting.")
	reportTsiCommand.Flags().StringVar(&tsiFlags.bucket, "bucket", "", "If bucket is specified, org must be specified")
	reportTsiCommand.Flags().StringVar(&tsiFlags.org, "org", "", "org to be searched")

	reportTsiCommand.SetOutput(tsiFlags.Stdout)

	return reportTsiCommand
}

// RunReportTsi executes the run command for ReportTsi.
func RunReportTsi(cmd *cobra.Command, args []string) error {
	// set up log
	config := logger.NewConfig()
	config.Level = zapcore.InfoLevel
	log, err := config.New(os.Stderr)
	// do some filepath walking, we are looking for index files
	//dir := os.Getenv("HOME") + "/.influxdbv2/engine/index"

	// if path is unset, set to os.Getenv("HOME") + "/.influxdbv2/engine"
	if tsiFlags.Path == "" {
		tsiFlags.Path = os.Getenv("HOME") + "/.influxdbv2/engine"
	}

	report := tsi1.NewReportCommand()
	report.Concurrency = tsiFlags.concurrency
	report.DataPath = tsiFlags.Path
	report.Logger = log

	if tsiFlags.org != "" {
		if orgID, err := influxdb.IDFromString(tsiFlags.org); err != nil {
			return err
		} else {
			report.OrgID = orgID
		}
	}

	if tsiFlags.bucket != "" {
		if bucketID, err := influxdb.IDFromString(tsiFlags.bucket); err != nil {
			return err
		} else if report.OrgID == nil {
			return errors.New("org must be provided if filtering by bucket ")
		} else {
			report.BucketID = bucketID
		}
	}

	report.Logger.Error("running report")
	err = report.Run()
	if err != nil {
		return err
	}
	return nil
}
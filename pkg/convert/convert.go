// Package convert transforms a GeoIP2/GeoLite2 CSV to various formats.
package convert

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/fholzer/csv2mmdb/pkg/convert/internal/valuecache"
	"github.com/maxmind/mmdbwriter/mmdbtype"

	"github.com/maxmind/mmdbwriter"
	"github.com/pkg/errors"
	progressbar "github.com/schollz/progressbar/v3"
)

// ConvertFile converts the MaxMind GeoIP2 or GeoLite2 CSV file `inputFile` to
// `outputFile` file using a different representation of the network. The
// representation can be specified by setting one or more of `cidr`,
// `ipRange`, `intRange` or `hexRange` to true. If none of these are set to true, it will
// strip off the network information.
func ConvertFile( //nolint: revive // stutters, should fix
	config *Config,
	inputFile string,
	outputFile string,
) error {
	outFile, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		return errors.Wrapf(err, "error creating output file (%s)", outputFile)
	}
	defer outFile.Close() //nolint: gosec

	inFile, err := os.Open(inputFile) //nolint: gosec
	if err != nil {
		return errors.Wrapf(err, "error opening input file (%s)", inputFile)
	}
	defer inFile.Close() //nolint: gosec

	inFileInfo, err := inFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "error retrieving input file stats (%s)", inputFile)
	}

	converter := NewConverter(config, inFile, inFileInfo.Size())
	err = converter.Convert(outFile)
	if err != nil {
		return err
	}
	err = outFile.Sync()
	if err != nil {
		return errors.Wrapf(err, "error syncing file (%s)", outputFile)
	}
	return nil
}

const (
	STR_START_IP string = "start_ip_int"
	STR_END_IP   string = "end_ip_int"
)

type Converter struct {
	config    *Config
	rowMapper *RowMapper
	mapCache  *valuecache.DataMap
	input     io.Reader
	inputSize int64
}

func NewConverter(config *Config, input io.Reader, inputSize int64) *Converter {
	return &Converter{
		config:    config,
		input:     input,
		inputSize: inputSize,
		mapCache:  valuecache.NewDataMap(),
	}
}

// Convert writes the MaxMind GeoIP2 or GeoLite2 CSV in the `input` io.Reader
// to the Writer `output`.
func (c *Converter) Convert(
	output io.Writer,
) error {
	tree, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            c.config.DatabaseType,
		IncludeReservedNetworks: true,
		IPVersion:               4,
		DisableMetadataPointers: true,
	})
	if err != nil {
		return errors.Wrap(err, "error creating new mmdb tree")
	}

	bar := progressbar.DefaultBytes(c.inputSize)
	defer bar.Close()
	bar.Clear()
	pbReader := progressbar.NewReader(c.input, bar)
	reader := csv.NewReader(&pbReader)

	header, err := reader.Read()
	if err != nil {
		return errors.Wrap(err, "error reading CSV header")
	}

	rowMapper, err := NewMapper(c.config, header)
	if err != nil {
		return errors.Wrap(err, "error creating row mapper")
	}
	c.rowMapper = rowMapper
	bar.Clear()

	// This holds the previously read, but not yet written row. We hold it instead of writing it immediately,
	// because we might be able to merge it with subsequent rows, if all the selected/filtered-for fields are
	// equal.
	// var lastReadRecord []string
	var row int = 0
	for {
		data, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return errors.Wrap(err, "error reading CSV")
		}
		row++

		// if lastReadRecord != nil {
		// 	// compare this to previous row
		// 	shouldMerge := rowMapper.CanMergeRows(lastReadRecord, data)

		// 	if shouldMerge {
		// 		// set previous row's end_ip to current row's end_ip
		// 		lastReadRecord[1] = strings.Clone(data[1])
		// 		continue
		// 	}

		// 	if err := c.insert(tree, lastReadRecord); err != nil {
		// 		return errors.Wrapf(err, "error writing output record (at input row %d)", row)
		// 	}
		// }
		if err := c.insert(tree, data); err != nil {
			return errors.Wrapf(err, "error writing output record (at input row %d)", row)
		}
		// lastReadRecord = data
	}

	// if lastReadRecord != nil {
	// 	if err := c.insert(tree, lastReadRecord); err != nil {
	// 		return errors.Wrapf(err, "error writing output record (at input row %d)", row)
	// 	}
	// }

	PrintMemUsage()
	log.Println("Writing mmdb tree data...")
	_, err = tree.WriteTo(output)
	if err == nil {
		log.Println("done writing")
	}

	return errors.Wrap(err, "error writing CSV")
}

func PrintMemUsage() {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func (c *Converter) insert(tree *mmdbwriter.Tree, data []string) error {
	iStart, err := strconv.ParseUint(data[0], 10, 32)
	if err != nil {
		return errors.Wrapf(err, "Error converting start IP to int: %s\n", data[0])
	}

	iEnd, err := strconv.ParseUint(data[1], 10, 32)
	if err != nil {
		return errors.Wrapf(err, "Error converting end IP to int: %s\n", data[1])
	}

	r, err := c.rowMapper.Map(data)
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}

	if c.config.UseValueCache {
		cv, err := c.mapCache.Store(r)
		if err != nil {
			return errors.Wrapf(err, "Error accessing value cache")
		}
		r = cv.Data.(mmdbtype.Map)
	}

	start := int2ip(uint32(iStart))
	end := int2ip(uint32(iEnd))

	tree.InsertRange(start, end, r)
	return nil
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

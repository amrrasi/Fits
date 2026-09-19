package fits

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	fitsio "github.com/astrogo/fitsio"

	"github.com/amrrasi/fits/internal/logger"
	"github.com/amrrasi/fits/internal/models"
)

type ParseResult struct {
	File     models.FITSFile
	Headers  []models.FITSHeader
	Metadata models.FITSMetadata
}

const (
	maxHeadersPerFile = 200_000
	maxTextLen        = 4096
)

// clean makes header text safe for PostgreSQL (no NUL bytes, valid UTF-8, bounded length).
func clean(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ToValidUTF8(s, "?")
	if len(s) > maxTextLen {
		s = strings.ToValidUTF8(s[:maxTextLen], "")
	}
	return s
}

// ParseFile parses a FITS file. A malformed/hostile file can never crash the process:
// panics from the underlying library are converted into errors.
func ParseFile(path string) (res *ParseResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("فایل FITS معتبر نیست (خطای پردازش): %v", r)
		}
	}()
	return parseFile(path)
}

func parseFile(path string) (*ParseResult, error) {
	log := logger.S().With("file", path)
	log.Debug("fits: opening file")

	// ── File metadata ──────────────────────────────────────────────────────────
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("fits: stat %q: %w", path, err)
	}

	checksum, err := sha256File(path)
	if err != nil {
		return nil, fmt.Errorf("fits: checksum %q: %w", path, err)
	}

	// ── Open FITS ──────────────────────────────────────────────────────────────
	// fitsio.Open requires an io.Reader, not a path — open the OS file first.
	osFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fits: open %q: %w", path, err)
	}
	defer osFile.Close()

	f, err := fitsio.Open(osFile)
	if err != nil {
		return nil, fmt.Errorf("fits: open %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Warnw("fits: close error", "err", cerr)
		}
	}()

	hdus := f.HDUs()

	result := &ParseResult{}
	result.File = models.FITSFile{
		FilePath: path,
		FileName: filepath.Base(path),
		FileSize: info.Size(),
		Checksum: checksum,
		HDUCount: len(hdus),
		Status:   models.FileStatusProcessing,
	}
	result.Metadata.FileID = 0 // will be filled in after DB insert

	// ── Process each HDU ──────────────────────────────────────────────────────
	for hduIdx, hdu := range hdus {
		hduName := hduName(hdu, hduIdx)
		log.Debugw("fits: processing HDU", "index", hduIdx, "name", hduName)

		hdr := hdu.Header()
		keys := hdr.Keys()
		for k := range keys {
			card := hdr.Card(k)
			keyword := strings.TrimSpace(card.Name)
			if len(result.Headers) >= maxHeadersPerFile {
				return nil, fmt.Errorf("تعداد هدرهای فایل بیش از حد مجاز است (%d)", maxHeadersPerFile)
			}
			keyword = clean(keyword)

			rawValue := clean(fmt.Sprintf("%v", card.Value))
			valType := inferType(card.Value)

			header := models.FITSHeader{
				// FileID filled in by repository after file insert
				HDUIndex:  hduIdx,
				HDUName:   hduName,
				Keyword:   keyword,
				Value:     rawValue,
				Comment:   clean(card.Comment),
				ValueType: valType,
			}
			result.Headers = append(result.Headers, header)

			// Extract well-known keywords into typed metadata (primary HDU only)
			if hduIdx == 0 {
				populateMetadata(&result.Metadata, keyword, card.Value)
			}
		}
	}

	log.Infow("fits: parsed",
		"hdus", len(hdus),
		"headers", len(result.Headers),
	)
	return result, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func hduName(hdu fitsio.HDU, idx int) string {
	if idx == 0 {
		return "PRIMARY"
	}
	// Try EXTNAME keyword
	hdr := hdu.Header()
	keys := hdr.Keys()
	for k := range keys {
		c := hdr.Card(k)
		if strings.TrimSpace(c.Name) == "EXTNAME" {
			if s, ok := c.Value.(string); ok && s != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return fmt.Sprintf("HDU%d", idx)
}

// sha256File computes a hex-encoded SHA-256 checksum of the file at path.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// inferType returns a string label for the Go type of a FITS card value.
func inferType(v interface{}) string {
	if v == nil {
		return "undefined"
	}
	switch v.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "int"
	case float32, float64:
		return "float"
	case complex64, complex128:
		return "complex"
	case string:
		return "string"
	default:
		return "string"
	}
}

// populateMetadata extracts a known keyword value into the FITSMetadata struct.
// Add new keywords here as the schema grows — no other file needs changing.
func populateMetadata(m *models.FITSMetadata, keyword string, value interface{}) {
	kw := strings.ToUpper(strings.TrimSpace(keyword))

	switch kw {
	// Image geometry
	case "NAXIS":
		if v := toInt(value); v != nil {
			m.NAXIS = v
		}
	case "NAXIS1":
		if v := toInt(value); v != nil {
			m.NAXIS1 = v
		}
	case "NAXIS2":
		if v := toInt(value); v != nil {
			m.NAXIS2 = v
		}
	case "NAXIS3":
		if v := toInt(value); v != nil {
			m.NAXIS3 = v
		}
	case "BITPIX":
		if v := toInt(value); v != nil {
			m.BITPIX = v
		}

	// Exposure
	case "EXPTIME", "EXPOSURE":
		if v := toFloat(value); v != nil {
			m.ExpTime = v
		}
	case "GAIN":
		if v := toFloat(value); v != nil {
			m.Gain = v
		}
	case "RDNOISE", "READNOIS":
		if v := toFloat(value); v != nil {
			m.ReadNoise = v
		}

	// Date / time
	case "DATE-OBS":
		if v := toString(value); v != nil {
			m.DateObs = v
		}
	case "TIME-OBS":
		if v := toString(value); v != nil {
			m.TimeObs = v
		}
	case "MJD-OBS":
		if v := toFloat(value); v != nil {
			m.MJDObs = v
		}
	case "MJD-END":
		if v := toFloat(value); v != nil {
			m.MJDEnd = v
		}

	// Instrument / camera
	case "INSTRUME":
		if v := toString(value); v != nil {
			m.Instrume = v
		}
	case "TELESCOP":
		if v := toString(value); v != nil {
			m.Telescop = v
		}
	case "OBSERVER":
		if v := toString(value); v != nil {
			m.Observer = v
		}
	case "OBJECT":
		if v := toString(value); v != nil {
			m.Object = v
		}
	case "ORIGIN":
		if v := toString(value); v != nil {
			m.Origin = v
		}
	case "SOFTWARE", "SWCREATE":
		if v := toString(value); v != nil {
			m.SoftwareVersion = v
		}
	case "EQUIPID", "SERIALNO":
		if v := toString(value); v != nil {
			m.EquipID = v
		}

	// Temperature
	case "SET-TEMP", "SETTEMP":
		if v := toFloat(value); v != nil {
			m.SetTemp = v
		}
	case "CCD-TEMP", "CCDTEMP":
		if v := toFloat(value); v != nil {
			m.CCDTemp = v
		}
	case "AMB-TEMP", "AMBTEMP":
		if v := toFloat(value); v != nil {
			m.AmbTemp = v
		}

	// Filter
	case "FILTER":
		if v := toString(value); v != nil {
			m.Filter = v
		}
	case "FILTERID", "FILTER_ID":
		if v := toString(value); v != nil {
			m.FilterID = v
		}

	// Site location
	case "SITELAT", "OBSLAT", "LATITUDE":
		if v := toFloat(value); v != nil {
			m.SiteLatitude = v
		}
	case "SITELONG", "OBSLON", "LONGITUD":
		if v := toFloat(value); v != nil {
			m.SiteLongitude = v
		}
	case "SITEELEV", "OBSELEV", "ELEVATIO":
		if v := toFloat(value); v != nil {
			m.SiteElevation = v
		}

	// Target / astrometry
	case "RA", "OBJCTRA", "RA_OBJ":
		if v := toFloat(value); v != nil {
			m.RA = v
		}
	case "DEC", "OBJCTDEC", "DEC_OBJ":
		if v := toFloat(value); v != nil {
			m.Dec = v
		}
	case "AIRMASS":
		if v := toFloat(value); v != nil {
			m.Airmass = v
		}

	// WCS
	case "CRVAL1":
		if v := toFloat(value); v != nil {
			m.CRVAL1 = v
		}
	case "CRVAL2":
		if v := toFloat(value); v != nil {
			m.CRVAL2 = v
		}
	case "CRPIX1":
		if v := toFloat(value); v != nil {
			m.CRPIX1 = v
		}
	case "CRPIX2":
		if v := toFloat(value); v != nil {
			m.CRPIX2 = v
		}
	case "CD1_1":
		if v := toFloat(value); v != nil {
			m.CD1_1 = v
		}
	case "CD1_2":
		if v := toFloat(value); v != nil {
			m.CD1_2 = v
		}
	case "CD2_1":
		if v := toFloat(value); v != nil {
			m.CD2_1 = v
		}
	case "CD2_2":
		if v := toFloat(value); v != nil {
			m.CD2_2 = v
		}
	case "CTYPE1":
		if v := toString(value); v != nil {
			m.CTYPE1 = v
		}
	case "CTYPE2":
		if v := toString(value); v != nil {
			m.CTYPE2 = v
		}

	// Scaling
	case "BSCALE":
		if v := toFloat(value); v != nil {
			m.BScale = v
		}
	case "BZERO":
		if v := toFloat(value); v != nil {
			m.BZero = v
		}
	}
}

// ── Type coercion helpers ─────────────────────────────────────────────────────

func toFloat(v interface{}) *float64 {
	switch val := v.(type) {
	case float64:
		return &val
	case float32:
		f := float64(val)
		return &f
	case int:
		f := float64(val)
		return &f
	case int64:
		f := float64(val)
		return &f
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			return &f
		}
	}
	return nil
}

func toInt(v interface{}) *int {
	switch val := v.(type) {
	case int:
		return &val
	case int64:
		i := int(val)
		return &i
	case float64:
		i := int(val)
		return &i
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return &i
		}
	}
	return nil
}

func toString(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprintf("%v", v)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

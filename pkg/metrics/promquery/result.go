package promquery

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ResultType enumerates the Prometheus result types this client decodes.
type ResultType string

const (
	ResultVector ResultType = "vector"
	ResultMatrix ResultType = "matrix"
	ResultScalar ResultType = "scalar"
)

// Point is a single (timestamp, value) sample.
type Point struct {
	T time.Time
	V float64
}

// Sample is one series with a single value (vector result).
type Sample struct {
	Metric map[string]string
	Point  Point
}

// Series is one series with a range of values (matrix result).
type Series struct {
	Metric map[string]string
	Points []Point
}

// Result holds a decoded query response. Exactly one of Vector/Matrix/Scalar is
// populated depending on Type.
type Result struct {
	Type   ResultType
	Vector []Sample
	Matrix []Series
	Scalar *Point
}

func decodeData(resultType string, raw json.RawMessage) (*Result, error) {
	r := &Result{Type: ResultType(resultType)}
	switch r.Type {
	case ResultVector:
		var samples []struct {
			Metric map[string]string `json:"metric"`
			Value  rawPoint          `json:"value"`
		}
		if err := json.Unmarshal(raw, &samples); err != nil {
			return nil, fmt.Errorf("decoding vector result: %w", err)
		}
		for _, s := range samples {
			r.Vector = append(r.Vector, Sample{Metric: s.Metric, Point: s.Value.point})
		}
	case ResultMatrix:
		var series []struct {
			Metric map[string]string `json:"metric"`
			Values []rawPoint        `json:"values"`
		}
		if err := json.Unmarshal(raw, &series); err != nil {
			return nil, fmt.Errorf("decoding matrix result: %w", err)
		}
		for _, s := range series {
			pts := make([]Point, 0, len(s.Values))
			for _, v := range s.Values {
				pts = append(pts, v.point)
			}
			r.Matrix = append(r.Matrix, Series{Metric: s.Metric, Points: pts})
		}
	case ResultScalar:
		var v rawPoint
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("decoding scalar result: %w", err)
		}
		p := v.point
		r.Scalar = &p
	default:
		return nil, fmt.Errorf("unsupported prometheus result type %q", resultType)
	}
	return r, nil
}

// rawPoint decodes a Prometheus [<unix_ts float>, "<value string>"] tuple.
type rawPoint struct {
	point Point
}

func (rp *rawPoint) UnmarshalJSON(b []byte) error {
	var tuple []json.RawMessage
	if err := json.Unmarshal(b, &tuple); err != nil {
		return err
	}
	if len(tuple) != 2 {
		return fmt.Errorf("expected [ts, value] pair, got %d elements", len(tuple))
	}
	var ts float64
	if err := json.Unmarshal(tuple[0], &ts); err != nil {
		return fmt.Errorf("decoding sample timestamp: %w", err)
	}
	var valStr string
	if err := json.Unmarshal(tuple[1], &valStr); err != nil {
		return fmt.Errorf("decoding sample value: %w", err)
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return fmt.Errorf("parsing sample value %q: %w", valStr, err)
	}
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	rp.point = Point{T: time.Unix(sec, nsec).UTC(), V: val}
	return nil
}

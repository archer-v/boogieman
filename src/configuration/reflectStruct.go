package configuration

// helper methods to set a struct field defined by string name to
// a new value. reflection is used to walk all struct field and
// set the value according an actual field type

// some pieces of this code was borrowed from github.com/vrischmann/envconfig/envconfig.go

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	durationType  = reflect.TypeOf((*time.Duration)(nil)).Elem()
	byteSliceType = reflect.TypeOf([]byte(nil))
)

const (
	sliceSeparator rune = ','
)

func isDurationField(t reflect.Type) bool {
	return t.AssignableTo(durationType)
}

func setStructField(structPtr any, fName string, fValue string) (err error) {
	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr {
		return errors.New("not a pointer")
	}
	v = v.Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if strings.EqualFold(field.Name, fName) {
			if e := setStructFieldToValue(v.Field(i), fValue); e != nil {
				return fmt.Errorf("can't set value to %v field: %w", fName, e)
			}
			return nil
		}
	}
	return
}

func setStructFieldToValue(field reflect.Value, str string) error {
	switch {
	case field.Type() == byteSliceType:
		err := parseBytesValue(field, str)
		if err != nil {
			err = fmt.Errorf("unable to parse value %q as bytes. err=%v", str, err)
		}
		return err

	case field.Kind() == reflect.Slice:
		return setSliceField(field, str)

	default:
		return parseValue(field, str)
	}
}

func setSliceField(value reflect.Value, str string) error {
	separator := sliceSeparator

	elType := value.Type().Elem()
	tnz := newSliceTokenizer(str, separator)

	slice := reflect.MakeSlice(value.Type(), 0, 0)

	for tnz.scan() {
		token := tnz.text()

		el := reflect.New(elType).Elem()

		if err := parseValue(el, token); err != nil {
			return err
		}

		slice = reflect.Append(slice, el)
	}

	value.Set(slice)

	return tnz.Err()
}

func parseValue(v reflect.Value, str string) (err error) {
	vtype := v.Type()

	// Special case when the type is a map: we need to make the map
	if vtype.Kind() == reflect.Map {
		v.Set(reflect.MakeMap(vtype))
	}

	kind := vtype.Kind()
	switch {
	case isDurationField(vtype):
		// Special case for time.Duration
		err = parseDuration(v, str)
	case kind == reflect.Bool:
		err = parseBoolValue(v, str)
	case kind == reflect.Int, kind == reflect.Int8, kind == reflect.Int16, kind == reflect.Int32, kind == reflect.Int64:
		err = parseIntValue(v, str)
	case kind == reflect.Uint, kind == reflect.Uint8, kind == reflect.Uint16, kind == reflect.Uint32, kind == reflect.Uint64:
		err = parseUintValue(v, str)
	case kind == reflect.Float32, kind == reflect.Float64:
		err = parseFloatValue(v, str)
	case kind == reflect.Ptr:
		v.Set(reflect.New(vtype.Elem()))
		return parseValue(v.Elem(), str)
	case kind == reflect.String:
		v.SetString(str)
	case kind == reflect.Struct:
		err = parseStruct(v, str)
	default:
		return fmt.Errorf("kind %v not supported", kind)
	}

	if err != nil {
		return fmt.Errorf("unable to parse value %q: %w", str, err)
	}

	return
}

func parseDuration(v reflect.Value, str string) error {
	d, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	v.SetInt(int64(d))

	return nil
}

// this is only called when parsing structs inside a slice.
func parseStruct(value reflect.Value, token string) error {
	separator := string(sliceSeparator)

	tokens := strings.Split(token[1:len(token)-1], separator)
	if len(tokens) != value.NumField() {
		return fmt.Errorf("struct token has %d fields but struct has %d", len(tokens), value.NumField())
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		t := tokens[i]

		if err := parseValue(field, t); err != nil {
			return err
		}
	}

	return nil
}

func parseBoolValue(v reflect.Value, str string) error {
	val, err := strconv.ParseBool(str)
	if err != nil {
		return err
	}
	v.SetBool(val)

	return nil
}

func parseIntValue(v reflect.Value, str string) error {
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}
	v.SetInt(val)

	return nil
}

func parseUintValue(v reflect.Value, str string) error {
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	v.SetUint(val)

	return nil
}

func parseFloatValue(v reflect.Value, str string) error {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	v.SetFloat(val)

	return nil
}

func parseBytesValue(v reflect.Value, str string) error {
	val, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	v.SetBytes(val)

	return nil
}

type sliceTokenizer struct {
	err          error
	r            *bufio.Reader
	separator    rune
	buf          bytes.Buffer
	inBraces     bool
	unparsedRune rune
}

var eof = rune(0)

func newSliceTokenizer(str string, separator rune) *sliceTokenizer {
	return &sliceTokenizer{
		r:         bufio.NewReader(strings.NewReader(str)),
		separator: separator,
	}
}

func (t *sliceTokenizer) scan() bool {
	sepFound := false
	// rune was read at previous call but wasn't written
	if t.unparsedRune != 0 {
		_, _ = t.buf.WriteRune(t.unparsedRune)
		t.unparsedRune = 0
	}
	for {
		if t.err == io.EOF && t.buf.Len() == 0 {
			return false
		}

		ch := t.readRune()
		if ch == eof {
			return true
		}

		if sepFound {
			if ch == ' ' {
				// skip whitespaces after comma
				continue
			} else {
				t.unparsedRune = ch
				return true
			}
		}
		if ch == '{' {
			t.inBraces = true
		}
		if ch == '}' {
			t.inBraces = false
		}

		if ch == t.separator && !t.inBraces {
			sepFound = true
			continue
		}

		// NOTE(vincent): we ignore the WriteRune error here because there is NO WAY
		// for WriteRune to return an error.
		// Yep. Seriously. Look here http://golang.org/src/bytes/buffer.go?s=7661:7714#L227
		_, _ = t.buf.WriteRune(ch)
	}
}

func (t *sliceTokenizer) readRune() rune {
	ch, _, err := t.r.ReadRune()
	if err != nil {
		t.err = err
		return eof
	}
	return ch
}

func (t *sliceTokenizer) text() string {
	str := t.buf.String()
	t.buf.Reset()

	return str
}

func (t *sliceTokenizer) Err() error {
	if t.err == io.EOF {
		return nil
	}
	return t.err
}

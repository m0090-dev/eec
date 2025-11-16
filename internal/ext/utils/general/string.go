package general
import (
	"bytes"
	"encoding/binary"
)


// ---------------------------
// ヘルパー関数
// ---------------------------
func WriteString(buf *bytes.Buffer, s string) error {
	length := int32(len(s))
	if err := binary.Write(buf, binary.LittleEndian, length); err != nil {
		return err
	}
	return binary.Write(buf, binary.LittleEndian, []byte(s))
}

func WriteStringSlice(buf *bytes.Buffer, ss []string) error {
	count := int32(len(ss))
	if err := binary.Write(buf, binary.LittleEndian, count); err != nil {
		return err
	}
	for _, s := range ss {
		if err := WriteString(buf, s); err != nil {
			return err
		}
	}
	return nil
}

func ReadString(buf *bytes.Reader) (string, error) {
	var length int32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	bytes := make([]byte, length)
	if err := binary.Read(buf, binary.LittleEndian, &bytes); err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ReadStringSlice(buf *bytes.Reader) ([]string, error) {
	var count int32
	if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	result := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		s, err := ReadString(buf)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}


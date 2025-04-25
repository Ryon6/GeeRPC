package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
)

type Person struct {
	Name    string
	Age     int
	Hobbies []string
	Details map[string]interface{}
}

type Writer struct {
	Arr []byte
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.Arr = p
	return len(p), nil
}

func main() {
	// JSON 编码示例
	person := Person{
		Name:    "Alice",
		Age:     30,
		Hobbies: []string{"reading", "coding"},
		Details: map[string]interface{}{
			"married": true,
			"score":   95.5,
		},
	}

	log.Println("\n=== JSON 编码 ===")
	w := &Writer{}
	jsonEnc := json.NewEncoder(w)
	err := jsonEnc.Encode(person)
	if err != nil {
		log.Fatal("JSON 编码错误:", err)
	}
	log.Printf("JSON 编码结果: %s\n", w.Arr)

	// JSON 解码示例
	var decodedPerson Person
	err = json.Unmarshal(w.Arr, &decodedPerson)
	if err != nil {
		log.Fatal("JSON 解码错误:", err)
	}
	log.Printf("JSON 解码结果: %+v\n", decodedPerson)

	// Gob 编码示例
	log.Println("\n=== Gob 编码 ===")
	var buf bytes.Buffer
	gobEnc := gob.NewEncoder(&buf)
	err = gobEnc.Encode(person)
	if err != nil {
		log.Fatal("Gob 编码错误:", err)
	}
	gobData := buf.Bytes()
	log.Printf("Gob 编码结果(十六进制): %x\n", gobData)
	log.Printf("Gob 编码结果(长度): %d bytes\n", len(gobData))

	// Gob 解码示例
	var gobDecoded Person
	buf = *bytes.NewBuffer(gobData)
	gobDec := gob.NewDecoder(&buf)
	err = gobDec.Decode(&gobDecoded)
	if err != nil {
		log.Fatal("Gob 解码错误:", err)
	}
	log.Printf("Gob 解码结果: %+v\n", gobDecoded)
}

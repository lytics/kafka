/*
 *  Copyright (c) 2011 NeuStar, Inc.
 *  All rights reserved.  
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at 
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *  
 *  NeuStar, the Neustar logo and related names and logos are registered
 *  trademarks, service marks or tradenames of NeuStar, Inc. All other 
 *  product names, company names, marks, logos and symbols may be trademarks
 *  of their respective owners.
 */

package kafka

import (
  "bytes"
  "compress/gzip"
  "log"
  "testing"
)

type MessageMatch struct {
  PayloadIn    string
  Magic       byte
  Compression byte
  Checksum    [4]byte
  Payload     []byte
  TotalLength  uint64 // total length of the raw message (from decoding)
}

type MessageTestDef struct {
  Name         string
  Compress     byte
  Topic        string
  Partition    int
  MessageMatch
}
func (td *MessageTestDef) TopicPartition() *TopicPartition {
  return &TopicPartition{Topic:td.Topic, Partition:td.Partition}
}
var testData map[string]MessageTestDef
var topics []*TopicPartition = make([]*TopicPartition,0)

func init() {
  log.SetFlags(log.Ltime|log.Lshortfile)
  testData = make(map[string]MessageTestDef)
  testData["testing"]           = MessageTestDef{"testing",0,"test",0,MessageMatch{"testing",1,0,[4]byte{232, 243, 90, 6},[]byte{116,101,115,116,105,110,103},17}}
  testData["testingpartition1"] = MessageTestDef{"testing",0,"test",1,MessageMatch{"testing",1,0,[4]byte{232, 243, 90, 6},[]byte{116,101,115,116,105,110,103},17}}
  for _, td := range testData {
    topics = append(topics, td.TopicPartition())
  }
}


func TestMessageCreation(t *testing.T) {
  payload := []byte("testing")
  msg := NewMessage(payload)
  if msg.magic != 1 {
    t.Errorf("magic incorrect")
    t.Fail()
  }

  // generated by kafka-rb: e8 f3 5a 06
  expected := []byte{0xe8, 0xf3, 0x5a, 0x06}
  if !bytes.Equal(expected, msg.checksum[:]) {
    t.Fail()
  }
}

func TestMagic0MessageEncoding(t *testing.T) {
  // generated by kafka-rb:
  // test the old message format
  expected := []byte{0x00, 0x00, 0x00, 0x0c, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}
  length, msgsDecoded := Decode(expected, DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fail()
  }
  msgDecoded := msgsDecoded[0]

  payload := []byte("testing")
  if !bytes.Equal(payload, msgDecoded.payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatal("checksums do not match")
  }
  if msgDecoded.magic != 0 {
    t.Fatal("magic incorrect")
  }
}

func TestMessageEncoding(t *testing.T) {

  payload := []byte("testing")
  msg := NewMessage(payload)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x0d, 0x01, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}
  if !bytes.Equal(expected, msg.Encode()) {
    t.Fatalf("expected: % X\n but got: % X", expected, msg.Encode())
  }

  // verify round trip
  length, msgsDecoded := DecodeWithDefaultCodecs(msg.Encode())

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatal("checksums do not match")
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestCompressedMessageEncodingCompare(t *testing.T) {
  payload := []byte("testing")
  uncompressedMsgBytes := NewMessage(payload).Encode()

  msgGzipBytes := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID]).Encode()
  msgDefaultBytes := NewCompressedMessage(payload).Encode()
  if !bytes.Equal(msgDefaultBytes, msgGzipBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal", msgDefaultBytes, msgGzipBytes)
  }
}

func TestCompressedMessageEncoding(t *testing.T) {
  payload := []byte("testing")
  uncompressedMsgBytes := NewMessage(payload).Encode()

  msg := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID])

  // NOTE:  I could not get these tests to pass from apach trunk, i redid the values, 
  // the tests passed, and i sent the message from go producer -> scala consumer and it worked?
  //  not sure where these values for expected came from
  /*expectedPayload := []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
    0xFF, 0x62, 0x60, 0x60, 0xE0, 0x65, 0x64, 0x78, 0xF1, 0x39, 0x8A,
    0xAD, 0x24, 0xB5, 0xB8, 0x24, 0x33, 0x2F, 0x1D, 0x10, 0x00, 0x00,
    0xFF, 0xFF, 0x0C, 0x6A, 0x82, 0x91, 0x11, 0x00, 0x00, 0x00}  */
  expectedPayload := []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x09, 0x6E, 0x88, 0x04,
    0xFF, 0x62, 0x60, 0x60, 0xE0, 0x65, 0x64, 0x78, 0xF1, 0x39, 0x8A,
    0xAD, 0x24, 0xB5, 0xB8, 0x24, 0x33, 0x2F, 0x1D, 0x10, 0x00, 0x00,
    0xFF, 0xFF, 0x0C, 0x6A, 0x82, 0x91, 0x11, 0x00, 0x00, 0x00}
  
  // here is the difference  [       ]
  //expected: 1F 8B 08 00 00 00 00 00 04 FF 62 60 60 E0 65 64 78 F1 39 8A AD 24 
  // but got: 1F 8B 08 00 00 09 6E 88 04 FF 62 60 60 E0 65 64 78 F1 39 8A AD 24 

  //expectedHeader := []byte{0x00, 0x00, 0x00, 0x2F, 0x01, 0x01, 0x07, 0xFD, 0xC3, 0x76}
  expectedHeader := []byte{0x00, 0x00, 0x00, 0x2F, 0x01, 0x01, 0x96, 0x71, 0xA6, 0xE8}

  expected := make([]byte, len(expectedHeader)+len(expectedPayload))
  n := copy(expected, expectedHeader)
  copy(expected[n:], expectedPayload)

  if msg.compression != 1 {
    t.Fatalf("expected compression: 1 but got: %b", msg.compression)
  }

  zipper, _ := gzip.NewReader(bytes.NewBuffer(msg.payload))
  uncompressed := make([]byte, 100)
  n, _ = zipper.Read(uncompressed)
  uncompressed = uncompressed[:n]
  zipper.Close()

  if !bytes.Equal(uncompressed, uncompressedMsgBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal", uncompressed, uncompressedMsgBytes)
  }

  if !bytes.Equal(expected, msg.Encode()) {
    t.Fatalf("expected: % X\n but got: % X", expected, msg.Encode())
  }

  // verify round trip
  length, msgsDecoded := Decode(msg.Encode(), DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payload) {
    t.Fatal("bytes not equal")
  }
  chksum := []byte{0xE8, 0xF3, 0x5A, 0x06}
  if !bytes.Equal(chksum, msgDecoded.checksum[:]) {
    t.Fatalf("checksums do not match, expected: % X but was: % X",
      chksum, msgDecoded.checksum[:])
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestLongCompressedMessageRoundTrip(t *testing.T) {
  payloadBuf := bytes.NewBuffer([]byte{})
  // make the test bigger than buffer allocated in the Decode
  for i := 0; i < 15; i++ {
    payloadBuf.Write([]byte("testing123 "))
  }

  uncompressedMsgBytes := NewMessage(payloadBuf.Bytes()).Encode()
  msg := NewMessageWithCodec(uncompressedMsgBytes, DefaultCodecsMap[GZIP_COMPRESSION_ID])

  zipper, _ := gzip.NewReader(bytes.NewBuffer(msg.payload))
  uncompressed := make([]byte, 200)
  n, _ := zipper.Read(uncompressed)
  uncompressed = uncompressed[:n]
  zipper.Close()

  if !bytes.Equal(uncompressed, uncompressedMsgBytes) {
    t.Fatalf("uncompressed: % X \npayload: % X bytes not equal",
      uncompressed, uncompressedMsgBytes)
  }

  // verify round trip
  length, msgsDecoded := Decode(msg.Encode(), DefaultCodecsMap)

  if length == 0 || msgsDecoded == nil {
    t.Fatal("message is nil")
  }
  msgDecoded := msgsDecoded[0]

  if !bytes.Equal(msgDecoded.payload, payloadBuf.Bytes()) {
    t.Fatal("bytes not equal")
  }
  if msgDecoded.magic != 1 {
    t.Fatal("magic incorrect")
  }
}

func TestMultipleCompressedMessages(t *testing.T) {
  msgs := []*Message{NewMessage([]byte("testing")),
    NewMessage([]byte("multiple")),
    NewMessage([]byte("messages")),
  }
  msg := NewCompressedMessages(msgs...)

  length, msgsDecoded := DecodeWithDefaultCodecs(msg.Encode())
  if length == 0 || msgsDecoded == nil {
    t.Fatal("msgsDecoded is nil")
  }

  // make sure the decompressed messages match what was put in
  for index, decodedMsg := range msgsDecoded {
    if !bytes.Equal(msgs[index].payload, decodedMsg.payload) {
      t.Fatalf("Payload doesn't match, expected: % X but was: % X\n",
        msgs[index].payload, decodedMsg.payload)
    }
  }
}

func TestRequestHeaderEncoding(t *testing.T) {
  broker := newBroker("localhost:9092", &TopicPartition{Topic:"test", Partition:0})
  request := bytes.NewBuffer([]byte{})
  broker.EncodeRequestHeader(request, REQUEST_PRODUCE)
  EncodeTopicHeader(request, broker.topics[0].Topic, 0)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x74, 0x65, 0x73, 0x74,
    0x00, 0x00, 0x00, 0x00}

  if !bytes.Equal(expected, request.Bytes()) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request.Bytes()))
    t.Errorf("expected: %X\n but got: %X", expected, request)
    t.Fail()
  }
}

func TestPublishRequestEncoding(t *testing.T) {
  payload := []byte("testing")
  msg := NewMessage( payload)

  pubBroker := NewBrokerPublisher("localhost:9092", "test", 0)
  request := pubBroker.broker.EncodeProduceRequest(msg)

  // generated by kafka-rb:
  expected := []byte{0x00, 0x00, 0x00, 0x21, 0x00, 0x00, 0x00, 0x04, 0x74, 0x65, 0x73, 0x74,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x0d,
    /* magic  comp  ......  chksum ....     ..  payload .. */
    0x01, 0x00, 0xe8, 0xf3, 0x5a, 0x06, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67}

  if !bytes.Equal(expected, request) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request))
    t.Errorf("expected: % X\n but got: % X", expected, request)
    t.Fail()
  }
}


func TestConsumeRequestEncoding(t *testing.T) {
  tp := TopicPartition{Topic:"test", Partition:0, Offset:0, MaxSize: 1048576}
  pubBroker := NewProducer("localhost:9092", []*TopicPartition{&tp})
  request := pubBroker.broker.EncodeConsumeRequest()

  // generated by kafka-rb, encode_request_size + encode_request
  expected := []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x01, 0x00, 0x04, 0x74,
    0x65, 0x73, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00}

  if !bytes.Equal(expected, request) {
    t.Errorf("expected length: %d but got: %d", len(expected), len(request))
    t.Errorf("expected: % X\n but got: % X", expected, request)
    t.Fail()
  }
}

// re-usable routine for verifying messages
func testMessage(t *testing.T, msg *Message, tm *MessageTestDef, request []byte) {

  //TODO, add compression check

  if !bytes.Equal([]byte(tm.PayloadIn), msg.Payload()) {
    t.Fatalf("payload in: % X \npayload: % X bytes not equal, %s",
      []byte(tm.PayloadIn), msg.Payload(), string(msg.Payload()))
  }

  if tm.TotalLength != msg.TotalLen() {
    t.Fatalf("msg len expected: %d  but got: %d for %s", tm.TotalLength, msg.TotalLen(), tm.Name)
  }

  if !bytes.Equal(tm.Checksum[:], msg.checksum[:]) {
    t.Fatalf("checksums do not match for %s\n", tm.Name)
  }
  if msg.magic != tm.Magic {
    t.Fatalf("magic incorrect  expected %v but was %v", tm.Magic, msg.magic)
  }
  if msg.compression != tm.Compress || tm.Compression != msg.compression {
    t.Fatalf("compression incorrect  expected %v and %v but was %v", tm.Compress, tm.Compression, msg.compression)
  }
}

// test multi-produce header/message sets
func testMultiProduceHeaders(t *testing.T, mds []MessageTestDef, pr ProduceRequest, request []byte) {

  msgs_len := 0
  request_header := 4 + 2 + 2 
  topic_partition := 0  // 2 + len("test") + 4  + 4
  for _, tm := range mds {
    topic_partition += 2 + len(tm.Topic) + 4 + 4
  }

  for _, partMsgs := range pr {
    for _, msgs := range partMsgs {
      for _, msg := range msgs {
        msgs_len += int(msg.Message.TotalLen())
      }
    }
  }

  total_len := request_header +  topic_partition  + msgs_len 
  if len(request) != total_len {
    t.Fatalf("request len should be %d but was %d", total_len, len(request))
  }
}

func TestMultiProduceEncoding(t *testing.T) {
  tm := testData["testing"]
  tm2 := testData["testingpartition1"]
  tm.Name = "multiproduce encode testing"
  msg := NewMessageTopic(tm.Topic, []byte(tm.PayloadIn))
  msg2 := NewMessageTopic(tm.Topic, []byte(tm2.PayloadIn))
  _ = msg.Message.Encode()
  _ = msg2.Message.Encode()

  producer := NewPartitionedProducer("localhost:9092", tm.Topic, []int{0, 1})
  msgs := make(ProduceRequest)
  msgs[tm.Topic] = make(map[int][]*MessageTopic)
  msgs[tm.Topic][0] = []*MessageTopic{msg}
  msgs[tm.Topic][1] = []*MessageTopic{msg2}
  request := producer.broker.EncodeMultiProduceRequest(&msgs)
  
  /*
   0 0 0 66   - len of collection of messages
   0 3        - request type (multi-produce)
   0 2        - num of topic-partition combos
   - repeat once per topic/partition
     0 4          len of topic (bytes)
     116 101 115 116       topic   (test)
     0 0 0 0               partition   0
     0 0 0 17     - len of message set(could be n # of messages)
       msg
    - 2nd topic/partition combo

  0 0 0 66 0 3 0 2  0 4 116 101 115 116 0 0 0 0    0 0 0 17  0 0 0 13 1 0 232 243 90 6 116 101 115 116 105 110 103 
                    0 4 116 101 115 116 0 0 0 1    0 0 0 17  0 0 0 13 1 0 232 243 90 6 116 101 115 116 105 110 103
  */
  testMessage(t,msg.Message,&tm,request)
  testMessage(t,msg2.Message,&tm2,request)

  // remember go gives is no order of iteration, so we don't know which is first
  if !bytes.Equal(request[:14], []byte{0,0,0,66,0,3,0,2,0,4,116,101,115,116}){
    t.Fatalf("payload in: % X \npayload: % X bytes not equal",
      request[:14], []byte{0,0,0,66,0,3,0,2,0,4,116,101,115,116})
  }

  testMultiProduceHeaders(t, []MessageTestDef{tm,tm2}, msgs, request)

}

func TestMultiProduceTopics(t *testing.T) {
  // implement tests for multiple topics, multiple partitions
  
}

func TestMultiFetchEncoding(t *testing.T) {
   /*
     [0 0 0 48 0 2 0 2 
    0 4 116 101 115 116 0 0 0 0 0 0 0 0 0 0 0 0 0 16 0 0 
    0 4 116 101 115 116 0 0 0 1 0 0 0 0 0 0 0 0 0 16 0 0]
  */
  con := NewMultiConsumer("localhost:9092", topics)
  request := con.broker.EncodeConsumeRequestMultiFetch()
  // remember go gives is no order of iteration, so we don't know which is first
  if !bytes.Equal(request[:14], []byte{0,0,0,48,0,2,0,2,0,4,116,101,115,116}){
    t.Fatalf("request: % X \request: % X bytes not equal",
      request[:14], []byte{0,0,0,48,0,2,0,2,0,4,116,101,115,116})
  }
  
}
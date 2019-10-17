package screencapture_test

import (
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/packet"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
)

type UsbTestDummy struct {
	dataReceiver        chan [] byte
	cmSampleBufConsumer chan coremedia.CMSampleBuffer
}

func (u UsbTestDummy) Consume(buf coremedia.CMSampleBuffer) error {
	u.cmSampleBufConsumer <- buf
	return nil
}

func (u UsbTestDummy) WriteDataToUsb(data []byte) {
	u.dataReceiver <- data
}

func TestMessageProcessorStopsOnUnknownPacket(t *testing.T) {
	usbDummy := UsbTestDummy{}
	stopChannel := make(chan interface{})
	mp := screencapture.NewMessageProcessor(usbDummy, stopChannel, usbDummy)
	go func() { mp.ReceiveData(make([]byte, 4)) }()
	<-stopChannel
}

type syncTestCase struct {
	receivedData  []byte
	expectedReply []byte
	descrription  string
}

func TestMessageProcessorRespondsCorrectlyToSyncMessages(t *testing.T) {
	cases := []syncTestCase{
		{
			receivedData:  packet.NewPingPacketAsBytes()[4:],
			expectedReply: packet.NewPingPacketAsBytes(),
			descrription:  "Expect Ping as a response to a ping packet",
		},
		{
			receivedData:  loadFromFile("afmt-request"),
			expectedReply: loadFromFile("afmt-reply"),
			descrription:  "Expect correct reply for afmt",
		},
	}

	usbDummy := UsbTestDummy{dataReceiver: make(chan []byte)}
	stopChannel := make(chan interface{})
	mp := screencapture.NewMessageProcessor(usbDummy, stopChannel, usbDummy)
	for _, testCase := range cases {
		go func() { mp.ReceiveData(testCase.receivedData) }()
		response := <-usbDummy.dataReceiver
		assert.Equal(t, testCase.expectedReply, response, testCase.descrription)
	}

}

func TestMessageProcessorForwardsFeed(t *testing.T) {
	dat, err := ioutil.ReadFile("packet/fixtures/asyn-feed")
	if err != nil {
		log.Fatal(err)
	}

	usbDummy := UsbTestDummy{dataReceiver: make(chan []byte), cmSampleBufConsumer: make(chan coremedia.CMSampleBuffer)}
	stopChannel := make(chan interface{})
	mp := screencapture.NewMessageProcessor(usbDummy, stopChannel, usbDummy)
	go func() { mp.ReceiveData(dat[4:]) }()
	response := <-usbDummy.cmSampleBufConsumer
	expected := "{OutputPresentationTS:CMTime{95911997690984/1000000000, flags:KCMTimeFlagsHasBeenRounded, epoch:0}, NumSamples:1, Nalus:[{len:30 type:SEI},{len:90712 type:IDR},], fdsc:fdsc:{MediaType:Video, VideoDimension:(1126x2436), Codec:AVC-1, PPS:27640033ac5680470133e69e6e04040404, SPS:28ee3cb0, Extensions:IndexKeyDict:[{49 : IndexKeyDict:[{105 : 0x01640033ffe1001127640033ac5680470133e69e6e0404040401000428ee3cb0fdf8f800},]},{52 : H.264},]}, attach:IndexKeyDict:[{28 : IndexKeyDict:[{46 : Float64[2436.000000]},{47 : Float64[2436.000000]},]},{29 : Int32[0]},{26 : IndexKeyDict:[{46 : Float64[1126.000000]},{47 : Float64[2436.000000]},{45 : Float64[0.000000]},{44 : Float64[0.000000]},]},{27 : IndexKeyDict:[{46 : Float64[1126.000000]},{47 : Float64[2436.000000]},{45 : Float64[0.000000]},{44 : Float64[0.000000]},]},], sary:IndexKeyDict:[{4 : %!s(bool=false)},], SampleTimingInfoArray:{Duration:CMTime{1/60, flags:KCMTimeFlagsHasBeenRounded, epoch:0}, PresentationTS:CMTime{95911997690984/1000000000, flags:KCMTimeFlagsHasBeenRounded, epoch:0}, DecodeTS:CMTime{0/0, flags:KCMTimeFlagsValid, epoch:0}}}"

	assert.Equal(t, expected, response.String())
}

func loadFromFile(name string) []byte {
	dat, err := ioutil.ReadFile("packet/fixtures/" + name)
	if err != nil {
		log.Fatal(err)
	}
	return dat[4:]
}

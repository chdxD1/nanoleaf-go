package nanoleaf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

var (
	VersionV1 = "v1"
	VersionV2 = "v2"
)

// NanoStream udp connection to nanoleaf
type NanoStream struct {
	nano      *Nanoleaf
	con       net.Conn
	connected bool
	address   string
	port      int
	version   string
}

// FrameEffect describes a frame for a panel
type FrameEffect struct {
	Red        int `json:"red"`
	Green      int `json:"green"`
	Blue       int `json:"blue"`
	White      int `json:"white"`
	Transition int `json:"transition"`
}

// PanelEffect describes effect for a specific panel
type PanelEffect struct {
	ID    int         `json:"id"`
	Frame FrameEffect `json:"frame"`
}

// StreamEffect will write panel effects
type StreamEffect struct {
	Panels []PanelEffect `json:"panels"`
}

// newNanoStream returns a new instance of NanoStream
func newNanoStream(nano *Nanoleaf) *NanoStream {
	return &NanoStream{
		nano:      nano,
		connected: false,
	}
}

// WriteEffect writes effect to nanoleaf
func (s *NanoStream) WriteEffect(effect StreamEffect) error {
	if len(effect.Panels) == 0 {
		return nil
	}

	buf := new(bytes.Buffer)

	if s.version == VersionV1 {
		binary.Write(buf, binary.LittleEndian, uint8(len(effect.Panels)))

		for _, panel := range effect.Panels {
			nFrames := 1
			binary.Write(buf, binary.LittleEndian, uint8(panel.ID))
			binary.Write(buf, binary.LittleEndian, nFrames)

			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Red))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Green))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Blue))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.White))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Transition))
		}
	} else if s.version == VersionV2 {
		binary.Write(buf, binary.LittleEndian, uint16(len(effect.Panels)))

		for _, panel := range effect.Panels {
			binary.Write(buf, binary.LittleEndian, uint16(panel.ID))

			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Red))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Green))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.Blue))
			binary.Write(buf, binary.LittleEndian, uint8(panel.Frame.White))
			binary.Write(buf, binary.LittleEndian, uint16(panel.Frame.Transition))
		}
	}

	if _, err := s.con.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// Activate activates extControl to allow creating a udp connection
func (s *NanoStream) Activate(version string) error {
	if version != VersionV1 && version != VersionV2 {
		return ErrInvalidVersion
	}

	s.version = version

	body := jsonPayload{
		"write": jsonPayload{
			"command":           "display",
			"animType":          "extControl",
			"extControlVersion": version,
		},
	}

	url := fmt.Sprintf("%s/%s/effects", s.nano.url, s.nano.token)
	resp, err := s.nano.client.R().SetHeader("Content-Type", "application/json").SetBody(body).Put(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if resp.StatusCode() != http.StatusOK {
		return ErrUnexpectedResponse
	}

	var jsonResponse struct {
		Address string `json:"streamControlIpAddr"`
		Port    int    `json:"streamControlPort"`
	}

	if err := json.Unmarshal(resp.Body(), &jsonResponse); err != nil {
		return ErrParsingJSON
	}

	s.address = jsonResponse.Address
	s.port = jsonResponse.Port

	return nil
}

// Connect connects to nanoleaf via udp
func (s *NanoStream) Connect() error {
	con, err := net.Dial("udp", fmt.Sprintf("%s:%d", s.address, s.port))

	if err != nil {
		return err
	}

	s.con = con
	s.connected = true
	return nil
}

// Disconnect closes udp connection
func (s *NanoStream) Disconnect() error {
	err := s.con.Close()

	if err != nil {
		return err
	}

	s.connected = false
	return nil
}

// IsConnected checks if there is a connection
func (s *NanoStream) IsConnected() bool {
	return s.connected
}

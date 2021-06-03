package client

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/containerd/console"
	"github.com/creack/goselect"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

const (
	committedSuicide = iota
	killed
)


type Client struct {
	Dialer          *websocket.Dialer
	Conn            *websocket.Conn
	URL             string
	token           string
	Connected       bool
	Output          io.Writer
	WriteMutex      *sync.Mutex
	EscapeKeys      []byte
	poison          chan bool
	StreamDataSequenceNumber int64
}

func NewClient(inputURL string, token string) (*Client, error) {
	return &Client{
		Dialer:     &websocket.Dialer{},
		URL:        inputURL,
		token:      token,
		WriteMutex: &sync.Mutex{},
		Output:     os.Stdout,
		StreamDataSequenceNumber: 0,
	}, nil
}

func (c *Client) write(data []byte) error {

	c.WriteMutex.Lock()
	defer c.WriteMutex.Unlock()
	return c.Conn.WriteMessage(websocket.BinaryMessage, data)
}

// Connect tries to dial a websocket server
func (c *Client) Connect() error {
	// Open WebSocket connection

	logrus.Debugln("Connecting to websocket: ", c.URL)
	header := http.Header{}
	header.Add("x-acs-session-token", c.token)
	conn, _, err := c.Dialer.Dial(c.URL, header)
	if err != nil {
		return err
	}
	c.Conn = conn
	c.Connected = true

	// Initialize message types for gotty
	// go c.pingLoop()

	return nil
}

func (c *Client) pingLoop() {
	for {
		if (c.Connected) {
			logrus.Debugf("Sending ping")
			err := c.Conn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				logrus.Warnf("c.write: %v", err)
			}
		}

		time.Sleep(30 * time.Second)
	}
}

var term_chan chan int
func (c *Client) Loop() error {

	if !c.Connected {
		err := c.Connect()
		if err != nil {
			return err
		}
	}
	term, err := console.ConsoleFromFile(os.Stdout)
	if err != nil {
		return fmt.Errorf("os.Stdout is not a valid terminal")
	}
	err = term.SetRaw()
	if err != nil {
		return fmt.Errorf("Error setting raw terminal: %v", err)
	}
	defer func() {
		_ = term.Reset()
	}()

	wg := &sync.WaitGroup{}
	term_chan = make(chan int)

	wg.Add(1)
	go c.termsizeLoop(wg)

	wg.Add(1)
	go c.readLoop(wg)

	wg.Add(1)
	go c.writeLoop(wg)

	/* Wait for all of the above goroutines to finish */
	//wg.Wait()
	<- term_chan

	logrus.Debug("Client.Loop() exiting")
	return nil
}

type winsize struct {
	Rows    uint16 `json:"rows"`
	Columns uint16 `json:"cols"`
}

func (c *Client) termsizeLoop(wg *sync.WaitGroup) int {
	defer wg.Done()
	fname := "termsizeLoop"

	ch := make(chan os.Signal, 1)
	notifySignalSIGWINCH(ch)
	defer resetSignalSIGWINCH()

	for {
		if b, err := syscallTIOCGWINSZ(); err != nil {
		//	logrus.Warn(err)
		} else {
			if err = c.SendResizeDataMessage(b); err != nil {
				log.GetLogger().Warnf("ws.WriteMessage failed: %v", err)
			}
		}
		select {
		case <-c.poison:
			/* Somebody poisoned the well; die */
			return die(fname, c.poison)
		case <-ch:
		}
	}
}

func (c *Client) readLoop(wg *sync.WaitGroup) int {
	defer wg.Done()
	fname := "readLoop"

	type MessageNonBlocking struct {
		Msg message.Message
		Err  error
	}
	msgChan := make(chan MessageNonBlocking)

	for {
		go func() {
			_, data, err := c.Conn.ReadMessage()
			if util.IsVerboseMode() {
				log.GetLogger().Infoln("read msg: ", string(data))
			}
			streamDataMessage := message.Message{}
			if err == nil {
				if err = streamDataMessage.Deserialize(data); err != nil {
					log.GetLogger().Errorf("Cannot deserialize raw message, err: %v.", err)
				}
			}

			if util.IsVerboseMode() {
				log.GetLogger().Infoln("read msg num : ", streamDataMessage.SequenceNumber)
			}

			msgChan <- MessageNonBlocking{Msg: streamDataMessage, Err: err}
			// time.Sleep(time.Second * 1)
			// msgChan <- MessageNonBlocking{Data:  []byte("c"), Err: nil}
		}()

		select {
		case <-c.poison:
			return die(fname, c.poison)
		case msg := <-msgChan:
			if msg.Err != nil {
                log.GetLogger().Errorln("read msg err", msg.Err)
				if _, ok := msg.Err.(*websocket.CloseError); !ok {
					log.GetLogger().Warnf("c.Conn.ReadMessage: %v", msg.Err)
				}
				return openPoison(fname, c.poison)
			}
			if msg.Msg.Validate() != nil {

				log.GetLogger().Errorln("An error has occured, msg is invalid")
				return openPoison(fname, c.poison)
			}

			switch msg.Msg.MessageType {
			case message.OutputStreamDataMessage: // data
			     c.Output.Write(msg.Msg.Payload)
			     break

			default:
				// logrus.Warnf("Unhandled protocol message")
			}
		}
	}
	return 0
}

type exposeFd interface {
	Fd() uintptr
}

func (c *Client) writeLoop(wg *sync.WaitGroup) int {
	defer wg.Done()
	fname := "writeLoop"

	buff := make([]byte, 128)

	rdfs := &goselect.FDSet{}
	reader := io.ReadCloser(os.Stdin)

	pr := NewEscapeProxy(reader, c.EscapeKeys)
	defer reader.Close()

	for {
		select {
		case <-c.poison:
			return die(fname, c.poison)
		default:
		}

		rdfs.Zero()
		rdfs.Set(reader.(exposeFd).Fd())
		err := goselect.Select(1, rdfs, nil, nil, 50*time.Millisecond)
		if err != nil {
			// log.GetLogger().Errorf("get raw input failed: %v", err)
			continue
			// return openPoison(fname, c.poison)
		}
		if rdfs.IsSet(reader.(exposeFd).Fd()) {
			size, err := pr.Read(buff)

			if err != nil {
				log.GetLogger().Infoln("err in input empty")
				if err == io.EOF {
					log.GetLogger().Infoln("EOF in input empty")
					// Send EOF to GoTTY

					// Send 'Input' marker, as defined in GoTTY::client_context.go,
					// followed by EOT (a translation of Ctrl-D for terminals)
					err = c.SendStreamDataMessage((append([]byte{}, byte(4))))

					if err != nil {
						return openPoison(fname, c.poison)
					}
					continue
				} else {
					log.GetLogger().Errorln("err in input empty", err)
					return openPoison(fname, c.poison)
				}
			}

			if size <= 0 {
				log.GetLogger().Infoln("user input empty")
				continue
			}

			data := buff[:size]
			if util.IsVerboseMode() {
				log.GetLogger().Infoln("begin send user input ", string(data))
			}
			err = c.SendStreamDataMessage(data)
			if err != nil {
				return openPoison(fname, c.poison)
			}
			log.GetLogger().Traceln("send data:", data)
		}
	}
    return 0
}

func (c *Client) SendStreamDataMessage(inputData []byte) (err error) {
	if len(inputData) == 0 {
		log.GetLogger().Debugf("Ignoring empty stream data payload.")
		return nil
	}

	agentMessage := &message.Message{
		MessageType:    message.InputStreamDataMessage,
		SchemaVersion:  1,
		CreatedDate:    uint64(time.Now().UnixNano() / 1000000),
		SequenceNumber: c.StreamDataSequenceNumber,
		PayloadLength:   uint32(len(inputData)),
		Payload:        inputData,
	}

	if util.IsVerboseMode() {
		log.GetLogger().Infoln("SendStreamDataMessage num: ", c.StreamDataSequenceNumber)
	}

	msg, err := agentMessage.Serialize()
	if err != nil {
		return fmt.Errorf("cannot serialize StreamData message %v", agentMessage)
	}

	if err = c.sendMessage(msg, websocket.BinaryMessage); err != nil {
		log.GetLogger().Errorf("Error sending stream data message %v", err)
		log.GetLogger().Infoln("disconnect, plugin exit")
		// os.Exit(1)
		c.Connected = false
		term_chan <- 1
	}

	if util.IsVerboseMode() {
		log.GetLogger().Println("SendStreamDataMessage:", msg)
	}

	c.StreamDataSequenceNumber = c.StreamDataSequenceNumber + 1
	return nil
}

func (c *Client) SendResizeDataMessage(inputData []byte) (err error) {
	if len(inputData) == 0 {
		log.GetLogger().Debugf("Ignoring empty stream data payload.")
		return nil
	}

	agentMessage := &message.Message{
		MessageType:    message.SetSizeDataMessage,
		SchemaVersion:  1,
		CreatedDate:    uint64(time.Now().UnixNano() / 1000000),
		SequenceNumber: c.StreamDataSequenceNumber,
		PayloadLength:   uint32(len(inputData)),
		Payload:        inputData,
	}
	msg, err := agentMessage.Serialize()
	if err != nil {
		log.GetLogger().Errorf("cannot serialize StreamData message %v", agentMessage)
		return fmt.Errorf("cannot serialize StreamData message %v", agentMessage)
	}

	if err = c.sendMessage(msg, websocket.BinaryMessage); err != nil {
		log.GetLogger().Errorf("Error sending stream data message %v", err)
		return err
	}

	c.StreamDataSequenceNumber = c.StreamDataSequenceNumber + 1
	return nil
}

func (c *Client) sendMessage(input []byte, inputType int) error {
	defer func() {
		if msg := recover(); msg != nil {
			log.GetLogger().Errorln("WebsocketChannel  run panic: %v", msg)
			log.GetLogger().Errorln("%s: %s", msg, debug.Stack())
		}
	}()

	if len(input) < 1 {
		log.GetLogger().Errorln("Can't send message: Empty input.")
		return errors.New("Can't send message: Empty input.")
	}

	c.WriteMutex.Lock()
	err := c.Conn.WriteMessage(inputType, input)
	if util.IsVerboseMode() {
		log.GetLogger().Infoln("begin send msg: ", string(input))
	}
	if err != nil {
		log.GetLogger().Errorf("send messagefaile, %v", err)
	}
	c.WriteMutex.Unlock()
	return err
}


func openPoison(fname string, poison chan bool) int {
	logrus.Debug(fname + " suicide")

	/*
	 * The close() may raise panic if multiple goroutines commit suicide at the
	 * same time. Prevent that panic from bubbling up.
	 */
	defer func() {
		if r := recover(); r != nil {
			logrus.Debug("Prevented panic() of simultaneous suicides", r)
		}
	}()

	/* Signal others to die */
	close(poison)
	return committedSuicide
}

func die(fname string, poison chan bool) int {
	logrus.Debug(fname + " died")

	wasOpen := <-poison
	if wasOpen {
		logrus.Error("ERROR: The channel was open when it wasn't supposed to be")
	}

	return killed
}
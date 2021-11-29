package esme

import (
	"fmt"
	"github.com/linxGnu/gosmpp"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
	"log"
	"strings"
	"sync"
	"time"
)

type Esme struct {
	auth     gosmpp.Auth
	session  *gosmpp.Session
	messages map[int32]*ShortMessage
}

var mutex = &sync.Mutex{}

func StartSession() (*Esme, error) {
	esme := new(Esme)
	esme.messages = make(map[int32]*ShortMessage)

	auth := gosmpp.Auth{
		SMSC:       "localhost:2775",
		SystemID:   "169994",
		Password:   "EDXPJU",
		SystemType: "",

		//SMSC:       "smscsim.melroselabs.com:2775",
		//SystemID:   "257601",
		//Password:   "71ece2",
	}

	session, err := gosmpp.NewSession(
		gosmpp.TRXConnector(
			gosmpp.NonTLSDialer,
			auth,
		),
		gosmpp.Settings{
			ReadTimeout: 61 * time.Second,
			EnquireLink: 60 * time.Second,
			OnPDU:       esme.handlePDU(),
			OnReceivingError: func(err error) {
				fmt.Println("Receiving PDU/Network error:", err)
			},

			OnSubmitError: func(_ pdu.PDU, err error) {
				log.Fatal("SubmitPDU error:", err)
			},
			OnRebindingError: func(err error) {
				fmt.Println("Rebinding but error:", err)
			},

			OnClosed: func(state gosmpp.State) {
				fmt.Println(state)
			},
		},
		5*time.Second,
	)
	if err != nil {
		panic(err)
	}

	esme.auth = auth
	esme.session = session

	log.Printf("Started ESME server")

	return esme, nil
}

func (e *Esme) handlePDU() func(pdu.PDU, bool) {
	concatenated := map[uint8][]string{}

	return func(p pdu.PDU, _ bool) {
		switch pd := p.(type) {
		case *pdu.SubmitSMResp:
			//fmt.Printf("SubmitSMResp:%+v\n", pd)
			log.Printf("SubmitSMResp:\n")

			log.Println("\t", pd.SequenceNumber)

			mutex.Lock()

			if v, ok := e.messages[pd.SequenceNumber]; ok {

				e.updateMessageBySequence(v.Sequence, "SUBMITTED")
				v.id = pd.MessageID

			}

			mutex.Unlock()

		case *pdu.GenericNack:
			log.Println("GenericNack Received")
		//
		case *pdu.EnquireLinkResp:
			log.Println("EnquireLinkResp Received")
		//
		case *pdu.DataSM:
			//fmt.Printf("DataSM:%+v\n", pd)
			fmt.Printf("DataSM:\n")

			//log.Println(pd.CommandID)

		//
		case *pdu.DeliverSM:
			//fmt.Printf("DeliverSM:%+v\n", pd)
			log.Printf("DeliverSM:\n")

			// region concatenated sms (sample code)
			message, err := pd.Message.GetMessage()
			if err != nil {
				log.Fatal(err)
			}

			totalParts, sequence, reference, found := pd.Message.UDH().GetConcatInfo()
			if found {
				if _, ok := concatenated[reference]; !ok {
					concatenated[reference] = make([]string, totalParts)
				}
				concatenated[reference][sequence-1] = message
			}
			if !found {
				//fmt.Println("\tNot Found: " , pd.SequenceNumber)
				log.Println("\t", pd.SequenceNumber)

				if strings.Contains(message, "id:") {
					fields := strings.Fields(message)

					id := strings.Split(fields[0], ":")[1]
					status := strings.Split(fields[7], ":")[1]

					e.updateMessageById(id, status)
				}

			} else if parts, ok := concatenated[reference]; ok && isConcatenatedDone(parts, totalParts) {
				//log.Println(strings.Join(parts, "-"))
				delete(concatenated, reference)
			}
			// endregion

			//default:
			//	fmt.Println("DEFAULT:\n\t")
			//	//fmt.Println(pd)
		}
	}
}

func (e *Esme) updateMessageBySequence(sequence int32, status string) {
	//TODO: Would it be faster/efficient to use ?
	if v, ok := e.messages[sequence]; ok {
		// Found
		v.Status = status
	}

	//for i := range e.messages {
	//	if e.messages[i].Sequence == sequence {
	//		// Found!
	//		e.messages[i].Status = status
	//		break
	//	}
	//}
}

func (e *Esme) updateMessageById(id string, status string) {
	for i := range e.messages {
		if e.messages[i].id == id {
			// Found!
			mutex.Lock()
			e.messages[i].Status = status
			mutex.Unlock()
			break
		}
	}
}

func (e *Esme) FindMessageBySequence(sequence int32) *ShortMessage {
	if v, ok := e.messages[sequence]; ok {
		// Found
		return v
	}
	return nil
}

func isConcatenatedDone(parts []string, total byte) bool {
	for _, part := range parts {
		if part != "" {
			total--
		}
	}
	return total == 0
}

type ShortMessage struct {
	id       string
	Sequence int32  `json:"id"`
	Message  string `json:"message"`
	Status   string `json:"status"`
}

func (e *Esme) SendSM(message *ShortMessage) {
	newSM := e.newSubmitSM(message)
	transceiver := e.session.Transceiver()

	if err := transceiver.Submit(newSM); err != nil {
		fmt.Println("SendSM: ", err)
	}

	message.Status = "PENDING"

	mutex.Lock()
	e.messages[message.Sequence] = message
	mutex.Unlock()
}

func (e Esme) GetMessages() map[int32]*ShortMessage {
	return e.messages
}

func (e Esme) newSubmitSM(message *ShortMessage) *pdu.SubmitSM {
	// build up submitSM
	srcAddr := pdu.NewAddress()
	srcAddr.SetTon(5)
	srcAddr.SetNpi(0)
	_ = srcAddr.SetAddress("00" + e.auth.SystemID)

	destAddr := pdu.NewAddress()
	destAddr.SetTon(1)
	destAddr.SetNpi(1)
	_ = destAddr.SetAddress("99" + e.auth.SystemID)

	submitSM := pdu.NewSubmitSM().(*pdu.SubmitSM)
	submitSM.SourceAddr = srcAddr
	submitSM.DestAddr = destAddr
	_ = submitSM.Message.SetMessageWithEncoding(message.Message, data.UCS2)
	submitSM.ProtocolID = 0
	submitSM.RegisteredDelivery = 1
	submitSM.ReplaceIfPresentFlag = 0
	submitSM.EsmClass = 0

	mutex.Lock()

	message.Sequence = submitSM.SequenceNumber

	mutex.Unlock()

	return submitSM
}

func (e *Esme) Close() {
	defer func() {
		_ = e.session.Close()
	}()

	log.Print("Stopped ESME Server")
}

package kdcproxy

import (
	"encoding/binary"
	"fmt"
	krbconfig "github.com/bolkedebruin/gokrb5/v8/config"
	"github.com/jcmturner/gofork/encoding/asn1"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	maxLength        = 128 * 1024
	systemConfigPath = "/etc/krb5.conf"
	timeout          = 5 * time.Second
)

type KdcProxyMsg struct {
	Message []byte `asn1:"tag:0,explicit"`
	Realm   string `asn1:"tag:1,optional"`
	Flags   int    `asn1:"tag:2,optional"`
}

type Kdc struct {
	Realm string
	Host  string
	Proto string
	Conn  net.Conn
}

type KerberosProxy struct {
	krb5Config *krbconfig.Config
}

func InitKdcProxy(krb5Conf string) KerberosProxy {
	path := systemConfigPath
	if krb5Conf != "" {
		path = krb5Conf
	}
	cfg, err := krbconfig.Load(path)
	if err != nil {
		log.Fatalf("Cannot load krb5 config %s due to %s", path, err)
	}

	return KerberosProxy{
		krb5Config: cfg,
	}
}

func (k KerberosProxy) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	length := r.ContentLength
	if length == -1 {
		http.Error(w, "Content length required", http.StatusLengthRequired)
		return
	}

	if length > maxLength {
		http.Error(w, "Request entity too large", http.StatusRequestEntityTooLarge)
		return
	}

	data := make([]byte, length)
	_, err := io.ReadFull(r.Body, data)
	if err != nil {
		log.Printf("Error reading from stream: %s", err)
		http.Error(w, "Error reading from stream", http.StatusInternalServerError)
		return
	}

	msg, err := decode(data)
	if err != nil {
		log.Printf("Cannot unmarshal: %s", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	krb5resp, err := k.forward(msg.Realm, msg.Message)
	if err != nil {
		log.Printf("cannot forward to kdc due to %s", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	reply, err := encode(krb5resp)
	if err != nil {
		log.Printf("unable to encode krb5 message due to %s", err)
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/kerberos")
	w.Write(reply)
}

func (k *KerberosProxy) forward(realm string, data []byte) (resp []byte, err error) {
	if realm == "" {
		realm = k.krb5Config.LibDefaults.DefaultRealm
	}

	// load udp first as is the default for kerberos
	udpCnt, udpKdcs, err := k.krb5Config.GetKDCs(realm, false)
	if err != nil {
		return nil, fmt.Errorf("cannot get udp kdc for realm %s due to %s", realm, err)
	}

	// load tcp
	tcpCnt, tcpKdcs, err := k.krb5Config.GetKDCs(realm, true)
	if err != nil {
		return nil, fmt.Errorf("cannot get tcp kdc for realm %s due to %s", realm, err)
	}

	if tcpCnt+udpCnt == 0 {
		return nil, fmt.Errorf("cannot get any kdcs (tcp or udp) for realm %s", realm)
	}

	// merge the kdcs
	kdcs := make([]Kdc, 0, tcpCnt+udpCnt)
	for i := 0; i <= udpCnt; i++ {
		if host, ok := udpKdcs[i]; ok {
			kdcs = append(kdcs, Kdc{Realm: realm, Host: host, Proto: "udp"})
		}
	}
	for i := 0; i <= tcpCnt; i++ {
		if host, ok := tcpKdcs[i]; ok {
			kdcs = append(kdcs, Kdc{Realm: realm, Host: host, Proto: "tcp"})
		}
	}
	if len(kdcs) == 0 {
		return nil, fmt.Errorf("cannot get any kdcs (tcp or udp) for realm %s", realm)
	}

	replies := make(chan []byte, len(kdcs))
	startedConns := make([]net.Conn, 0, len(kdcs))
	var lastErr error
	for i := range kdcs {
		conn, err := net.DialTimeout(kdcs[i].Proto, kdcs[i].Host, timeout)

		if err != nil {
			lastErr = err
			log.Printf("error connecting to %s due to %s, trying next if available", kdcs[i], err)
			continue
		}
		conn.SetDeadline(time.Now().Add(timeout))

		// if we proxy over UDP remove the length prefix
		if kdcs[i].Proto == "tcp" {
			_, err = conn.Write(data)
		} else {
			if len(data) < 4 {
				lastErr = fmt.Errorf("udp kdc packet too short: %d bytes", len(data))
				log.Printf("cannot write packet data to %s due to %s, trying next if available", kdcs[i], lastErr)
				conn.Close()
				continue
			}
			_, err = conn.Write(data[4:])
		}
		if err != nil {
			lastErr = err
			log.Printf("cannot write packet data to %s due to %s, trying next if available", kdcs[i], err)
			conn.Close()
			continue
		}

		kdcs[i].Conn = conn
		startedConns = append(startedConns, conn)
		go awaitReply(conn, kdcs[i].Proto == "udp", replies)
	}

	for _, conn := range startedConns {
		defer conn.Close()
	}

	if len(startedConns) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("no kdc connections started for realm %s: %w", realm, lastErr)
		}
		return nil, fmt.Errorf("no kdc connections started for realm %s", realm)
	}

	for range startedConns {
		reply := <-replies
		if reply != nil {
			return reply, nil
		}
	}

	return nil, fmt.Errorf("no replies received from kdcs for realm %s", realm)
}

func decode(data []byte) (msg *KdcProxyMsg, err error) {
	var m KdcProxyMsg
	rest, err := asn1.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	if len(rest) > 0 {
		return nil, fmt.Errorf("trailing data in request")
	}

	return &m, nil
}

func encode(krb5data []byte) (r []byte, err error) {
	m := KdcProxyMsg{Message: krb5data}
	enc, err := asn1.Marshal(m)
	if err != nil {
		log.Printf("cannot marshal due to %s", err)
		return nil, err
	}
	return enc, nil
}

func awaitReply(conn net.Conn, isUdp bool, reply chan<- []byte) {
	if isUdp {
		buf := make([]byte, maxLength)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("error reading from kdc due to %s", err)
			reply <- nil
			return
		}

		// udp will be missing the length prefix so add it
		resp := make([]byte, n+4)
		binary.BigEndian.PutUint32(resp[:4], uint32(n))
		copy(resp[4:], buf[:n])
		reply <- resp
		return
	}

	lengthPrefix := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthPrefix); err != nil {
		log.Printf("error reading from kdc due to %s", err)
		reply <- nil
		return
	}

	length := binary.BigEndian.Uint32(lengthPrefix)
	if length > maxLength {
		log.Printf("error reading from kdc due to response length %d exceeding max length", length)
		reply <- nil
		return
	}

	resp := make([]byte, int(length)+4)
	copy(resp[:4], lengthPrefix)
	if _, err := io.ReadFull(conn, resp[4:]); err != nil {
		log.Printf("error reading from kdc due to %s", err)
		reply <- nil
		return
	}
	reply <- resp
}
